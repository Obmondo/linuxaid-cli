package puppet

import (
	"fmt"
	"go-scripts/constants"
	"go-scripts/pkg/webtee"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bitfield/script"
	"github.com/schollz/progressbar/v3"
)

const (
	sleepTime = 5
)

var certName = os.Getenv("CERTNAME")

// 200 HTTP code
var closeWindowSuccessStatuses = map[int]struct{}{
	http.StatusOK: {},
}

// enable puppet-agent
func EnablePuppetAgent() bool {
	pipe := script.Exec("puppet agent --enable")
	pipe.Wait()
	exitStatus := pipe.ExitStatus()
	if exitStatus != 0 {
		log.Println("Failed To Enable Puppet")
		return false
	}

	log.Println("Successfully Enabled Puppet")
	return true
}

// disable puppet-agent with a msg
func DisablePuppetAgent(msg string) bool {
	cmdString := fmt.Sprintf("puppet agent --disable '%s'", msg)
	pipe := script.Exec(cmdString)
	pipe.Wait()
	exitStatus := pipe.ExitStatus()
	if exitStatus != 0 {
		log.Println("Failed To Disable Puppet")
		return false
	}

	log.Println("Successfully Disabled Puppet")
	return true
}

// run puppet-agent
func RunPuppetAgent(remoteLog bool, noopStatus string) int {
	cmdString := fmt.Sprintf("puppet agent -t --%s --detailed-exitcodes", noopStatus)
	if remoteLog {
		webtee.RemoteLogObmondo([]string{cmdString}, certName)
		return 0
	}

	log.Printf("Running puppet agent in '%s' mode", noopStatus)
	pipe := script.Exec(cmdString)
	_, err := pipe.Stdout()
	if err != nil {
		log.Println(err)
	}

	pipe.Wait()
	exitStatus := pipe.ExitStatus()
	log.Println("Completed puppet agent run")
	return exitStatus
}

// check if puppet agent is running or not
func isPuppetAgentRunning() bool {
	_, err := os.Stat(constants.AgentRunningLockFile)

	if err == nil {
		return true
	} else if os.IsNotExist(err) {
		return false
	}

	return true
}

// facter for new installation
func FacterNewSetup() {
	script.Exec("mkdir -p /etc/puppetlabs/facter/facts.d")

	// NOTE: not a fan, but quick and ugly one
	facter := `---
install_date: ` + time.Now().Format("20060102") + `
`

	_, err := script.Echo(facter).WriteFile(constants.ExternalFacterFile)
	if err != nil {
		errMsg := fmt.Sprintf("echo Can not create external facter file: %s ", err.Error())
		webtee.RemoteLogObmondo([]string{errMsg}, certName)
	}

}

// config setup for puppet-agent
func ConfigurePuppetAgent() {
	_, customerID, _ := strings.Cut(certName, ".")
	config := `[main]
server = ` + customerID + `.puppet.obmondo.com
certname = ` + certName + `
stringify_facts = false
masterport = 443

[agent]
report = true
pluginsync = true
noop = true
`
	_, err := script.Echo(config).WriteFile(constants.PuppetConfig)
	if err != nil {
		errMsg := fmt.Sprintf("echo Can not create puppet configuration file: %s ", err.Error())
		webtee.RemoteLogObmondo([]string{errMsg}, certName)
	}
}

// disable puppet-agent running as a service (sanity-check)
func DisablePuppetAgentService() {
	webtee.RemoteLogObmondo([]string{"puppet resource service puppet ensure=stopped enable=false"}, certName)
}

// If Puppet is already running we wait for up to 10 minutes before exiting.
//
// Note that if for some reason Puppet agent is running in daemon mode we'll end
// up here waiting for it to terminate, which will never happen. If that becomes
// an issue we might want to actively kill Puppet, but let's wait and see.
func WaitForPuppetAgent() {
	timeoutDuration := 600
	timeout := time.Now().Add(time.Duration(timeoutDuration) * time.Second)
	for isPuppetAgentRunning() {
		// time.Since calculates the time difference between time.Now() and the provided time in the argument.
		// Since we're comparing with a future time, the difference will be negative.
		// Hence, we'll timeout once the time difference becomes positive.
		if time.Since(timeout) >= 0 {
			log.Println("Puppet is running, aborting")
			// puppet kill/abort logic goes here
			break
		}

		time.Sleep(sleepTime * time.Second)
	}
}

// puppet-agent is already installed
func PuppetAgentIsInstalled() {
	bar := progressbar.Default(constants.BarProgressSize,
		"puppet-agent install...")

	// Just to have a nice progress bar
	tenErr := bar.Set(constants.BarSizeTen)
	if tenErr != nil {
		log.Println("failed to set the progressbar size")
	}

	time.Sleep(sleepTime * time.Millisecond)
	hundredErr := bar.Set(constants.BarSizeHundred)
	if hundredErr != nil {
		log.Println("failed to set the progressbar size")
	}

	finishErr := bar.Finish()
	if finishErr != nil {
		log.Println("failed to set the progressbar size")
	}

	webtee.RemoteLogObmondo([]string{"echo puppet-agent is already installed"}, certName)
}

// download puppet-agent and install it
func DownloadPuppetAgent(downloadPath string, url string) {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if _, exists := closeWindowSuccessStatuses[resp.StatusCode]; !exists {
		webtee.RemoteLogObmondo([]string{"echo puppet-agent debian file not present at this url"}, url)
	}

	f, _ := os.Create(downloadPath)
	defer f.Close()

	bar := progressbar.DefaultBytes(
		resp.ContentLength,
		"download puppet-agent..",
	)
	_, rerr := io.Copy(io.MultiWriter(f, bar), resp.Body)
	if rerr != nil {
		log.Fatal("downloading puppet-agent failed, exiting")
	}
}
