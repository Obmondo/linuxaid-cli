package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
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
	agentDisabledFile = constants.AgentDisabledFile
	path              = constants.PuppetPath
	sleepTime         = 5
)

func cleanup() {
	isEnabled := puppet.EnableAgent()
	if !isEnabled {
		log.Println("Not able to remove agent disable file")
	}
}

func cleanupAndExit() {
	cleanup()
	os.Exit(1)
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

func CloseWidow(obmondoAPICient api.ObmondoClient) (*http.Response, error) {
	closeWindow, err := obmondoAPICient.CloseServiceWindow()
	if err != nil {
		log.Println("Failed to close service window")
		return nil, err
	}

	return closeWindow, err
}

func RunPuppet() int {
	return script.Exec("puppet agent -t --noop --detailed-exitcodes").ExitStatus()
}

func updateDebian() *script.Pipe {
	pipe := script.Exec("export DEBIAN_FRONTEND=noninteractive")
	pipe = pipe.Exec("apt-get update")
	pipe = pipe.Exec("apt-get upgrade -y")
	pipe = pipe.Exec("apt-get autoremove -y")

	return pipe
}

func getKernelForDebian(pipe *script.Pipe) string {
	pipe = pipe.Exec("dpkg-query -Wf '${Installed-Size}\t${Package}\t${Status}\n'")
	installedKernel, _ := pipe.Exec("grep linux-image").Exec("grep installed").Exec("sort -nr").Exec("awk '{print $2}'").Exec("sed 's/linux-image-//g'").String()

	return installedKernel
}

func updateSUSE() *script.Pipe {
	pipe := script.Exec("zypper refresh")
	pipe = pipe.Exec("zypper update -y")

	return pipe
}

func getKernelForSUSE(pipe *script.Pipe) string {
	installedKernel, _ := pipe.Exec("rpm -qa").Exec("grep kernel-default").Exec("sort -tr").Exec("sed 's/kernel-default-//g'").Exec("head -1").Exec("cut -d. -f1-3").Exec("sed 's/$/-default/g'").String()

	return installedKernel
}

func updateCentOS(osVersion string) *script.Pipe {
	pipe := script.Exec("yum clean metadata")
	pipe = pipe.Exec("yum update -y")
	if osVersion == "8" {
		pipe = pipe.Exec("yum remove $(yum repoquery --installonly --latest-limit=-3 -q)")
	} else {
		pipe = pipe.Exec("package-cleanup --oldkernels --count=2 -y")
	}

	return pipe
}

func getKernelForCentOS(pipe *script.Pipe) string {
	pipe = pipe.Exec("yum history package-info kernel")
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

func GetCentOsVersion(osVersion string) string {
	if strings.Contains(osVersion, "8") {
		vers := strings.Split(osVersion, ".")
		return vers[0]
	} else {
		return osVersion
	}
}

func GetInstalledKernel(distribution string) string {
	switch distribution {
	case "Ubuntu", "Debian":
		pipe := updateDebian()
		return getKernelForDebian(pipe)
	case "SUSE", "openSUSE":
		pipe := updateSUSE()
		return getKernelForSUSE(pipe)
	case "CentOS", "RedHat":
		osVersion := GetOsVersion()
		osVersion = GetCentOsVersion(osVersion)
		pipe := updateCentOS(osVersion)
		return getKernelForCentOS(pipe)
	default:
		log.Println("Unknown distribution")
		return ""
	}
}

func waitForPuppet() {
	timeout := 600
	for {
		isPuppetRunning := puppet.IsPuupetRunning()

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

func handlePuppetRun(puppetClean *int) {
	// NOTE: Added to avoid magic number issue with puppet exit codes
	//nolint:all
	var puppetExitCodes = map[string]int{
		"two":  2,
		"four": 4,
		"five": 5,
		"six":  6,
	}
	exitCode := puppet.RunPuppet()
	puppet.DisableAgent("Puppet has been disabled by the systme update script.")

	switch exitCode {
	case 0, puppetExitCodes["two"]:
		log.Println("Everything is fine, let's continue.")
		*puppetClean = 1
	case puppetExitCodes["four"], puppetExitCodes["six"]:
		log.Println("Puppet has pending changes, aborting.")
		return
	default:
		log.Println("Puppet failed with exit code", strconv.Itoa(exitCode), ", aborting.")
		return
	}
}

func closeServiceWindow(obmondoAPICient api.ObmondoClient) {
	closeWindow, err := CloseWidow(obmondoAPICient)
	if err != nil {
		log.Println("Failed to close Service Window")
		cleanupAndExit()
	}
	defer closeWindow.Body.Close()
	if closeWindow.StatusCode != http.StatusOK {
		log.Println("Failed to close Service Window")
		cleanupAndExit()
	}
}

func main() {
	log.Println("Starting Obmondo System Update Srcipt")

	// check if agent disable file exists
	if _, err := os.Stat(agentDisabledFile); err == nil {
		log.Println("Puppet has been disabled, exiting")
		os.Exit(1)
	}

	// script.IfExists(agentDisabledFile).Echo("Puppet has been disabled, exiting").Stdout()

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

	if isServiceWindow {
		var puppetClean int
		waitForPuppet()
		handlePuppetRun(&puppetClean)

		installedKernel := GetInstalledKernel(distribution)
		if installedKernel == "" {
			cleanupAndExit()
		}

		closeServiceWindow(obmondoAPICient)
		runningKernel, err := script.Exec("uname -r").String()
		if err != nil {
			log.Println("Failed to fetch Running Kernel")
			cleanupAndExit()
		}

		if installedKernel != runningKernel {
			log.Println("Rebooting server")
			script.Exec("reboot --force")
		}

		cleanup()

		if puppetClean != 0 {
			exitCode := puppet.RunPuppet()
			log.Println("Puppet exited with exit code", exitCode)
		}

		// restart mode=automatic, batch mode, set DEBIAN_FRONTEND to noninteractive
		// commented here to be fixed
		// needrestart -ra -b -f noninteractive
	}
}
