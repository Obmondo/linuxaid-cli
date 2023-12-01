package puppet

import (
	"fmt"
	"log"
	"os"

	"github.com/bitfield/script"

	constants "go-scripts/constants"
)

const (
	agentRunningLockFile = constants.AgentRunningLockFile
)

func EnableAgent() bool {
	p := script.Exec("puppet agent --enable")
	p.Wait()
	if p.ExitStatus() != 0 {
		log.Println("Failed To Enable Puppet")
		return false
	}
	log.Println("Successfully Enabled Puppet")

	return true
}

func DisableAgent(msg string) bool {
	cmdString := fmt.Sprintf("puppet agent --disable '%s'", msg)
	p := script.Exec(cmdString)
	p.Wait()
	if p.ExitStatus() != 0 {
		log.Println("Failed To Disable Puppet")
		return false
	}
	log.Println("Successfully Disabled Puppet")

	return true
}

func RunPuppet(noopStatus string) int {
	log.Printf("Running puppet agent in '%s' mode", noopStatus)
	cmdString := fmt.Sprintf("puppet agent -t --%s --detailed-exitcodes", noopStatus)
	p := script.Exec(cmdString)
	_, err := p.Stdout()
	if err != nil {
		log.Println(err)
	}
	p.Wait()
	exitStatus := p.ExitStatus()
	log.Println("Completed puppet agent run")
	return exitStatus
}

// check if puppet agent is running or not
func IsPuppetRunning() bool {
	_, err := os.Stat(agentRunningLockFile)

	if err == nil {
		log.Println("Puppet is already running or stuck, please check it manually")
		return true
	} else if os.IsNotExist(err) {
		log.Println("Puppet agent is not running currently, great")
		return false
	}

	return true
}
