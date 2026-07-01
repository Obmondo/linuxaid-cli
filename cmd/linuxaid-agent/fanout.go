package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const saDir = "/var/run/secrets/kubernetes.io/serviceaccount"

var (
	faImage          string
	faCertname       string
	faEnforce        bool
	faOpenvoxEnv     string
	faPuppetServer   string
	faSecretName     string
	faHostPath       string
	faNodeSelector   string
	faNamePrefix     string
	faNamespace      string
	faTTLSeconds     int
	faActiveDeadline int
	faBackoffLimit   int
)

var fanoutCmd = &cobra.Command{
	Use:   "fanout",
	Short: "List cluster nodes and create one agent Job per node",
	RunE: func(_ *cobra.Command, _ []string) error {
		return runFanout()
	},
}

func init() {
	f := fanoutCmd.Flags()
	f.StringVar(&faImage, "image", "", "agent container image (required)")
	f.StringVar(&faCertname, "certname", "", "obmondo-clientcert CN, shared by all nodes (required)")
	f.BoolVar(&faEnforce, "enforce", false, "apply changes (puppet --no-noop) instead of report-only")
	f.StringVar(&faOpenvoxEnv, "openvox-environment", "master", "LinuxAid/OpenVox environment")
	f.StringVar(&faPuppetServer, "puppet-server", "", "optional puppetserver override")
	f.StringVar(&faSecretName, "cert-secret", "obmondo-clientcert", "secret holding the pre-signed client cert")
	f.StringVar(&faHostPath, "host-path", "/opt/obmondo", "host path for staging the CLI binary")
	f.StringVar(&faNodeSelector, "node-selector", "", "label selector to filter nodes (empty = all)")
	f.StringVar(&faNamePrefix, "name-prefix", "linuxaid-agents", "prefix for the per-node Job names")
	f.StringVar(&faNamespace, "namespace", "", "namespace to create Jobs in (default: the pod's namespace)")
	f.IntVar(&faTTLSeconds, "ttl-seconds", 600, "ttlSecondsAfterFinished on the agent Jobs")
	f.IntVar(&faActiveDeadline, "active-deadline-seconds", 1200, "activeDeadlineSeconds on the agent Jobs")
	f.IntVar(&faBackoffLimit, "backoff-limit", 1, "backoffLimit on the agent Jobs")
	rootCmd.AddCommand(fanoutCmd)
}

func runFanout() error {
	if faImage == "" || faCertname == "" {
		return fmt.Errorf("--image and --certname are required")
	}

	cfg, err := inClusterConfig()
	if err != nil {
		return err
	}

	nodes, err := cfg.listNodes(faNodeSelector)
	if err != nil {
		return err
	}
	if len(nodes) == 0 {
		slog.Warn("no nodes matched", slog.String("selector", faNodeSelector))
		return nil
	}

	slog.Info("fanning out agent jobs",
		slog.Int("nodes", len(nodes)),
		slog.String("namespace", cfg.namespace),
		slog.Bool("enforce", faEnforce))

	var failed int
	for _, node := range nodes {
		slug := slugify(node)
		name := jobName(faNamePrefix, slug)

		manifest, err := buildJobManifest(node, name, slug)
		if err != nil {
			slog.Error("build job", slog.String("node", node), slog.Any("error", err))
			failed++
			continue
		}

		created, err := cfg.createJob(manifest)
		switch {
		case err != nil:
			slog.Error("create job", slog.String("node", node), slog.Any("error", err))
			failed++
		case created:
			slog.Info("created job", slog.String("job", name), slog.String("node", node))
		default:
			slog.Info("skipped, already present (no-concurrency)", slog.String("job", name), slog.String("node", node))
		}
	}

	if failed > 0 {
		return fmt.Errorf("%d of %d nodes failed", failed, len(nodes))
	}
	return nil
}

type clusterConfig struct {
	host      string
	token     string
	namespace string
	client    *http.Client
}

func inClusterConfig() (*clusterConfig, error) {
	host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
	if host == "" || port == "" {
		return nil, fmt.Errorf("not running in-cluster: KUBERNETES_SERVICE_HOST/PORT are unset")
	}

	token, err := os.ReadFile(saDir + "/token")
	if err != nil {
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

	namespace := faNamespace
	if namespace == "" {
		b, err := os.ReadFile(saDir + "/namespace")
		if err != nil {
			return nil, fmt.Errorf("read namespace: %w", err)
		}
		namespace = strings.TrimSpace(string(b))
	}

	return &clusterConfig{
		host:      fmt.Sprintf("https://%s:%s", host, port),
		token:     strings.TrimSpace(string(token)),
		namespace: namespace,
		client: &http.Client{
			Timeout:   30 * time.Second, //nolint:mnd
			Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}},
		},
	}, nil
}

func (c *clusterConfig) do(method, path string, body []byte) (*http.Response, error) {
	var r io.Reader = http.NoBody
	if body != nil {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, c.host+path, r)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.client.Do(req)
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

	var out struct {
		Items []struct {
			Metadata struct {
				Name string `json:"name"`
			} `json:"metadata"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	names := make([]string, 0, len(out.Items))
	for _, item := range out.Items {
		names = append(names, item.Metadata.Name)
	}
	return names, nil
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
		{Name: "CERTNAME", Value: faCertname},
		{Name: "ENFORCE", Value: fmt.Sprintf("%t", faEnforce)},
		{Name: "OPENVOX_ENVIRONMENT", Value: faOpenvoxEnv},
		{Name: "CLIENT_CERT_DIR", Value: "/obmondo-clientcert"},
	}
	if faPuppetServer != "" {
		env = append(env, envVar{Name: "PUPPET_SERVER", Value: faPuppetServer})
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
			BackoffLimit:            faBackoffLimit,
			ActiveDeadlineSeconds:   faActiveDeadline,
			TTLSecondsAfterFinished: faTTLSeconds,
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
						Image:           faImage,
						SecurityContext: secCtx{Privileged: true, RunAsUser: 0},
						Env:             env,
						VolumeMounts: []volumeMount{
							{Name: "obmondo-clientcert", MountPath: "/obmondo-clientcert", ReadOnly: true},
							{Name: "host-obmondo", MountPath: faHostPath},
						},
					}},
					Volumes: []volume{
						{Name: "obmondo-clientcert", Secret: &secretVol{SecretName: faSecretName}},
						{Name: "host-obmondo", HostPath: &hostPathVol{Path: faHostPath, Type: "DirectoryOrCreate"}},
					},
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
	Name     string       `json:"name"`
	Secret   *secretVol   `json:"secret,omitempty"`
	HostPath *hostPathVol `json:"hostPath,omitempty"`
}

type secretVol struct {
	SecretName string `json:"secretName"`
}

type hostPathVol struct {
	Path string `json:"path"`
	Type string `json:"type,omitempty"`
}
