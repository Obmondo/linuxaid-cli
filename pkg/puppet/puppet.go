package puppet

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"time"

	"gitea.obmondo.com/EnableIT/go-scripts/config"
	"gitea.obmondo.com/EnableIT/go-scripts/constant"
	api "gitea.obmondo.com/EnableIT/go-scripts/pkg/obmondo"
	"gitea.obmondo.com/EnableIT/go-scripts/pkg/webtee"

	"github.com/bitfield/script"
)

type Service struct {
	webtee       *webtee.Webtee
	apiClient    api.ObmondoClient
	certName     string
	puppetServer string
}

// NewService initializes a new Puppet service instance
func NewService(apiClient api.ObmondoClient, webtee *webtee.Webtee) *Service {
	return &Service{
		apiClient:    apiClient,
		certName:     config.GetCertName(),
		puppetServer: config.GetPupeptServer(),
		webtee:       webtee,
	}
}

// Enable agent
func (*Service) EnableAgent() error {
	pipe := script.Exec("puppet agent --enable")
	if err := pipe.Wait(); err != nil {
		return fmt.Errorf("failed to enable puppet agent: %w", err)
	}
	if pipe.ExitStatus() != 0 {
		return fmt.Errorf("puppet agent enable exited with non-zero status")
	}
	slog.Info("successfully enabled puppet")
	return nil
}

// Disable puppet-agent service (sanity-check)
func (s *Service) DisableAgentService() {
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

// Disable agent with message
func (*Service) DisableAgent(msg string) error {
	cmd := fmt.Sprintf("puppet agent --disable '%s'", msg)
	pipe := script.Exec(cmd)
	if err := pipe.Wait(); err != nil {
		return fmt.Errorf("failed to disable puppet agent: %w", err)
	}
	if pipe.ExitStatus() != 0 {
		return fmt.Errorf("puppet agent disable exited with non-zero status")
	}
	slog.Info("successfully disabled puppet")
	return nil
}

// Run agent
func (s *Service) RunAgent(remoteLog bool, noopMode string) int {
	cmd := fmt.Sprintf("puppet agent -t --%s --detailed-exitcodes", noopMode)
	if remoteLog {
		s.webtee.RemoteLogObmondo([]string{cmd}, s.certName)
		return 0
	}

	slog.Info("running puppet agent", slog.String("mode", noopMode))
	pipe := script.Exec(cmd)
	if _, err := pipe.Stdout(); err != nil {
		slog.Error("stdout error", slog.Any("error", err))
	}
	if err := pipe.Wait(); err != nil {
		slog.Error("puppet run failed", slog.Any("error", err))
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
	cfg := `[main]
server = %s
certname = %s
stringify_facts = false
masterport = 443

[agent]
report = true
pluginsync = true
noop = true
environment = master
`
	content := fmt.Sprintf(cfg, s.puppetServer, s.certName)
	if _, err := script.Echo(content).WriteFile(constant.PuppetConfig); err != nil {
		s.webtee.RemoteLogObmondo([]string{fmt.Sprintf("echo failed to configure puppet: %s", err)}, s.certName)
		os.Exit(1)
	}
}

// Check server status
func (s *Service) CheckServerStatus() error {
	url := fmt.Sprintf("https://%s/status/v1/services", s.puppetServer)
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		// nolint: mnd
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		s.webtee.RemoteLogObmondo([]string{fmt.Sprintf("echo failed to reach Puppet server: %s", err)}, s.certName)
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

	if _, err := io.Copy(io.MultiWriter(f), resp.Body); err != nil {
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
