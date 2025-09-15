package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"gitea.obmondo.com/EnableIT/go-scripts/config"
	"gitea.obmondo.com/EnableIT/go-scripts/constant"
	"gitea.obmondo.com/EnableIT/go-scripts/helper"
	"gitea.obmondo.com/EnableIT/go-scripts/pkg/disk"
	api "gitea.obmondo.com/EnableIT/go-scripts/pkg/obmondo"
	"gitea.obmondo.com/EnableIT/go-scripts/pkg/puppet"
	"gitea.obmondo.com/EnableIT/go-scripts/pkg/security"
	"gitea.obmondo.com/EnableIT/go-scripts/pkg/webtee"

	"github.com/bitfield/script"
)

const (
	obmondoAPIURL       = constant.ObmondoAPIURL
	agentDisabledFile   = constant.AgentDisabledLockFile
	path                = constant.PuppetPath
	sleepTime           = 5
	bootDirectory       = "/boot"
	securityExporterURL = "http://127.254.254.254:63396"
)

// 202 -> When a certname says it's done but the overall window is not auto-closed
// 204 -> When a certname says it's done AND the overall window is auto-closed
// 208 -> When any of the above requests happen again and again
var closeWindowSuccessStatuses = map[int]struct{}{
	http.StatusAccepted:        {},
	http.StatusNoContent:       {},
	http.StatusAlreadyReported: {},
}

type ServiceWindow struct {
	IsWindowOpen bool   `json:"is_window_open"`
	WindowType   string `json:"window_type"`
	Timezone     string `json:"timezone"`
}

func cleanup(puppetService *puppet.Service) {
	if err := puppetService.EnableAgent(); err != nil {
		slog.Error("unable to remove agent disable file and enable puppet agent")
	}

	slog.Info("ending obmondo-system-update script")
}

// ------------------------------------------------
// ------------------------------------------------

func GetServiceWindowDetails(response []byte) (*ServiceWindow, error) {
	type ServiceWindowResponse struct {
		Data ServiceWindow `json:"data"`
	}

	var serviceWindowResponse ServiceWindowResponse

	if err := json.Unmarshal(response, &serviceWindowResponse); err != nil {
		slog.Error("failed to parse service window JSON", slog.String("error", err.Error()))
		return nil, err
	}

	return &serviceWindowResponse.Data, nil
}

func GetServiceWindowStatus(obmondoAPICient api.ObmondoClient) (*ServiceWindow, error) {
	resp, err := obmondoAPICient.FetchServiceWindowStatus()
	if err != nil {
		slog.Error("unexpected error fetching service window url", slog.String("error", err.Error()))
		return nil, err
	}

	defer resp.Body.Close()
	statusCode, responseBody, err := helper.ParseResponse(resp)
	if err != nil {
		slog.Error("unexpected error reading response body", slog.String("error", err.Error()))
		return nil, err
	}

	if statusCode != http.StatusOK {
		slog.Error("unexpected", slog.Int("status_code", statusCode), slog.String("response", string(responseBody)))
		return nil, fmt.Errorf("unexpected non-200 HTTP status code received: %d", statusCode)
	}

	serviceWindow, err := GetServiceWindowDetails(responseBody)
	if err != nil {
		slog.Error("unable to determine the service window", slog.String("error", err.Error()))
		return nil, err
	}

	return serviceWindow, nil
}

func CloseServiceWindow(obmondoAPICient api.ObmondoClient, windowType string, location string) error {
	closeWindow, err := closeWindow(obmondoAPICient, windowType, location)
	if err != nil {
		slog.Error("closing service window failed", slog.String("error", err.Error()))
		return err
	}
	defer closeWindow.Body.Close()

	if _, exists := closeWindowSuccessStatuses[closeWindow.StatusCode]; !exists {
		bodyBytes, err := io.ReadAll(closeWindow.Body)
		if err != nil {
			slog.Error("failed to read response body", slog.String("error", err.Error()))
			return err
		}

		// Log the response status code and body
		slog.Error("closing service window failed", slog.Int("status_code", closeWindow.StatusCode), slog.String("response", string(bodyBytes)))
		return fmt.Errorf("incorrect response code received from API: %d", closeWindow.StatusCode)
	}

	return nil
}

func closeWindow(obmondoAPICient api.ObmondoClient, windowType string, location string) (*http.Response, error) {
	closeWindow, err := obmondoAPICient.CloseServiceWindow(windowType, location)
	if err != nil {
		slog.Error("failed to close service window", slog.String("error", err.Error()))
		return nil, err
	}

	return closeWindow, err
}

