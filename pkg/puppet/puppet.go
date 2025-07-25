package puppet

import (
	"crypto/tls"
	"errors"
	"fmt"
	"go-scripts/constants"
	"go-scripts/pkg/webtee"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bitfield/script"
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
	if err := pipe.Wait(); err != nil {
		slog.Error("failed to enable puppet agent", slog.String("error", err.Error()))
	}
	exitStatus := pipe.ExitStatus()
	if exitStatus != 0 {
		slog.Error("failed To enable puppet")
		return false
	}

	slog.Info("successfully enabled puppet")
	return true
}

// disable puppet-agent with a msg
func DisablePuppetAgent(msg string) bool {
	cmdString := fmt.Sprintf("puppet agent --disable '%s'", msg)
	pipe := script.Exec(cmdString)
	if err := pipe.Wait(); err != nil {
		slog.Error("failed to disable puppet agent", slog.String("error", err.Error()))
	}
	exitStatus := pipe.ExitStatus()
	if exitStatus != 0 {
		slog.Error("failed To disable puppet")
		return false
	}

	slog.Info("successfully disabled puppet")
	return true
}

// run puppet-agent
func RunPuppetAgent(remoteLog bool, noopStatus string) int {
	cmdString := fmt.Sprintf("puppet agent -t --%s --detailed-exitcodes", noopStatus)
	if remoteLog {
		webtee.RemoteLogObmondo([]string{cmdString}, certName)
		return 0
	}

	slog.Info("running puppet agent in", slog.String("mode", noopStatus))
	pipe := script.Exec(cmdString)
	_, err := pipe.Stdout()
	if err != nil {
		slog.Error(err.Error())
	}

	if err := pipe.Wait(); err != nil {
		slog.Error("completed puppet agent run", slog.String("error", err.Error()))
	}
	exitStatus := pipe.ExitStatus()
	slog.Info("completed puppet agent run")
	return exitStatus
}

// check if puppet agent is running or not
func isPuppetAgentRunning() bool {
	_, err := os.Stat(constants.AgentRunningLockFile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			slog.Debug("puppet agent lock file not found", slog.String("lock_file", constants.AgentRunningLockFile))
			webtee.RemoteLogObmondo([]string{"echo unable to find puppet agent lock file"}, certName)
			return false
		}

		slog.Debug("error checking puppet agent lock file", slog.String("lock_file", constants.AgentRunningLockFile), slog.Any("error", err))
		webtee.RemoteLogObmondo([]string{"echo unable to fetch puppet agent lock file details"}, certName)
		return false
	}

	return true
}

// facter for new installation
func FacterNewSetup() {
	script.Exec("mkdir -p /etc/puppetlabs/facter/facts.d")

	currentTime := time.Now()
	facter := fmt.Sprintf("---\ninstall_date: %d%d%d\n", currentTime.Year(), currentTime.Month(), currentTime.Day())

	_, err := script.Echo(facter).WriteFile(constants.ExternalFacterFile)
	if err != nil {
		slog.Debug("failed to write external facter file", slog.String("file_path", constants.ExternalFacterFile), slog.Any("error", err))
		errMsg := fmt.Sprintf("echo can not create external facter file: %s ", err.Error())
		webtee.RemoteLogObmondo([]string{errMsg}, certName)
	}

}

// config setup for puppet-agent
func ConfigurePuppetAgent() {
	_, customerID, _ := strings.Cut(certName, ".")

	puppetURL := fmt.Sprintf("https://%s.puppet.obmondo.com/status/v1/services", customerID)
	tlsConfigTransport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	httpClient := &http.Client{
		Transport: tlsConfigTransport,
		// nolint: mnd
		Timeout: 5 * time.Second,
	}
	resp, err := httpClient.Get(puppetURL)
	if err != nil {
		slog.Debug("failed to check puppet domain status", slog.String("puppet_url", puppetURL), slog.Any("error", err))
		errMsg := fmt.Sprintf("echo failed to reach Puppet server: %s", err.Error())
		webtee.RemoteLogObmondo([]string{errMsg}, certName)
		os.Exit(1)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		customerID = constants.DefaultPuppetServerCustomerID
	}

	configFmt := `[main]
server = %s.puppet.obmondo.com
certname = %s
stringify_facts = false
masterport = 443

[agent]
report = true
pluginsync = true
noop = true
environment = master
`
	_, err = script.Echo(fmt.Sprintf(configFmt, customerID, certName)).WriteFile(constants.PuppetConfig)
	if err != nil {
		slog.Debug("failed to configure puppet agent", slog.Any("error", err))
		errMsg := fmt.Sprintf("echo can not create puppet configuration file: %s ", err.Error())
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
			slog.Warn("puppet is running, aborting")
			// puppet kill/abort logic goes here
			break
		}

		time.Sleep(sleepTime * time.Second)
	}
}

// download puppet-agent and install it
func DownloadPuppetAgent(downloadPath string, url string) {
	resp, err := http.Get(url)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	defer resp.Body.Close()

	if _, exists := closeWindowSuccessStatuses[resp.StatusCode]; !exists {
		slog.Debug("puppet agent download failed", "url", url)
		webtee.RemoteLogObmondo([]string{"echo puppet-agent debian file not present at this url"}, url)
		os.Exit(1)
	}

	f, err := os.Create(downloadPath)
	if err != nil {
		slog.Error("failed to create file", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer f.Close()

	_, rerr := io.Copy(io.MultiWriter(f), resp.Body)
	if rerr != nil {
		slog.Error("downloading puppet-agent failed, exiting")
		os.Exit(1)
	}
}
