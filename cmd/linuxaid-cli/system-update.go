package main

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"slices"
	"strings"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/config"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/constant"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/helper"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/pkg/disk"
	api "gitea.obmondo.com/EnableIT/linuxaid-cli/pkg/obmondo"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/pkg/puppet"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/pkg/security"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/pkg/webtee"

	"github.com/bitfield/script"
	"github.com/spf13/cobra"
)

const (
	agentDisabledFile          = constant.AgentDisabledLockFile
	bootDirectory              = "/boot"
	defaultSecurityExporterURL = "http://127.254.254.254:63396"
)

var securityExporterURLFlag string

var systemUpdateCmd = &cobra.Command{
	Use:     "system-update",
	Short:   "Execute system-update command",
	Long:    "A longer description of system-update command",
	Example: `$ linuxaid-cli system-update --certname web01.example --no-reboot`,
	PreRun: func(*cobra.Command, []string) {
		if config.ShouldSkipOpenvox() {
			slog.Info("Openvox-agent run will be skipped")
		}
	},
	Run: func(*cobra.Command, []string) {
		SystemUpdate()
	},
}

func cleanup(puppetService *puppet.Service) {
	if err := puppetService.EnableAgent(); err != nil {
		slog.Error("unable to remove agent disable file and enable puppet agent", slog.Any("error", err))
	}

	slog.Info("ending system-update")
}

// UpdateSystem performs a system update based on the specified Linux distribution.
//
// This function accepts a `distribution` string representing the type of Linux distribution that needs
// to be updated. Depending on the distribution provided, it will invoke the appropriate update function.
func UpdateSystem(distribution string) error {
	switch distribution {
	case "ubuntu", "debian":
		return updateDebian()
	case "sles":
		return updateSUSE()
	case "centos", "rhel", "rocky":
		return updateRedHat()
	default:
		slog.Error("unknown distribution", slog.String("distribution", distribution))
		return nil
	}
}

func updateDebian() error {
	slog.Info("running apt update/upgrade/autoremove")
	enverr := os.Setenv("DEBIAN_FRONTEND", "noninteractive")
	if enverr != nil {
		slog.Error(enverr.Error())
		os.Exit(1)
	}

	if err := script.Exec("apt-get update").Wait(); err != nil {
		slog.Error("failed to update all repositories", slog.String("error", err.Error()))
	}
	pipe := script.Exec("apt-get --with-new-pkgs upgrade -y")
	_, err := pipe.Stdout()
	if err != nil {
		slog.Error("failed to upgrade all packages", slog.String("error", err.Error()))
		return err
	}

	exitStatus := pipe.ExitStatus()
	if exitStatus != 0 {
		slog.Error("exiting, apt update failed")
		return fmt.Errorf("apt-get update and upgrade failed: exit status %d", exitStatus)
	}

	pipe = script.Exec("apt-get autoremove -y")
	_, err = pipe.Stdout()
	if err != nil {
		slog.Error("failed to remove unused dependencies", slog.String("error", err.Error()))
		return err
	}

	return nil
}

func updateSUSE() error {
	slog.Info("running zypper refresh/update")
	if err := script.Exec("zypper refresh").Wait(); err != nil {
		slog.Error("failed to refresh all repositories", slog.String("error", err.Error()))
	}

	pipe := script.Exec("zypper update -y")
	_, err := pipe.Stdout()
	if err != nil {
		slog.Error("failed to update all repositories", slog.String("error", err.Error()))
		return err
	}

	exitStatus := pipe.ExitStatus()
	if exitStatus != 0 {
		slog.Error("exiting, suse update failed")
		return fmt.Errorf("suse update failed: exit status %d", exitStatus)
	}

	return nil
}

func updateRedHat() error {
	slog.Info("running yum repolist/update")
	if err := script.Exec("yum repolist").Wait(); err != nil {
		slog.Error("failed to fetch all repositories", slog.String("error", err.Error()))
	}

	pipe := script.Exec("yum update -y")
	_, err := pipe.Stdout()
	if err != nil {
		slog.Error("failed to update all packages", slog.String("error", err.Error()))
		return err
	}

	exitStatus := pipe.ExitStatus()
	if exitStatus != 0 {
		slog.Error("exiting, yum update failed")
		return fmt.Errorf("yum update failed: exit status %d", exitStatus)
	}

	return nil
}

// ------------------------------------------------
// ------------------------------------------------