// ------------------------------------------------
// ------------------------------------------------

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
		slog.Error("unable to write the output to Stdout", slog.String("error", err.Error()))
		return err
	}

	if err := pipe.Wait(); err != nil {
		slog.Error("failed to upgrade all packages", slog.String("error", err.Error()))
	}
	exitStatus := pipe.ExitStatus()
	if exitStatus != 0 {
		slog.Error("exiting, apt update failed")
		return fmt.Errorf(" apt-get update and upgrade failed: exit status %d", exitStatus)
	}

	if err := script.Exec("apt-get autoremove -y").Wait(); err != nil {
		slog.Error("failed to remove unused dependencies", slog.String("error", err.Error()))
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
		slog.Error("unable to write the output to Stdout", slog.String("error", err.Error()))
		return err
	}

	if err := pipe.Wait(); err != nil {
		slog.Error("failed to update all repositories", slog.String("error", err.Error()))
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
		slog.Error("unable to write the output to Stdout", slog.String("error", err.Error()))
		return err
	}

	if err := pipe.Wait(); err != nil {
		slog.Error("failed to update all packages", slog.String("error", err.Error()))
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
	// NOTE: Added to avoid magic number issue with puppet exit codes
	//nolint:all
	var puppetExitCodes = map[string]int{
		"zero": 0,
		"one":  1,
		"two":  2,
		"four": 4,
		"six":  6,
	}
	exitCode := puppetService.RunAgent(false, "noop")

	switch exitCode {
	case puppetExitCodes["zero"], puppetExitCodes["two"]:
		slog.Info("everything is fine with puppet agent run, let's continue.")
		return nil
	case puppetExitCodes["one"]:
		slog.Error("puppet run failed, or wasn't attempted due to another run already in progress.")
		return errors.New("unable to run puppet, or it's already running")
	case puppetExitCodes["four"], puppetExitCodes["six"]:
		slog.Warn("puppet has pending changes, aborting.")
		return errors.New("aborting: puppet has pending changes")
	default:
		slog.Error("puppet failed, aborting.", slog.Int("exit_code", exitCode))
		return fmt.Errorf("puppet failed with exit code: %d", exitCode)
	}
}

// ------------------------------------------------
// ------------------------------------------------

// CheckKernelAndRebootIfNeeded checks if a new kernel is installed and reboots if necessary.
func CheckKernelAndRebootIfNeeded(puppetService *puppet.Service, reboot bool) error {
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
	if installedKernel != runningKernel && reboot {
		// Enable the puppet agent, so puppet runs after reboot and don't exit the script
		// otherwise reboot won't be triggered
		cleanup(puppetService)
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

func obmondoSystemUpdate() {

	reboot := config.DoReboot()

	helper.LoadOSReleaseEnv()

	envErr := os.Setenv("PATH", constant.PuppetPath)
	if envErr != nil {
		slog.Error("failed to set the PATH env, exiting")
		os.Exit(1)
	}

	helper.RequireRootUser()
	helper.RequirePuppetEnv()
	helper.RequireOSNameEnv()
	cmds, err := helper.IsSupportedOS()
	if err != nil {
		slog.Error("OS not supported", slog.String("err", err.Error()))
		os.Exit(1)
	}

	slog.Info("starting obmondo-system-update script")

	// check if agent disable file exists
	if _, err := os.Stat(agentDisabledFile); err == nil {
		slog.Warn("puppet has been disabled, exiting")
		return
	}

	obmondoAPI := api.NewObmondoClient(false)
	puppetService := puppet.NewService(obmondoAPI, webtee.NewWebtee(obmondoAPI))

	serviceWindowNow, err := GetServiceWindowStatus(obmondoAPI)
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

	// Check if any existing puppet agent is already running
	puppetService.WaitForAgent(constant.PuppetWaitForCertTimeOut)

	// Run puppet-agent and check the exit code, and exit this script, if it's not 0 or 2
	if err := HandlePuppetRun(puppetService); err != nil {
		slog.Error("unable to run puppet-agent", slog.String("error", err.Error()))
		return
	}

	// Disable puppet-agent, since we'll be running upgrade commands
	if err := puppetService.DisableAgent("puppet has been disabled by the obmondo-system-update script."); err != nil {
		slog.Error("failed to disable agent", slog.Any("error", err))
		return
	}

	// Ensure the cleanup is done regardless of the outcome of the update script execution
	defer cleanup(puppetService)

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
	if err := CloseServiceWindow(obmondoAPI, serviceWindowNow.WindowType, serviceWindowNow.Timezone); err != nil {
		slog.Error("unable to close the service window", slog.String("error", err.Error()))
		return
	}

	slog.Info("service window is closed now for this respective node")

	if err := CheckKernelAndRebootIfNeeded(puppetService, reboot); err != nil {
		slog.Error("unable to check kernel and reboot", slog.String("error", err.Error()))
		return
	}
}
