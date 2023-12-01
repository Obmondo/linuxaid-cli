package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/user"
	"regexp"
	"strconv"
	"strings"
	"time"

	constants "go-scripts/constants"
	api "go-scripts/pkg/obmondo_api"
	puppet "go-scripts/pkg/puppet"
	"go-scripts/util"

	"github.com/bitfield/script"
)

const (
	obmondoAPIURL     = constants.ObmondoAPIURL
	agentDisabledFile = constants.AgentDisabledLockFile
	path              = constants.PuppetPath
	sleepTime         = 5
)

func cleanup() {
	isEnabled := puppet.EnableAgent()
	if !isEnabled {
		log.Println("Not able to remove agent disable file")
	}
	log.Println("Ending Obmondo System Update Script")
}

func cleanupAndExit() {
	cleanup()
	os.Exit(0)
}

func GetIsServiceWindow(response []byte) string {
	var serviceWindow map[string]interface{}
	err := json.Unmarshal(response, &serviceWindow)
	if err != nil {
		log.Println("Failed to parse service window json")
	}
	isServiceWindow, ok := serviceWindow["data"].(string)
	if !ok {
		log.Println("Data field not found in reposne for service window")
		return ""
	}
	return strings.TrimSpace(isServiceWindow)
}

func GetServiceWindowStatus(obmondoAPICient api.ObmondoClient) bool {
	resp, err := obmondoAPICient.FetchServiceWindowStatus()
	if err != nil || resp == nil {
		log.Printf("unexpected error fetching service window url: %s", err)
		cleanupAndExit()
	}

	defer resp.Body.Close()

	statusCode, responseBody, err := util.ParseResponse(resp)
	if err != nil {
		log.Printf("unexpected error reading response body: %s", err)
		cleanupAndExit()
	}

	if statusCode != http.StatusOK {
		log.Printf("http status is not 200")
		cleanupAndExit()
	}

	serviceWindow := GetIsServiceWindow(responseBody)
	if serviceWindow == "" {
		return false
	}
	if serviceWindow == "yes" {
		return true
	}
	return false
}

func GetSystemDistribution() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		log.Println("Error reading os release file:", err)
		return ""
	}
	content := string(data)
	r := regexp.MustCompile(`(?:\n|^)NAME="([^"]+)"`)
	matches := r.FindStringSubmatch(content)

	if len(matches) > 1 {
		dist := strings.Trim(matches[1], "\"")
		return strings.TrimSpace(dist)
	}

	return ""
}

func CloseWindow(obmondoAPICient api.ObmondoClient) (*http.Response, error) {
	closeWindow, err := obmondoAPICient.CloseServiceWindow()
	if err != nil {
		log.Println("Failed to close service window", err)
		return nil, err
	}

	return closeWindow, err
}

func updateDebian() {
	log.Println("Running apt update/upgrade/autoremove")
	enverr := os.Setenv("DEBIAN_FRONTEND", "noninteractive")
	if enverr != nil {
		log.Fatal(enverr)
	}
	script.Exec("apt-get update").Wait()
	pipe := script.Exec("apt-get upgrade -y")
	_, err := pipe.Stdout()
	if err != nil {
		log.Fatal(err)
	}
	pipe.Wait()
	script.Exec("apt-get autoremove -y").Wait()
}

func getKernelForDebian() string {
	pipe := script.Exec("dpkg-query -Wf '${Installed-Size}\t${Package}\t${Status}\n'")
	installedKernel, _ := pipe.Exec("grep linux-image").Exec("grep installed").Exec("sort -nr").Exec("awk '{print $2}'").Exec("sed 's/linux-image-//g'").String()

	return installedKernel
}

func updateSUSE() {
	log.Println("Running zypper refresh/update")
	script.Exec("zypper refresh").Wait()
	pipe := script.Exec("zypper update -y")
	_, err := pipe.Stdout()
	if err != nil {
		log.Fatal(err)
	}
	pipe.Wait()
}

func getKernelForSUSE() string {
	installedKernel, _ := script.Exec("rpm -qa").Exec("grep kernel-default").Exec("sort -tr").Exec("sed 's/kernel-default-//g'").Exec("head -1").Exec("cut -d. -f1-3").Exec("sed 's/$/-default/g'").String()

	return installedKernel
}

func updateRedHat() {
	log.Println("Running yum repolist/update")
	script.Exec("yum repolist").Wait()
	pipe := script.Exec("yum update -y")
	_, err := pipe.Stdout()
	if err != nil {
		log.Fatal(err)
	}
	pipe.Wait()
	osVersion := GetRedHatVersion(GetOsVersion())

	if osVersion == "8" {
		script.Exec("yum remove $(yum repoquery --installonly --latest-limit=-3 -q)")
	} else {
		script.Exec("package-cleanup --oldkernels --count=2 -y")
	}
}

func getKernelForRedHat() string {
	pipe := script.Exec("yum history package-info kernel")
	installedKernel, _ := pipe.Exec("grep '^Package '").Exec("head -n 1").Exec("sed 's/P.*:.*kernel-//g'").String()

	return installedKernel
}

