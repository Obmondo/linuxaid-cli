package puppet

import (
	"fmt"
	"log"
	"strconv"

	"go-scripts/constants"
	"go-scripts/util"

	"github.com/bitfield/script"
)

const (
	MAILTO = constants.MAILTO
)

var (
	HOST      = util.GetHost()
	CUSTOMER  = util.GetCustomer()
	KEY_FILE  string
	CERT_FILE string
	HOSTNAME  string
)

func EnableAgent() bool {
	exitStatus := script.Exec("puppet agent --enable").ExitStatus()
	if exitStatus != 0 {
		log.Println("Failed To Enable Puppet")
		return false
	}

	log.Println("Successfully Enabled Puppet")
	return true
}

func DisableAgent(msg string) bool {
	cmdString := fmt.Sprintf("puppet agent --disable '%s'", msg)
	exitStatus := script.Exec(cmdString).ExitStatus()
	if exitStatus != 0 {
		log.Println("Failed To Disable Puppet")
		return false
	}

	log.Println("Successfully Disabled Puppet")
	return true
}

func RunPuppet() int {
	pipe := script.Exec("puppet agent -t --noop --detailed-exitcodes")
	exitStatus := pipe.ExitStatus()
	err := pipe.Error()
	if exitStatus != 0 {
		log.Println(err)
		return exitStatus
	}

	log.Println("Successfully Ran Puppet")
	return exitStatus
}

func RunPuppetWithRemoteLog() {
	log.Println("Contacting obmondo.com...")

	util.Remotelog("puppet agent -t --no-noop")
	exitStatus := util.Remotelog("puppet agent -t --no-noop").ExitStatus()

	// Puppet returns 0 on no changes, 1 on failures, 2 on successful run with
	// changes, and 4 or 6 if the run failed wholly or partially
	if exitStatus == 1 || exitStatus == 4 || exitStatus == 6 {
		puppetRunCount := 1

		for puppetRunCount <= 5 {
			util.Remotelog("puppet agent -t --no-noop")
			exitStatus = util.Remotelog("puppet agent -t --no-noop").ExitStatus()

			// 4 and 6 means more changes are pending, lets run again
			if exitStatus == 4 || exitStatus == 6 {
				puppetRunCount = puppetRunCount + 1
				log.Println("Running puppet agent " + strconv.Itoa(puppetRunCount) + " times now. last run exit code is " + strconv.Itoa(exitStatus))
			} else if exitStatus == 2 { // 2 means no more changes and a clean run, lets close the loop
				break
			} else if exitStatus == 1 { // 1 means puppet is failing to run somehow, lets exit and complain
				util.InstallFailed()
				script.Echo("Client install failed with code " + strconv.Itoa(exitStatus) + ". Contact EnableIT - " + MAILTO)
				break
			}
		}
	}

	log.Println("Installation succeeded. Please head to https://obmondo.com/server/" + HOSTNAME + " to continue configuration.")
}

func IsPuppetRunning() bool {
	exitStatus := script.IfExists("/opt/puppetlabs/puppet/cache/state/agent_catalog_run.lock").ExitStatus()
	if exitStatus == 0 {
		log.Println("Failed To Run Puppet")
		return false
	}

	return true
}

func IsPuppetInstalled() {
	if script.Exec("rpm -qa puppet-agent >/dev/null || dpkg -l puppet-agent >/dev/null").ExitStatus() == 0 {
		if KEY_FILE == "" {
			KEY_FILE = "/etc/puppetlabs/puppet/ssl/private_keys/" + HOST + "." + CUSTOMER + ".pem"
		}
		if CERT_FILE == "" {
			CERT_FILE = "/etc/puppetlabs/puppet/ssl/" + HOST + "." + CUSTOMER + ".pem"
		}
	}
}

func isPuppetDisabled() string {
	str, _ := script.Exec("puppet agent --configprint agent_disabled_lockfile").String()
	return str
}

func PuppetDisabled() bool {
	str := isPuppetDisabled()
	err := script.Exec("test -e " + str).Error()
	return err == nil
}

func PuppetDisabledNewInstall() bool {
	str := isPuppetDisabled()
	err := script.File(str).Match("Disabled by default on new or unconfigured old installations").Error()
	return err == nil
} 