package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go-scripts/constants"
	disk "go-scripts/pkg/disk"
	api "go-scripts/pkg/obmondo"
	puppet "go-scripts/pkg/puppet"
	"go-scripts/util"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/bitfield/script"
)

const (
	obmondoAPIURL     = constants.ObmondoAPIURL
	agentDisabledFile = constants.AgentDisabledLockFile
	path              = constants.PuppetPath
	sleepTime         = 5
	bootDirectory     = "/boot"
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
}

func cleanup() {
	if !puppet.EnablePuppetAgent() {
		log.Println("Unable to remove agent disable file and enable puppet agent")
	}

	log.Println("Ending Obmondo System Update Script")
}

// ------------------------------------------------
// ------------------------------------------------

func GetServiceWindowDetails(response []byte) (*ServiceWindow, error) {
	type ServiceWindowResponse struct {
		Data ServiceWindow `json:"data"`
	}

	var serviceWindowResponse ServiceWindowResponse

	if err := json.Unmarshal(response, &serviceWindowResponse); err != nil {
		log.Printf("Failed to parse service window JSON: %v", err)
		return nil, err
	}

	return &serviceWindowResponse.Data, nil
}

func GetServiceWindowStatus(obmondoAPICient api.ObmondoClient) (bool, string, error) {
	resp, err := obmondoAPICient.FetchServiceWindowStatus()
	if err != nil {
		log.Printf("Unexpected error fetching service window url: %s\n", err)
		return false, "", err
	}

	defer resp.Body.Close()
	statusCode, responseBody, err := util.ParseResponse(resp)
	if err != nil {
		log.Printf("Unexpected error reading response body: %s\n", err)
		return false, "", err
	}

	if statusCode != http.StatusOK {
		log.Printf("Response: %s\n", string(responseBody))
		log.Printf("HTTP status is not 200; status code: %d\n", statusCode)
		return false, "", fmt.Errorf("unexpected non-200 http status code received: %d", statusCode)
	}

	serviceWindow, err := GetServiceWindowDetails(responseBody)
	if err != nil {
		log.Printf("Unable to determine the service window: %s", err)
		return false, "", err
	}

	return serviceWindow.IsWindowOpen, serviceWindow.WindowType, nil
}

func CloseServiceWindow(obmondoAPICient api.ObmondoClient, windowType string) error {
	closeWindow, err := closeWindow(obmondoAPICient, windowType)
	if err != nil {
		log.Printf("Closing service window failed: %s", err)
		return err
	}
	defer closeWindow.Body.Close()

	if _, exists := closeWindowSuccessStatuses[closeWindow.StatusCode]; !exists {
		bodyBytes, err := io.ReadAll(closeWindow.Body)
		if err != nil {
			log.Printf("Failed to read response body: %s", err)
			return err
		}

		// Log the response status code and body
		log.Printf("Closing service window failed, wrong response code from API: %d, Response body: %s", closeWindow.StatusCode, bodyBytes)
		return fmt.Errorf("incorrect response code received from API: %d", closeWindow.StatusCode)
	}

	return nil
}

func closeWindow(obmondoAPICient api.ObmondoClient, windowType string) (*http.Response, error) {
	closeWindow, err := obmondoAPICient.CloseServiceWindow(windowType)
	if err != nil {
		log.Println("Failed to close service window", err)
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
		log.Println("Unknown distribution")
		return nil
	}
}

func updateDebian() error {
	log.Println("Running apt update/upgrade/autoremove")
	enverr := os.Setenv("DEBIAN_FRONTEND", "noninteractive")
	if enverr != nil {
		log.Fatal(enverr)
	}

	script.Exec("apt-get update").Wait()
	pipe := script.Exec("apt-get upgrade -y")
	_, err := pipe.Stdout()
	if err != nil {
		log.Printf("unable to write the output to Stdout: %s", err)
		return err
	}

	pipe.Wait()
	exitStatus := pipe.ExitStatus()
	if exitStatus != 0 {
		log.Println("exiting, apt update failed")
		return fmt.Errorf(" apt-get update and upgrade failed: exit status %d", exitStatus)
	}

	script.Exec("apt-get autoremove -y").Wait()

	return nil
}

func updateSUSE() error {
	log.Println("Running zypper refresh/update")
	script.Exec("zypper refresh").Wait()

	pipe := script.Exec("zypper update -y")
	_, err := pipe.Stdout()
	if err != nil {
		log.Printf("unable to write the output to Stdout: %s", err)
		return err
	}

	pipe.Wait()
	exitStatus := pipe.ExitStatus()
	if exitStatus != 0 {
		log.Println("exiting, suse update failed")
		return fmt.Errorf("suse update failed: exit status %d", exitStatus)
	}

	return nil
}

func updateRedHat() error {
	log.Println("Running yum repolist/update")
	script.Exec("yum repolist").Wait()

	pipe := script.Exec("yum update -y")
	_, err := pipe.Stdout()
	if err != nil {
		log.Printf("unable to write the output to Stdout: %s", err)
		return err
	}

	pipe.Wait()
	exitStatus := pipe.ExitStatus()
	if exitStatus != 0 {
		log.Println("exiting, yum update failed")
		return fmt.Errorf("yum update failed: exit status %d", exitStatus)
	}

	return nil
}

