package main

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	constants "go-scripts/contants"
	api "go-scripts/pkg/obmondo_api"
	puppet "go-scripts/pkg/puppet"
	"go-scripts/util"

	"github.com/bitfield/script"
)

const (
	obmondoAPIURL     = constants.OBMONDO_API_URL
	agentDisabledFile = constants.AGENT_DISABLED_FILE
	path              = constants.PUPPET_PATH
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

func GetCommonNameFromCertFile(certPath string) string {
	hostCert, err := os.ReadFile(certPath)
	if err != nil {
		log.Printf("Failed to fetch hostcert: %s", err)
		return ""
	}

	block, _ := pem.Decode(hostCert)
	if block == nil {
		log.Printf("Failed to decode hostcert: %s", err)
		return ""
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		log.Printf("Failed to parse hostcert: %s", err)
		return ""
	}

	return cert.Subject.CommonName
}

func GetCustomerId(certname string) string {
	parts := strings.Split(certname, ".")
	if len(parts) < 2 {
		log.Println("Incorrect format for certname")
		return ""
	}
	return parts[1]
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

func GetServiceWindowStatus(obmondoApiCient api.ObmondoClient) bool {
	resp, err := obmondoApiCient.FetchServiceWindowStatus()
	if err != nil || resp == nil {
		log.Printf("unexpected error fetching service window url: %s", err)
		cleanupAndExit()
	}
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

func GetSytemDistribution() string {
	pipe := script.Exec("cat /etc/os-release")
	pipe = pipe.Exec("grep ^Name -i")
	pipe = pipe.Exec("cut -d '=' -f2")
	output, err := pipe.Exec("tr -d '\"'").String()
	if err != nil {
		fmt.Println("Error executing command:", err)
		return ""
	}

	return strings.TrimSpace(output)
}

func CloseWidow(obmondoApiCient api.ObmondoClient) (*http.Response, error) {
	close_window, err := obmondoApiCient.CloseServiceWindow()
	if err != nil {
		log.Println("Failed to close service window")
		return nil, err
	}

	return close_window, err
}

func RunPuppet() int {
	return script.Exec("puppet agent -t --noop --detailed-exitcodes").ExitStatus()
}

func updateDebian() *script.Pipe {
	pipe := script.Exec("export DEBIAN_FRONTEND=noninteractive")
	pipe = pipe.Exec("apt-get update")
	pipe = pipe.Exec("apt-get upgrade -y")
	pipe.Exec("apt-get autoremove -y")

	return pipe
}

func getKernelForDebian(pipe *script.Pipe) string {
	pipe = pipe.Exec("dpkg-query -Wf '${Installed-Size}\t${Package}\t${Status}\n'")
	installedKernel, _ := pipe.Exec("grep linux-image").Exec("grep installed").Exec("sort -nr").Exec("awk '{print $2}'").Exec("sed 's/linux-image-//g'").String()

	return installedKernel
}

func updateSUSE() *script.Pipe {
	pipe := script.Exec("zypper refresh")
	pipe.Exec("zypper update -y")

	return pipe
}

func getKernelForSUSE(pipe *script.Pipe) string {
	installedKernel, _ := pipe.Exec("rpm -qa").Exec("grep kernel-default").Exec("sort -tr").Exec("sed 's/kernel-default-//g'").Exec("head -1").Exec("cut -d. -f1-3").Exec("sed 's/$/-default/g'").String()

	return installedKernel
}

func updateCentOS() *script.Pipe {
	pipe := script.Exec("yum clean metadata")
	pipe = pipe.Exec("yum update -y")
	pipe.Exec("package-cleanup --oldkernels --count=2 -y")

	return pipe
}

func getKernelForCentOS(pipe *script.Pipe) string {
	pipe = pipe.Exec("yum history package-info kernel")
	installedKernel, _ := pipe.Exec("grep '^Package '").Exec("head -n 1").Exec("sed 's/P.*:.*kernel-//g'").String()

	return installedKernel
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
		pipe := updateCentOS()
		return getKernelForCentOS(pipe)
	default:
		log.Println("Unknown distribution")
		return ""
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
	distribution := GetSytemDistribution()
	if distribution == "" {
		cleanupAndExit()
	}
	obmondoApiCient := api.NewObmondoClient()
	is_service_window := GetServiceWindowStatus(obmondoApiCient)

	if is_service_window {
		timeout := 600
		var puppet_clean int
		for {
			// _, err := script.Exec("pgrep -f /opt/puppetlabs/puppet/bin/puppet agent").String()
			isPuppetRunning := puppet.IsPuupetRunning()

			// if err == nil {
			if isPuppetRunning {
				if timeout <= 0 {
					log.Println("Puppet is running, aborting")
					cleanup()
					os.Exit(1)
				}

				timeout -= 5
				time.Sleep(5 * time.Second)
			} else {
				break
			}
		}

		exitCode := puppet.RunPuppet()
		puppet.DisableAgent("Puppet has been disabled by the systme update script.")

		switch exitCode {
		case 0, 2:
			log.Println("Everything is fine, let's continue.")
			puppet_clean = 1
		case 4, 6:
			log.Println("Puppet has pending changes, aborting.")
			return
		default:
			log.Println("Puppet failed with exit code", strconv.Itoa(exitCode), ", aborting.")
			return
		}

		installedKernel := GetInstalledKernel(distribution)
		if installedKernel == "" {
			cleanupAndExit()
		}

		close_window, err := CloseWidow(obmondoApiCient)
		if err != nil {
			log.Println("Failed to close Service Window")
			cleanupAndExit()
		}
		if close_window.StatusCode != 200 {
			log.Println("Failed to close Service Window")
			cleanupAndExit()
		}

		running_kernel, err := script.Exec("uname -r").String()
		if err != nil {
			log.Println("Failed to fetch Running Kernel")
			cleanupAndExit()
		}

		if installedKernel != running_kernel {
			log.Println("Rebooting server")
			script.Exec("reboot --force")
		}

		cleanup()

		if puppet_clean != 0 {
			exitCode := puppet.RunPuppet()
			log.Println("Puppet exited with exit code", exitCode)
		}

		// restart mode=automatic, batch mode, set DEBIAN_FRONTEND to noninteractive
		// FIXME
		// needrestart -ra -b -f noninteractive
	}
}
