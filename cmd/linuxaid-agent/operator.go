package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

const saDir = "/var/run/secrets/kubernetes.io/serviceaccount"

const (
	defaultReconcileInterval     = 4 * time.Hour
	defaultTTLSeconds            = 600
	defaultActiveDeadlineSeconds = 1200
)

var (
	opImage           string
	opImagePullPolicy string
	opCertname        string
	opEnforce         bool
	opOpenvoxEnv      string
	opPuppetServer    string
	opSecretName      string
	opHostPath        string
	opNodeSelector    string
	opNamePrefix      string
	opNamespace       string
	opTTLSeconds      int
	opActiveDeadline  int
	opBackoffLimit    int
	opInterval        time.Duration
	opNode            string
	opControlRepoURL  string
	opControlRepoRef  string
	opHieraConfigMap  string
	opGitSecretName   string
)

var operatorCmd = &cobra.Command{
	Use:   "operator",
	Short: "Reconcile one agent Job per node as a daemon, or once for a single --node",
	RunE: func(_ *cobra.Command, _ []string) error {
		return runOperator()
	},
}

func init() {
	f := operatorCmd.Flags()
	f.StringVar(&opImage, "image", "", "agent container image (required)")
	f.StringVar(&opImagePullPolicy, "image-pull-policy", "IfNotPresent", "imagePullPolicy for the agent Jobs")
	f.StringVar(&opCertname, "certname", "", "obmondo-clientcert CN, shared by all nodes (required)")
	f.BoolVar(&opEnforce, "enforce", false, "apply changes (puppet --no-noop) instead of report-only")
	f.StringVar(&opOpenvoxEnv, "openvox-environment", "master", "LinuxAid/OpenVox environment")
	f.StringVar(&opPuppetServer, "puppet-server", "", "optional puppetserver override")
	f.StringVar(&opSecretName, "cert-secret", "obmondo-clientcert", "secret holding the pre-signed client cert")
	f.StringVar(&opHostPath, "host-path", "/opt/obmondo", "host path for staging the CLI binary")
	f.StringVar(&opNodeSelector, "node-selector", "", "label selector to filter nodes (empty = all)")
	f.StringVar(&opNamePrefix, "name-prefix", "linuxaid-agents", "prefix for the per-node Job names")
	f.StringVar(&opNamespace, "namespace", "", "namespace to create Jobs in (default: the pod's namespace)")
	f.IntVar(&opTTLSeconds, "ttl-seconds", defaultTTLSeconds, "ttlSecondsAfterFinished on the agent Jobs")
	f.IntVar(&opActiveDeadline, "active-deadline-seconds", defaultActiveDeadlineSeconds, "activeDeadlineSeconds on the agent Jobs")
	f.IntVar(&opBackoffLimit, "backoff-limit", 1, "backoffLimit on the agent Jobs")
	f.DurationVar(&opInterval, "interval", defaultReconcileInterval, "reconcile interval in daemon mode")
	f.StringVar(&opNode, "node", "", "create a Job for a single node and exit (skips if one already exists)")
	f.StringVar(&opControlRepoURL, "control-repo-url", "", "git URL of the puppet control-repo; when set the agent Jobs run masterless puppet apply")
	f.StringVar(&opControlRepoRef, "control-repo-ref", "", "tag (or ref) of the control-repo to checkout, from the chart's linuxaid.tag (default: latest tag)")
	f.StringVar(&opHieraConfigMap, "hiera-configmap", "", "ConfigMap with the rendered Helm-values hiera data, mounted into the agent Jobs at /hiera-data")
	f.StringVar(&opGitSecretName, "git-secret", "", "secret with git credentials (ssh-privatekey or token key) mounted into the agent Jobs at /git-credentials")
	rootCmd.AddCommand(operatorCmd)
}

// runOperator dispatches to a single-node run (--node) or the daemon loop.
func runOperator() error {
	if opImage == "" || opCertname == "" {
		return fmt.Errorf("--image and --certname are required")
	}

	cfg, err := inClusterConfig()
	if err != nil {
		return err
	}

	if opNode != "" {
		return runOnce(cfg, opNode)
	}
	return runDaemon(cfg)
}

