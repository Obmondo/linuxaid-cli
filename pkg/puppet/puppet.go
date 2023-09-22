package puppet

import (
	"log"
	"fmt"

	"github.com/bitfield/script"
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
	log.Println("Successfully Disbaled Puppet")

	return true
}

func RunPuppet() int {
	exitStatus := script.Exec("puppet agent -t --noop --detailed-exitcodes").ExitStatus()
	if exitStatus != 0 {
		log.Println("Failed To Run Puppet")
		return exitStatus
	}
	log.Println("Successfully Ran Puppet")

	return exitStatus
}

func IsPuupetRunning() bool {
	exitStatus := script.IfExists("/opt/puppetlabs/puppet/cache/state/agent_catalog_run.lock").ExitStatus()
	if exitStatus == 0 {
		log.Println("Failed To Run Puppet")
		return false
	}
	
	return true
}