// ------------------------------------------------
// ------------------------------------------------

// HandlePuppetRun is resposible to run the puppet-agent and handle the status codes of the execution
func HandlePuppetRun() error {
	// NOTE: Added to avoid magic number issue with puppet exit codes
	//nolint:all
	var puppetExitCodes = map[string]int{
		"zero": 0,
		"one":  1,
		"two":  2,
		"four": 4,
		"six":  6,
	}
	exitCode := puppet.RunPuppetAgent(false, "noop")

	switch exitCode {
	case puppetExitCodes["zero"], puppetExitCodes["two"]:
		log.Println("Everything is fine with puppet agent run, let's continue.")
		return nil
	case puppetExitCodes["one"]:
		log.Println("Puppet run failed, or wasn't attempted due to another run already in progress.")
		return errors.New("unable to run puppet, or it's already running")
	case puppetExitCodes["four"], puppetExitCodes["six"]:
		log.Println("Puppet has pending changes, aborting.")
		return errors.New("aborting: puppet has pending changes")
	default:
		log.Println("Puppet failed with exit code", exitCode, ", aborting.")
		return fmt.Errorf("puppet failed with exit code: %d", exitCode)
	}
}

// ------------------------------------------------
// ------------------------------------------------

// CheckKernelAndRebootIfNeeded checks if a new kernel is installed and reboots if necessary.
func CheckKernelAndRebootIfNeeded(reboot bool) error {
	// Get installed kernel of the system
	// If kernel is installed, then only we will try to reboot.
	// In lxc kernel wont be present
	installedKernel, err := getInstalledKernel(bootDirectory)
	if err != nil {
		log.Printf("error occurred while trying to find kernel :%s", err)
		return err
	}
	if installedKernel == "" {
		log.Println("Looks like no kernel is installed on the node")
		return nil
	}

	// Get running kernel of the system
	runningKernel, err := script.Exec("uname -r").String()
	if err != nil {
		log.Printf("Failed to fetch Running Kernel: %s", err)
		return err
	}
	runningKernel = strings.TrimSpace(runningKernel)

	// Check the disk size
	if err := disk.CheckDiskSize(); err != nil {
		log.Printf("unable to check disk size: %s", err)
		return err
	}

	// Reboot the node, if we have installed a new kernel
	if installedKernel != runningKernel && reboot {
		// Enable the puppet agent, so puppet runs after reboot and don't exit the script
		// otherwise reboot won't be triggered
		cleanup()
		log.Println("Looks like newer kernel is installed, so going ahead with reboot now")
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

func main() {
	reboot := flag.Bool("reboot", true, "Set this flag false to prevent reboot")

	flag.Parse()

	util.LoadOSReleaseEnv()

	envErr := os.Setenv("PATH", constants.PuppetPath)
	if envErr != nil {
		log.Fatal("failed to set the PATH env, exiting")
	}

	util.CheckUser()
	util.CheckPuppetEnv()
	util.CheckOSNameEnv()
	util.SupportedOS()

	log.Println("Starting Obmondo System Update Script")

	// check if agent disable file exists
	if _, err := os.Stat(agentDisabledFile); err == nil {
		log.Println("Puppet has been disabled, exiting")
		return
	}

	obmondoAPICient := api.NewObmondoClient()
	isServiceWindow, windowType, err := GetServiceWindowStatus(obmondoAPICient)
	if err != nil {
		log.Printf("Unable to get service window status: %s", err)
		return
	}

	// lets fail with exit 0, otherwise systemd service will be in failed status
	if !isServiceWindow {
		log.Println("Exiting, Service window is NOT active")
		return
	}

	log.Println("Service window is active, going ahead")

	// Check if any existing puppet agent is already running
	puppet.WaitForPuppetAgent()

	// Run puppet-agent and check the exit code, and exit this script, if it's not 0 or 2
	if err := HandlePuppetRun(); err != nil {
		log.Printf("unable to run puppet-agent: %s", err)
		return
	}

	// Disable puppet-agent, since we'll be running upgrade commands
	if !puppet.DisablePuppetAgent("Puppet has been disabled by the obmondo-system-update script.") {
		log.Println("unable to disable the puppet agent")
		return
	}

	// Ensure the cleanup is done regardless of the outcome of the update script execution
	defer cleanup()

	distribution, distIDExists := os.LookupEnv("ID")
	if !distIDExists {
		log.Println("ID env variable not set")
		return
	}

	// Apt/Yum/Zypper update
	if err := UpdateSystem(distribution); err != nil {
		log.Printf("unable to update system: %s", err)
		return
	}

	// Close the service window
	// we need to close it with diff close msg, incase if there is a failure, but that's for later
	if err := CloseServiceWindow(obmondoAPICient, windowType); err != nil {
		log.Printf("unable to close the service window: %s", err)
		return
	}

	log.Println("Service window is closed now for this respective node")

	if err := CheckKernelAndRebootIfNeeded(*reboot); err != nil {
		log.Printf("unable to check kernel and reboot: %s", err)
		return
	}

}
