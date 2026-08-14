package puppet

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/config"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/constant"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/helper"
	api "gitea.obmondo.com/EnableIT/linuxaid-cli/pkg/obmondo"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/pkg/webtee"

	"github.com/bitfield/script"
)

type Service struct {
	webtee        *webtee.Webtee
	apiClient     api.ObmondoClient
	certName      string
	openvoxServer string
}

// NewService initializes a new Puppet service instance
func NewService(apiClient api.ObmondoClient, webtee *webtee.Webtee) *Service {

	return &Service{
		apiClient:     apiClient,
		certName:      helper.GetCertname(),
		openvoxServer: config.GetOpenvoxServer(),
		webtee:        webtee,
	}
}

// setAgentState runs a puppet agent enable/disable command and reports how it went.
func setAgentState(cmd, action string) error {
	pipe := script.Exec(cmd)
	if err := pipe.Wait(); err != nil {
		return fmt.Errorf("failed to %s puppet agent: %w", action, err)
	}
	if pipe.ExitStatus() != 0 {
		return fmt.Errorf("puppet agent %s exited with non-zero status", action)
	}
	slog.Info("successfully " + action + "d puppet")
	return nil
}

// Enable agent
func (*Service) EnableAgent() error {
	return setAgentState("puppet agent --enable", "enable")
}

// Disable puppet-agent service (sanity-check)
func (s *Service) DisableAgentService() {
	// There is no init script named unattended-upgrades, and puppet in /etc/init.d/ in TurrisOS system
	if os.Getenv("ID") != helper.ConstDistributionNameTurrisOS {
		// Disable unattended-upgrades so puppet-agent package does not update
		s.webtee.RemoteLogObmondo([]string{
			"puppet resource service unattended-upgrades ensure=stopped enable=false",
		}, s.certName)

		// Stop puppet agent service, since we manage it via run_puppet service
		s.webtee.RemoteLogObmondo([]string{
			"puppet resource service puppet ensure=stopped enable=false",
		}, s.certName)

		slog.Debug("puppet agent service disabled")
	}
}

// Disable agent with message
func (*Service) DisableAgent(msg string) error {
	return setAgentState(fmt.Sprintf("puppet agent --disable '%s'", msg), "disable")
}

// RunAgent runs the puppet agent. The environment is passed on the command line rather than read
// from puppet.conf, which no longer pins one: the caller decides which environment a run uses.
func (s *Service) RunAgent(remoteLog bool, noopMode, environment string) int {
	cmd := fmt.Sprintf("puppet agent -t --%s --detailed-exitcodes", noopMode)
	if environment != "" {
		cmd += " --environment " + environment
	}
	if remoteLog {
		s.webtee.RemoteLogObmondo([]string{cmd}, s.certName)
		return 0
	}

	slog.Info("running puppet agent", slog.String("mode", noopMode))
	pipe := script.Exec(cmd)
	if _, err := pipe.Stdout(); err != nil {
		if !helper.IsPuppetSuccessExitCode(pipe.ExitStatus()) {
			slog.Error("stdout error", slog.Any("error", err))
		}
	}

	return pipe.ExitStatus()
}

// Check if agent is running
func (s *Service) IsAgentRunning() bool {
	_, err := os.Stat(constant.AgentRunningLockFile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			slog.Debug("puppet lock file not found")
			s.webtee.RemoteLogObmondo([]string{"echo lock file not found"}, s.certName)
			return false
		}
		slog.Debug("error checking lock file", slog.Any("error", err))
		s.webtee.RemoteLogObmondo([]string{"echo error checking lock file"}, s.certName)
		return false
	}
	return true
}

// Wait until agent stops (or timeout)
func (s *Service) WaitForAgent(timeoutSeconds int) {
	timeout := time.Now().Add(time.Duration(timeoutSeconds) * time.Second)
	for s.IsAgentRunning() {
		if time.Now().After(timeout) {
			slog.Warn("puppet still running, aborting wait")
			break
		}
		time.Sleep(constant.SleepTime * time.Second)
	}
}

// Configure agent
func (s *Service) ConfigureAgent() {
	// no environment here on purpose: pinning one in puppet.conf would compete with the
	// environment the caller passes to each run, and the two would eventually disagree
	cfg := `[main]
server = %s
certname = %s
stringify_facts = false
masterport = 443

[agent]
report = true
pluginsync = true
noop = true
`
	server := helper.NormalizeToHostname(s.openvoxServer)
	content := fmt.Sprintf(cfg, server, s.certName)
	if err := os.WriteFile(constant.PuppetConfig, []byte(content), os.FileMode(os.O_TRUNC|os.O_CREATE)); err != nil {
		s.webtee.RemoteLogObmondo([]string{fmt.Sprintf("echo failed to configure puppet: %s", err)}, s.certName)
		os.Exit(1)
	}
}

// Check server status
func (s *Service) CheckServerStatus() error {
	// The openvox server value is a bare hostname; default to https when no
	// scheme is present.
	server := s.openvoxServer
	if parsed, err := url.Parse(server); err != nil || parsed.Scheme == "" {
		server = "https://" + server
	}

	statusURL := fmt.Sprintf("%s/status/v1/services", server)
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		// nolint: mnd
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(statusURL)
	if err != nil {
		s.webtee.RemoteLogObmondo([]string{fmt.Sprintf("echo Unable to reach Puppetserver: %s", err)}, s.certName)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("puppet server not reachable: %d", resp.StatusCode)
	}
	return nil
}

// Install agent from URL
func (s *Service) DownloadAgent(downloadPath, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.webtee.RemoteLogObmondo([]string{"echo deb file not present at url"}, url)
		return fmt.Errorf("puppet agent download failed with status %d", resp.StatusCode)
	}

	f, err := os.Create(downloadPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}
	return nil
}

func (s *Service) FacterNewSetup() {
	// Ensure facts.d directory exists
	if _, err := script.Exec("mkdir -p /etc/puppetlabs/facter/facts.d").Stdout(); err != nil {
		slog.Error("failed to create facts directory", slog.Any("error", err))
	}

	currentTime := time.Now()
	facter := fmt.Sprintf(
		"---\ninstall_date: %d%02d%02d\n",
		currentTime.Year(),
		currentTime.Month(),
		currentTime.Day(),
	)

	_, err := script.Echo(facter).WriteFile(constant.ExternalFacterFile)
	if err != nil {
		slog.Debug("failed to write external facter file",
			slog.String("file_path", constant.ExternalFacterFile),
			slog.Any("error", err),
		)
		errMsg := fmt.Sprintf("echo cannot create external facter file: %s", err.Error())
		s.webtee.RemoteLogObmondo([]string{errMsg}, s.certName)
		os.Exit(1)
	}

	slog.Debug("facter external setup file created", slog.String("path", constant.ExternalFacterFile))
}
