package app

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"slices"

	api "gitea.obmondo.com/EnableIT/linuxaid-cli/internal/obmondo"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/shell"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/certs"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/config"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/constant"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/puppet"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/security"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/system"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/webtee"
)

const (
	agentDisabledFile     = constant.AgentDisabledLockFile
	defaultPrometheusHost = "prometheus.obmondo.com"
)

func cleanup(puppetService *puppet.Service) {
	if err := puppetService.EnableAgent(); err != nil {
		slog.Error("unable to remove agent disable file and enable puppet agent", slog.Any("error", err))
	}

	slog.Info("ending system-update")
}

// HandlePuppetRun is resposible to run the puppet-agent and handle the status codes of the execution
func HandlePuppetRun(puppetService *puppet.Service, environment string) error {
	exitCode := puppetService.RunAgent(false, "noop", environment)
	if slices.Contains(constant.PuppetSuccessExitCodes, exitCode) {
		slog.Info("everything is fine with puppet agent run, let's continue.")
		return nil
	}

	slog.Error("puppet failed, aborting.", slog.Int("exit_code", exitCode))
	return fmt.Errorf("puppet failed with exit code: %d", exitCode)
}

func buildPostUpdateComment(exporter security.SecurityExporter) string {
	const fallback = "server has been updated"

	result, err := exporter.TriggerScan()
	if err != nil {
		slog.Error("failed to trigger post-update vulnerability scan", slog.Any("error", err))
		return fallback
	}

	switch {
	case !result.Success:
		slog.Warn("post-update scan returned failure", slog.String("error", result.Error))
		return fallback
	default:
		comment := fmt.Sprintf("server updated, %d/%d CVEs fixed (critical: %d/%d, high: %d/%d, medium: %d/%d, low: %d/%d), %d remaining",
			result.CVEsFixed, result.PreviousTotalCVEs,
			result.CriticalCVEsFixed, result.CriticalCVEsFixed+result.CriticalCVEs,
			result.HighCVEsFixed, result.HighCVEsFixed+result.HighCVEs,
			result.MediumCVEsFixed, result.MediumCVEsFixed+result.MediumCVEs,
			result.LowCVEsFixed, result.LowCVEsFixed+result.LowCVEs,
			result.TotalCVEs)
		slog.Info("post-update scan completed", "comment", comment)
		return comment
	}
}

// extractHostname parses a URL and returns just the hostname.
// Returns empty string if parsing fails.
func extractHostname(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err == nil && parsed.Hostname() != "" {
		return parsed.Hostname()
	}
	return ""
}

// resolveCustomerURLs fetches customer settings and returns resolved prometheus
// and puppet server hostnames. Falls back to defaults if customer has not
// configured them or if the API call fails.
func resolveCustomerURLs(obmondoAPI api.ObmondoClient, cfg config.Config) (prometheusHost, puppetServerHost string) {
	defaultPuppetServer := constant.DefaultPuppetServerCustomerID + constant.DefaultPuppetServerDomainSuffix
	prometheusHost = defaultPrometheusHost
	puppetServerHost = defaultPuppetServer

	// An explicitly provided puppet server (--puppet-server flag or
	// PUPPET_SERVER env) always wins over customer settings.
	defer func() {
		if server := cfg.OpenvoxServer; server != "" {
			if h := extractHostname(server); h != "" {
				server = h
			}
			puppetServerHost = server
		}
	}()

	// Opensource nodes are not registered with Obmondo, so there are no
	// customer settings to look up.
	if cfg.Opensource {
		return
	}

	customerID := certs.GetCustomerID(cfg.Certname)
	if customerID == "" {
		slog.Warn("could not determine customer ID from certname, using defaults",
			slog.String("certname", cfg.Certname))
		return
	}

	settings, err := obmondoAPI.GetCustomerSettings(customerID)
	if err != nil {
		slog.Warn("failed to fetch customer settings, using defaults", slog.Any("error", err))
		return
	}

	if settings.LinuxAid == nil {
		slog.Info("no linuxaid settings configured for customer, using defaults",
			slog.String("customer_id", customerID))
		return
	}

	if settings.LinuxAid.PrometheusURL != "" {
		h := extractHostname(settings.LinuxAid.PrometheusURL)
		if h == "" {
			slog.Warn("could not extract hostname from prometheus_url, using default",
				slog.String("prometheus_url", settings.LinuxAid.PrometheusURL))
		}
		if h != "" {
			prometheusHost = h
		}
	}

	if settings.LinuxAid.PuppetserverURL != "" {
		h := extractHostname(settings.LinuxAid.PuppetserverURL)
		if h == "" {
			slog.Warn("could not extract hostname from puppetserver_url, using default",
				slog.String("puppetserver_url", settings.LinuxAid.PuppetserverURL))
		}
		if h != "" {
			puppetServerHost = h
		}
	}

	return
}