func GetOsVersion() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		log.Println("Error reading os release file:", err)
		return ""
	}
	content := string(data)
	r := regexp.MustCompile(`(?:\n|^)VERSION_ID="([^"]+)"`)
	matches := r.FindStringSubmatch(content)

	if len(matches) > 1 {
		dist := strings.Trim(matches[1], "\"")
		return strings.TrimSpace(dist)
	}

	return ""
}

func GetRedHatVersion(osVersion string) string {
	if strings.Contains(osVersion, "8") {
		version := strings.Split(osVersion, ".")
		return version[0]
	} else {
		return osVersion
	}
}

func GetInstalledKernel(distribution string) string {
	switch distribution {
	case "Ubuntu", "Debian":
		updateDebian()
		return getKernelForDebian()
	case "SUSE", "openSUSE", "SLES":
		updateSUSE()
		return getKernelForSUSE()
	case "CentOS", "Red Hat Enterprise Linux Server", "Red Hat Enterprise Linux":
		updateRedHat()
		return getKernelForRedHat()
	default:
		log.Println("Unknown distribution")
		return ""
	}
}

func waitForPuppet() {
	timeout := 600
	for {
		isPuppetRunning := puppet.IsPuppetRunning()

		if !isPuppetRunning {
			break
		}

		if timeout <= 0 {
			log.Println("Puppet is running, aborting")
			cleanupAndExit()
		}

		timeout -= 5
		time.Sleep(sleepTime * time.Second)
	}
}

func handlePuppetRun() {
	// NOTE: Added to avoid magic number issue with puppet exit codes
	//nolint:all
	var puppetExitCodes = map[string]int{
		"two":  2,
		"four": 4,
		"six":  6,
	}
	exitCode := puppet.RunPuppet("noop")

	switch exitCode {
	case 0, puppetExitCodes["two"]:
		log.Println("Everything is fine with puppet agent run, let's continue.")
		return
	case 1:
		log.Println("Puppet run failed, or wasn't attempted due to another run already in progress.")
		os.Exit(0)
	case puppetExitCodes["four"], puppetExitCodes["six"]:
		log.Println("Puppet has pending changes, aborting.")
		os.Exit(0)
	default:
		log.Println("Puppet failed with exit code", strconv.Itoa(exitCode), ", aborting.")
		os.Exit(0)
	}
}

func closeServiceWindow(obmondoAPICient api.ObmondoClient) {
	closeWindow, err := CloseWindow(obmondoAPICient)
	if err != nil {
		cleanupAndExit()
	}
	defer closeWindow.Body.Close()

	// 202 -> When a certname says it's done but the overall window is not auto-closed
	// 204 -> When a certname says it's done AND the overall window is auto-closed
	// 208 -> When any of the above requests happen again and again
	if closeWindow.StatusCode != http.StatusAccepted || closeWindow.StatusCode != http.StatusNoContent || closeWindow.StatusCode != http.StatusAlreadyReported {
		log.Println("Failed to close Service Window, window 3")
		cleanupAndExit()
	}
}

func checkUser() {
	user, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	if user.Username == "root" {
		return
	}
	log.Fatal("exiting, obmondo-system-update script needs to be run as root, current user is ", user.Username)
}

func main() {
	checkUser()

	log.Println("Starting Obmondo System Update Script")

	// check if agent disable file exists
	if _, err := os.Stat(agentDisabledFile); err == nil {
		log.Println("Puppet has been disabled, exiting")
		os.Exit(0)
	}

	// assuming that clean up will not be done if the script fails
	defer cleanup()

	// If Puppet is already running we wait for up to 10 minutes before exiting.
	//
	// Note that if for some reason Puppet agent is running in daemon mode we'll end
	// up here waiting for it to terminate, which will never happen. If that becomes
	// an issue we might want to actively kill Puppet, but let's wait and see.
	distribution := GetSystemDistribution()
	if distribution == "" {
		cleanupAndExit()
	}

	obmondoAPICient := api.NewObmondoClient()
	isServiceWindow := GetServiceWindowStatus(obmondoAPICient)

	if !isServiceWindow {
		// lets fail with exit 0, otherwise systemd service will be in failed status
		log.Println("Exiting, Service window is NOT active")
		os.Exit(0)
	}

	log.Println("Service window is active, going ahead")

	// Check if any existing puppet agent is already running
	waitForPuppet()

	// Run puppet-agent and check the exit code, and exit this script, if it's not 0 or 2
	handlePuppetRun()

	// Disable puppet-agent, since we'll be running upgrade commands
	puppet.DisableAgent("Puppet has been disabled by the obmondo-system-update script.")

	// Get installed kernel of the system
	installedKernel := GetInstalledKernel(distribution)
	if installedKernel == "" {
		log.Println("Looks like no kernel is installed on the node")
	}

	// Close the service window
	// we need to close it with diff close msg, incase if there is a failure, but that's for later
	closeServiceWindow(obmondoAPICient)
	log.Println("Service window is closed now for this respective node")

	// If kernel is installed, then only we will try to reboot.
	// in lxc kernel wont be present
	if installedKernel != "" {
		// Get running kernel of the system
		runningKernel, err := script.Exec("uname -r").String()
		if err != nil {
			log.Println("Failed to fetch Running Kernel")
			cleanupAndExit()
		}

		// Reboot the node, if we have installed a new kernel
		if installedKernel != runningKernel {
			log.Println("Rebooting server")
			script.Exec("reboot --force")
		}
	}
}