// runOnce creates the Job for a single named node and returns. A pre-existing Job
// (still running or within its TTL) is left untouched, so this is safe to trigger
// repeatedly for the same node.
func runOnce(cfg *clusterConfig, node string) error {
	ok, err := cfg.nodeExists(node)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("node %q not found in cluster", node)
	}

	created, err := createJobForNode(cfg, node)
	if err != nil {
		return err
	}
	if !created {
		slog.Info("job already present, skipped", slog.String("node", node))
		return nil
	}
	slog.Info("created job", slog.String("node", node))
	return nil
}

// runDaemon reconciles immediately, then every opInterval, until SIGINT/SIGTERM.
// A failed reconcile is logged and retried on the next tick rather than crashing
// the daemon.
func runDaemon(cfg *clusterConfig) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ticker := time.NewTicker(opInterval)
	defer ticker.Stop()

	slog.Info("operator started",
		slog.Duration("interval", opInterval),
		slog.String("namespace", cfg.namespace),
		slog.Bool("enforce", opEnforce))

	for {
		if err := reconcile(cfg); err != nil {
			slog.Error("reconcile failed", slog.Any("error", err))
		}
		select {
		case <-ctx.Done():
			slog.Info("shutting down")
			return nil
		case <-ticker.C:
		}
	}
}

// reconcile lists the matching nodes and ensures one agent Job exists per node.
func reconcile(cfg *clusterConfig) error {
	nodes, err := cfg.listNodes(opNodeSelector)
	if err != nil {
		return err
	}
	if len(nodes) == 0 {
		slog.Warn("no nodes matched", slog.String("selector", opNodeSelector))
		return nil
	}

	slog.Info("reconciling", slog.Int("nodes", len(nodes)), slog.Bool("enforce", opEnforce))

	var failed int
	for _, node := range nodes {
		created, err := createJobForNode(cfg, node)
		switch {
		case err != nil:
			slog.Error("create job", slog.String("node", node), slog.Any("error", err))
			failed++
		case created:
			slog.Info("created job", slog.String("node", node))
		default:
			slog.Info("skipped, already present", slog.String("node", node))
		}
	}

	if failed > 0 {
		return fmt.Errorf("%d of %d nodes failed", failed, len(nodes))
	}
	return nil
}

// createJobForNode builds and POSTs the agent Job for one node. created=false with a
// nil error means a Job already exists (HTTP 409) — the source of at-most-one Job per node.
func createJobForNode(cfg *clusterConfig, node string) (bool, error) {
	slug := slugify(node)
	name := jobName(opNamePrefix, slug)

	manifest, err := buildJobManifest(node, name, slug)
	if err != nil {
		return false, fmt.Errorf("build job for %s: %w", node, err)
	}
	return cfg.createJob(manifest)
}

type clusterConfig struct {
	host      string
	namespace string
	client    *http.Client
}

func inClusterConfig() (*clusterConfig, error) {
	host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
	if host == "" || port == "" {
		return nil, fmt.Errorf("not running in-cluster: KUBERNETES_SERVICE_HOST/PORT are unset")
	}

	// Sanity-check the token exists now; it is re-read per request (see do) because the
	// kubelet rotates bound service-account tokens on disk during a long-running daemon.
	if _, err := os.ReadFile(saDir + "/token"); err != nil {
		return nil, fmt.Errorf("read service account token: %w", err)
	}

	caPEM, err := os.ReadFile(saDir + "/ca.crt")
	if err != nil {
		return nil, fmt.Errorf("read cluster CA: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("parse cluster CA")
	}

	namespace := opNamespace
	if namespace == "" {
		b, err := os.ReadFile(saDir + "/namespace")
		if err != nil {
			return nil, fmt.Errorf("read namespace: %w", err)
		}
		namespace = strings.TrimSpace(string(b))
	}

	return &clusterConfig{
		host:      fmt.Sprintf("https://%s:%s", host, port),
		namespace: namespace,
		client: &http.Client{
			Timeout:   30 * time.Second, //nolint:mnd
			Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}},
		},
	}, nil
}

