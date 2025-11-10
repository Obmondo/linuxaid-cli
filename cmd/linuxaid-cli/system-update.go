package main

import (
	"fmt"
	"log/slog"
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
	agentDisabledFile   = constant.AgentDisabledLockFile
	bootDirectory       = "/boot"
	securityExporterURL = "http://127.254.254.254:63396"
)

var systemUpdateCmd = &cobra.Command{
	Use:     "system-update",
	Short:   "Execute system-update command",
	Long:    "A longer description of system-update command",
	Example: `$ linuxaid-cli system-update --certname web01.example --no-reboot`,
	Run: func(*cobra.Command, []string) {
		SystemUpdate()
	},
}

func cleanup(puppetService *puppet.Service) {
	if err := puppetService.EnableAgent(); err != nil {
		slog.Error("unable to remove agent disable file and enable puppet agent")
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
	case "centos", "rhel":
		return updateRedHat()
	default:
		slog.Error("unknown distribution")
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
		return fmt.Errorf(" apt-get update and upgrade failed: exit status %d", exitStatus)
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

// ------------------------------------------------
// ------------------------------------------------

func SystemUpdate() {
	helper.LoadOSReleaseEnv()

	envErr := os.Setenv("PATH", constant.PuppetPath)
	if envErr != nil {
		slog.Error("failed to set the PATH env, exiting")
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
	if _, err := os.Stat(agentDisabledFile); err == nil {
		slog.Warn("puppet has been disabled, exiting")
		return
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

	puppetService := puppet.NewService(obmondoAPI, webtee.NewWebtee(obmondoAPI))

	if !config.ShouldSkipOpenvox() {
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

	securityExporterService := security.NewSecurityExporter(securityExporterURL)
	if _, err := securityExporterService.GetNumberOfPackageUpdates(); err != nil {
		slog.Error("failed to get response from security exporter for number of package updates endpoint", slog.Any("error", err))
	}

	// Close the service window
	// we need to close it with diff close msg, incase if there is a failure, but that's for later
	if err := obmondoAPI.CloseServiceWindow(serviceWindowNow.WindowType, serviceWindowNow.Timezone); err != nil {
		slog.Error("unable to close the service window", slog.String("error", err.Error()))
		return
	}

	slog.Info("service window is closed now for this respective node")

	// Enable the puppet agent, so puppet runs after reboot and don't exit the script
	// otherwise reboot won't be triggered
	cleanup(puppetService)

	if err := CheckKernelAndRebootIfNeeded(); err != nil {
		slog.Error("unable to check kernel and reboot", slog.String("error", err.Error()))
		return
	}
}

func init() {
	rootCmd.AddCommand(systemUpdateCmd)

	systemUpdateCmd.Flags().BoolVar(&rebootFlag, constant.CobraFlagNoReboot, false, "Set this flag to prevent reboot (default will reboot)")
	systemUpdateCmd.Flags().BoolVar(&skipOpenvoxFlag, constant.CobraFlagSkipOpenvox, false, "Set this flag to prevent running openvox")

	// Bind flags to viper
	v := config.GetViperInstance()
	v.BindPFlag(constant.CobraFlagNoReboot, systemUpdateCmd.Flags().Lookup(constant.CobraFlagNoReboot))
	v.BindPFlag(constant.CobraFlagSkipOpenvox, systemUpdateCmd.Flags().Lookup(constant.CobraFlagSkipOpenvox))

	// Bind environment variables
	v.BindEnv(constant.CobraFlagNoReboot, "NO_REBOOT")
	v.BindEnv(constant.CobraFlagSkipOpenvox, "SKIP_OPENVOX")
}