// HandlePuppetRun is resposible to run the puppet-agent and handle the status codes of the execution
func HandlePuppetRun(puppetService *puppet.Service) error {
	exitCode := puppetService.RunAgent(false, "noop")
	if slices.Contains(constant.PuppetSuccessExitCodes, exitCode) {
		slog.Info("everything is fine with puppet agent run, let's continue.")
		return nil
	}

	slog.Error("puppet failed, aborting.", slog.Int("exit_code", exitCode))
	return fmt.Errorf("puppet failed with exit code: %d", exitCode)
}

// ------------------------------------------------
// ------------------------------------------------

// CheckKernelAndRebootIfNeeded checks if a new kernel is installed and reboots if necessary.
func CheckKernelAndRebootIfNeeded() error {
	// Get installed kernel of the system
	// If kernel is installed, then only we will try to reboot.
	// In lxc kernel wont be present
	installedKernel, err := getInstalledKernel(bootDirectory)
	if err != nil {
		slog.Error("error occurred while trying to find kernel", slog.String("error", err.Error()))
		return err
	}
	if installedKernel == "" {
		slog.Warn("looks like no kernel is installed on the node")
		return nil
	}

	// Get running kernel of the system
	runningKernel, err := script.Exec("uname -r").String()
	if err != nil {
		slog.Error("Failed to fetch Running Kernel", slog.String("error", err.Error()))
		return err
	}
	runningKernel = strings.TrimSpace(runningKernel)

	// Check the disk size
	if err := disk.CheckDiskSize(); err != nil {
		slog.Error("unable to check disk size", slog.String("error", err.Error()))
		return err
	}

	// Reboot the node, if we have installed a new kernel
	if installedKernel != runningKernel && !config.NoReboot() {
		slog.Info("looks like newer kernel is installed, so going ahead with reboot now")
		script.Exec("reboot --force")
	}

	return nil
}