func (c *clusterConfig) do(method, path string, body []byte) (*http.Response, error) {
	token, err := os.ReadFile(saDir + "/token")
	if err != nil {
		return nil, fmt.Errorf("read service account token: %w", err)
	}

	var r io.Reader = http.NoBody
	if body != nil {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, c.host+path, r)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(string(token)))
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.client.Do(req)
}

// nodeList is the trimmed shape decoded from the Kubernetes node-list response.
type nodeList struct {
	Items []nodeListItem `json:"items"`
}

type nodeListItem struct {
	Metadata objMeta `json:"metadata"`
}

func (c *clusterConfig) listNodes(selector string) ([]string, error) {
	path := "/api/v1/nodes"
	if selector != "" {
		path += "?labelSelector=" + url.QueryEscape(selector)
	}

	resp, err := c.do(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list nodes: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}

	var out nodeList
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	names := make([]string, 0, len(out.Items))
	for _, item := range out.Items {
		names = append(names, item.Metadata.Name)
	}
	return names, nil
}

// nodeExists reports whether a node object with the given name is present, so a --node
// run fails fast on a typo instead of creating a Job that never schedules.
func (c *clusterConfig) nodeExists(name string) (bool, error) {
	resp, err := c.do(http.MethodGet, "/api/v1/nodes/"+url.PathEscape(name), nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		b, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("get node %q: %s: %s", name, resp.Status, strings.TrimSpace(string(b)))
	}
}

// createJob POSTs the Job; a 409 means it already exists (still running) and is
// treated as a skip, which is what gives us no-concurrency per node.
func (c *clusterConfig) createJob(manifest []byte) (created bool, err error) {
	resp, err := c.do(http.MethodPost, fmt.Sprintf("/apis/batch/v1/namespaces/%s/jobs", c.namespace), manifest)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusCreated, http.StatusOK:
		return true, nil
	case http.StatusConflict:
		return false, nil
	default:
		b, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
}

func buildJobManifest(node, name, slug string) ([]byte, error) {
	env := []envVar{
		{Name: "CERTNAME", Value: opCertname},
		{Name: "ENFORCE", Value: fmt.Sprintf("%t", opEnforce)},
		{Name: "OPENVOX_ENVIRONMENT", Value: opOpenvoxEnv},
		{Name: "CLIENT_CERT_DIR", Value: "/obmondo-clientcert"},
	}
	if opPuppetServer != "" {
		env = append(env, envVar{Name: "PUPPET_SERVER", Value: opPuppetServer})
	}
	if opControlRepoURL != "" {
		env = append(env, envVar{Name: "CONTROL_REPO_URL", Value: opControlRepoURL})
	}
	if opControlRepoRef != "" {
		env = append(env, envVar{Name: "CONTROL_REPO_REF", Value: opControlRepoRef})
	}

	mounts := []volumeMount{
		{Name: "obmondo-clientcert", MountPath: "/obmondo-clientcert", ReadOnly: true},
		{Name: "host-obmondo", MountPath: opHostPath},
	}
	volumes := []volume{
		{Name: "obmondo-clientcert", Secret: &secretVol{SecretName: opSecretName}},
		{Name: "host-obmondo", HostPath: &hostPathVol{Path: opHostPath, Type: "DirectoryOrCreate"}},
	}
	if opGitSecretName != "" {
		mounts = append(mounts, volumeMount{Name: "git-credentials", MountPath: "/git-credentials", ReadOnly: true})
		volumes = append(volumes, volume{Name: "git-credentials", Secret: &secretVol{SecretName: opGitSecretName}})
	}
	if opHieraConfigMap != "" {
		mounts = append(mounts, volumeMount{Name: "hiera-data", MountPath: "/hiera-data", ReadOnly: true})
		volumes = append(volumes, volume{Name: "hiera-data", ConfigMap: &configMapVol{Name: opHieraConfigMap}})
	}

	labels := map[string]string{
		"app.kubernetes.io/name":           "linuxaid-agents",
		"app.kubernetes.io/managed-by":     "linuxaid-agent",
		"linuxaid-agents.obmondo.com/node": slug,
	}

	j := k8sJob{
		APIVersion: "batch/v1",
		Kind:       "Job",
		Metadata:   objMeta{Name: name, Labels: labels},
		Spec: jobSpec{
			BackoffLimit:            opBackoffLimit,
			ActiveDeadlineSeconds:   opActiveDeadline,
			TTLSecondsAfterFinished: opTTLSeconds,
			Template: podTemplate{
				Metadata: objMeta{Labels: labels},
				Spec: podSpec{
					RestartPolicy:                "Never",
					HostPID:                      true,
					AutomountServiceAccountToken: false,
					NodeName:                     node,
					Tolerations:                  []toleration{{Operator: "Exists"}},
					Containers: []container{{
						Name:            "agent",
						Image:           opImage,
						ImagePullPolicy: opImagePullPolicy,
						SecurityContext: secCtx{Privileged: true, RunAsUser: 0},
						Env:             env,
						VolumeMounts:    mounts,
					}},
					Volumes: volumes,
				},
			},
		},
	}
	return json.Marshal(j)
}