// resolveSystemUpdateEnvironment decides which puppet environment this update runs with. An
// automatic window pins the linuxaid tag of its update cycle, and that tag is what the whole group
// updates to, so it always wins over the environment the API resolves for the node - both over the
// latest release, which would drift ahead mid-cycle, and over an environment pinned for the
// certname. Everything else (adhoc windows, opensource nodes, and a window that pinned no tag)
// falls back to the usual resolution.
func resolveSystemUpdateEnvironment(obmondoAPI api.ObmondoClient, certname string, serviceWindow *api.ServiceWindow, opensource bool) string {
	if !opensource {
		if environment := serviceWindow.PuppetEnvironment(); environment != "" {
			slog.Info("using the environment pinned by the automatic service window",
				slog.String("environment", environment))
			return environment
		}

		if serviceWindow != nil && serviceWindow.WindowType == constant.ServiceWindowTypeAutomatic {
			slog.Warn("the automatic service window pinned no linuxaid tag, asking the API for the environment")
		}
	}

	return resolveOpenvoxEnvironment(obmondoAPI, certname, "", opensource)
}

// SystemUpdate runs the update workflow. A returned error means the run failed in a way the
// caller should surface as a non-zero exit; the paths that give up quietly return nil on purpose,
// so systemd does not mark the unit failed for an inactive window or an unreachable API.
func SystemUpdate(cfg config.Config) error {
	if err := system.LoadOSReleaseEnv(); err != nil {
		return err
	}

	if err := os.Setenv("PATH", constant.PuppetPath); err != nil {
		return fmt.Errorf("failed to set the PATH env: %w", err)
	}

	if err := system.RequireRootUser(); err != nil {
		return err
	}

	if err := system.RequireOSNameEnv(); err != nil {
		return err
	}

	runner := shell.New()

	cmds, err := system.IsSupportedOS(runner)
	if err != nil {
		return fmt.Errorf("OS not supported: %w", err)
	}

	slog.Info("starting system-update")

	// check if agent disable file exists
	openvoxInitiallyEnabled := true
	if _, err := os.Stat(agentDisabledFile); err == nil {
		openvoxInitiallyEnabled = false
		slog.Warn("openvox agent was disabled before system-update, will skip openvox operations")
	}

	obmondoAPIURL := api.GetObmondoURL()
	obmondoAPI := api.NewObmondoClient(obmondoAPIURL, false, cfg.Certname)

	serviceWindowNow, err := obmondoAPI.GetServiceWindowStatus()
	if err != nil {
		slog.Error("unable to get service window status", slog.String("error", err.Error()))
		return nil
	}

	// lets fail with exit 0, otherwise systemd service will be in failed status
	if !serviceWindowNow.IsWindowOpen {
		slog.Warn("exiting, service window is inactive")
		return nil
	}

	slog.Info("service window is active, going ahead")

	if err := cmds.UpdateRepositoryList(); err != nil {
		return fmt.Errorf("unable to update repository: %w", err)
	}

	if err := cmds.CheckAndInstallCaCertificates(); err != nil {
		return fmt.Errorf("unable to check if ca certs are installed: %w", err)
	}

	prometheusHost, puppetServer := resolveCustomerURLs(obmondoAPI, cfg)

	// the resolved customer puppet server is what the rest of the run must use; this used to be
	// written back into the global viper instance for other packages to pick up
	cfg.OpenvoxServer = puppetServer
	slog.Info("resolved customer URLs",
		slog.String("prometheus", prometheusHost),
		slog.String("puppet_server", puppetServer))

	puppetService := puppet.NewService(obmondoAPI, webtee.NewWebtee(obmondoAPI), runner, cfg)

	if openvoxInitiallyEnabled && !cfg.SkipOpenvox {
		// Check if any existing puppet agent is already running
		puppetService.WaitForAgent(constant.PuppetWaitForCertTimeOut)

		// puppet.conf no longer pins an environment, so the run needs the one this window updates to
		environment := resolveSystemUpdateEnvironment(obmondoAPI, cfg.Certname, serviceWindowNow, cfg.Opensource)

		// Run puppet-agent and check the exit code, and exit this script, if it's not 0 or 2
		if err := HandlePuppetRun(puppetService, environment); err != nil {
			slog.Error("unable to run puppet-agent", slog.String("error", err.Error()))
			return nil
		}

		// Disable puppet-agent, since we'll be running upgrade commands
		if err := puppetService.DisableAgent("puppet has been disabled by the system-update"); err != nil {
			slog.Error("failed to disable agent", slog.Any("error", err))
			return nil
		}

		// Ensure the cleanup is done regardless of the outcome of the update script execution
		defer cleanup(puppetService)
	}

	distribution, distIDExists := os.LookupEnv("ID")
	if !distIDExists {
		slog.Error("env variable ID not set")
		return nil
	}

	// Apt/Yum/Zypper update
	if err := system.UpdateSystem(runner, distribution); err != nil {
		slog.Error("unable to update system", slog.String("error", err.Error()))
		return nil
	}

	securityExporterURL := cfg.SecurityExporterURL
	closeComment := buildPostUpdateComment(security.NewSecurityExporter(securityExporterURL))

	if err := obmondoAPI.CloseServiceWindow(serviceWindowNow.WindowType, cfg.Certname, serviceWindowNow.Timezone, closeComment); err != nil {
		slog.Error("unable to close the service window", slog.String("error", err.Error()))
		return nil
	}

	slog.Info("service window is closed now for this respective node")

	// Enable the openvox agent, so openvox runs after reboot and don't exit the script
	// otherwise reboot won't be triggered
	if openvoxInitiallyEnabled {
		cleanup(puppetService)
	}

	if err := system.CheckKernelAndRebootIfNeeded(runner, cfg.NoReboot); err != nil {
		slog.Error("unable to check kernel and reboot", slog.String("error", err.Error()))
		return nil
	}

	return nil
}