// getInstalledKernel returns the installed Kernel
func getInstalledKernel(bootDirectory string) (string, error) {
	formatedBashCommand := fmt.Sprintf("find %s/vmlinuz-* | sort -V | tail -n 1 | sed 's|.*vmlinuz-||'", bootDirectory)
	installedKernel, err := script.Exec(fmt.Sprintf("/bin/bash -c \"%s\"", formatedBashCommand)).String()
	installedKernel = strings.TrimSpace(installedKernel)
	return installedKernel, err
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

const defaultPrometheusHost = "prometheus.obmondo.com"

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
func resolveCustomerURLs(obmondoAPI api.ObmondoClient, certname string) (prometheusHost, puppetServerHost string) {
	defaultPuppetServer := constant.DefaultPuppetServerCustomerID + constant.DefaultPuppetServerDomainSuffix
	prometheusHost = defaultPrometheusHost
	puppetServerHost = defaultPuppetServer

	// An explicitly provided puppet server (--puppet-server flag or
	// PUPPET_SERVER env) always wins over customer settings.
	defer func() {
		if server := config.GetOpenvoxServer(); server != "" {
			if h := extractHostname(server); h != "" {
				server = h
			}
			puppetServerHost = server
		}
	}()

	customerID := helper.GetCustomerID(certname)
	if customerID == "" {
		slog.Warn("could not determine customer ID from certname, using defaults",
			slog.String("certname", certname))
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

// ------------------------------------------------
// ------------------------------------------------

func SystemUpdate() {
	helper.LoadOSReleaseEnv()

	envErr := os.Setenv("PATH", constant.PuppetPath)
	if envErr != nil {
		slog.Error("failed to set the PATH env, exiting", slog.Any("error", envErr))
		os.Exit(1)
	}

	helper.RequireRootUser()
	helper.RequireOSNameEnv()
	cmds, err := helper.IsSupportedOS()
	if err != nil {
		slog.Error("OS not supported", slog.String("err", err.Error()))
		os.Exit(1)
	}

	slog.Info("starting system-update")

	// check if agent disable file exists
	openvoxInitiallyEnabled := true
	if _, err := os.Stat(agentDisabledFile); err == nil {
		openvoxInitiallyEnabled = false
		slog.Warn("openvox agent was disabled before system-update, will skip openvox operations")
	}

	obmondoAPIURL := api.GetObmondoURL()
	obmondoAPI := api.NewObmondoClient(obmondoAPIURL, false)

	serviceWindowNow, err := obmondoAPI.GetServiceWindowStatus()
	if err != nil {
		slog.Error("unable to get service window status", slog.String("error", err.Error()))
		return
	}

	// lets fail with exit 0, otherwise systemd service will be in failed status
	if !serviceWindowNow.IsWindowOpen {
		slog.Warn("exiting, service window is inactive")
		return
	}

	slog.Info("service window is active, going ahead")

	if err := cmds.UpdateRepositoryList(); err != nil {
		slog.Error("unable to update repository", slog.String("err", err.Error()))
		os.Exit(1)
	}

	if err := cmds.CheckAndInstallCaCertificates(); err != nil {
		slog.Error("unable to check if ca certs are installed", slog.String("err", err.Error()))
		os.Exit(1)
	}

	certname := helper.GetCertname()
	prometheusHost, puppetServer := resolveCustomerURLs(obmondoAPI, certname)
	config.GetViperInstance().Set(constant.CobraFlagOpenvoxServer, puppetServer)
	slog.Info("resolved customer URLs",
		slog.String("prometheus", prometheusHost),
		slog.String("puppet_server", puppetServer))

	puppetService := puppet.NewService(obmondoAPI, webtee.NewWebtee(obmondoAPI))

	if openvoxInitiallyEnabled && !config.ShouldSkipOpenvox() {
		// Check if any existing puppet agent is already running
		puppetService.WaitForAgent(constant.PuppetWaitForCertTimeOut)

		// Run puppet-agent and check the exit code, and exit this script, if it's not 0 or 2
		if err := HandlePuppetRun(puppetService); err != nil {
			slog.Error("unable to run puppet-agent", slog.String("error", err.Error()))
			return
		}

		// Disable puppet-agent, since we'll be running upgrade commands
		if err := puppetService.DisableAgent("puppet has been disabled by the system-update"); err != nil {
			slog.Error("failed to disable agent", slog.Any("error", err))
			return
		}

		// Ensure the cleanup is done regardless of the outcome of the update script execution
		defer cleanup(puppetService)
	}

	distribution, distIDExists := os.LookupEnv("ID")
	if !distIDExists {
		slog.Error("env variable ID not set")
		return
	}

	// Apt/Yum/Zypper update
	if err := UpdateSystem(distribution); err != nil {
		slog.Error("unable to update system", slog.String("error", err.Error()))
		return
	}

	securityExporterURL := config.GetSecurityExporterURL()
	closeComment := buildPostUpdateComment(security.NewSecurityExporter(securityExporterURL))

	if err := obmondoAPI.CloseServiceWindow(serviceWindowNow.WindowType, helper.GetCertname(), serviceWindowNow.Timezone, closeComment); err != nil {
		slog.Error("unable to close the service window", slog.String("error", err.Error()))
		return
	}

	slog.Info("service window is closed now for this respective node")

	// Enable the openvox agent, so openvox runs after reboot and don't exit the script
	// otherwise reboot won't be triggered
	if openvoxInitiallyEnabled {
		cleanup(puppetService)
	}

	if err := CheckKernelAndRebootIfNeeded(); err != nil {
		slog.Error("unable to check kernel and reboot", slog.String("error", err.Error()))
		return
	}
}

func init() {
	rootCmd.AddCommand(systemUpdateCmd)

	systemUpdateCmd.Flags().BoolVar(&rebootFlag, constant.CobraFlagNoReboot, false, "Set this flag to prevent reboot (default will reboot)")
	systemUpdateCmd.Flags().BoolVar(&skipOpenvoxFlag, constant.CobraFlagSkipOpenvox, false, "Set this flag to prevent running openvox")
	systemUpdateCmd.Flags().StringVar(&securityExporterURLFlag, constant.CobraFlagSecurityExporterURL, defaultSecurityExporterURL, "Security exporter URL")

	// Bind flags to viper
	v := config.GetViperInstance()
	v.BindPFlag(constant.CobraFlagNoReboot, systemUpdateCmd.Flags().Lookup(constant.CobraFlagNoReboot))
	v.BindPFlag(constant.CobraFlagSkipOpenvox, systemUpdateCmd.Flags().Lookup(constant.CobraFlagSkipOpenvox))
	v.BindPFlag(constant.CobraFlagSecurityExporterURL, systemUpdateCmd.Flags().Lookup(constant.CobraFlagSecurityExporterURL))

	// Bind environment variables
	v.BindEnv(constant.CobraFlagNoReboot, "NO_REBOOT")
	v.BindEnv(constant.CobraFlagSkipOpenvox, "SKIP_OPENVOX")
	v.BindEnv(constant.CobraFlagSecurityExporterURL, "SECURITY_EXPORTER_URL")
}