// slugify turns a node name into a DNS-1123 label fragment.
func slugify(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	slug := b.String()
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	slug = strings.Trim(slug, "-")
	if len(slug) > 40 { //nolint:mnd
		slug = strings.Trim(slug[:40], "-")
	}
	return slug
}

// jobName keeps the name a valid DNS-1123 label with room for the pod suffix.
func jobName(prefix, slug string) string {
	name := prefix + "-" + slug
	if len(name) > 57 { //nolint:mnd
		name = strings.TrimRight(name[:57], "-")
	}
	return name
}

type k8sJob struct {
	APIVersion string  `json:"apiVersion"`
	Kind       string  `json:"kind"`
	Metadata   objMeta `json:"metadata"`
	Spec       jobSpec `json:"spec"`
}

type objMeta struct {
	Name   string            `json:"name,omitempty"`
	Labels map[string]string `json:"labels,omitempty"`
}

type jobSpec struct {
	BackoffLimit            int         `json:"backoffLimit"`
	ActiveDeadlineSeconds   int         `json:"activeDeadlineSeconds"`
	TTLSecondsAfterFinished int         `json:"ttlSecondsAfterFinished"`
	Template                podTemplate `json:"template"`
}

type podTemplate struct {
	Metadata objMeta `json:"metadata,omitempty"`
	Spec     podSpec `json:"spec"`
}

type podSpec struct {
	RestartPolicy                string       `json:"restartPolicy"`
	HostPID                      bool         `json:"hostPID"`
	AutomountServiceAccountToken bool         `json:"automountServiceAccountToken"`
	NodeName                     string       `json:"nodeName"`
	Tolerations                  []toleration `json:"tolerations,omitempty"`
	Containers                   []container  `json:"containers"`
	Volumes                      []volume     `json:"volumes"`
}

type toleration struct {
	Operator string `json:"operator"`
}

type container struct {
	Name            string        `json:"name"`
	Image           string        `json:"image"`
	ImagePullPolicy string        `json:"imagePullPolicy,omitempty"`
	SecurityContext secCtx        `json:"securityContext"`
	Env             []envVar      `json:"env,omitempty"`
	VolumeMounts    []volumeMount `json:"volumeMounts"`
}

type secCtx struct {
	Privileged bool `json:"privileged"`
	RunAsUser  int  `json:"runAsUser"`
}

type envVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type volumeMount struct {
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
	ReadOnly  bool   `json:"readOnly,omitempty"`
}

type volume struct {
	Name      string        `json:"name"`
	Secret    *secretVol    `json:"secret,omitempty"`
	HostPath  *hostPathVol  `json:"hostPath,omitempty"`
	ConfigMap *configMapVol `json:"configMap,omitempty"`
}

type secretVol struct {
	SecretName string `json:"secretName"`
}

type configMapVol struct {
	Name string `json:"name"`
}

type hostPathVol struct {
	Path string `json:"path"`
	Type string `json:"type,omitempty"`
}
