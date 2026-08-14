package webtee

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/helper"
	api "gitea.obmondo.com/EnableIT/linuxaid-cli/pkg/obmondo"
)

// Pipenames
const (
	pipeNameStdout = "stdout"
	pipeNameStderr = "stderr"
)

type Webtee struct {
	obmondoAPIURL string
	obmondoAPI    api.ObmondoClient
}

// failInstall reports the failure to Obmondo, logs it, and exits.
func (w *Webtee) failInstall(certname, msg string, err error) {
	// nolint: errcheck
	w.obmondoAPI.NotifyInstallScriptFailure(&api.InstallScriptInput{
		Certname: certname,
	})

	slog.Error(msg, slog.String("error", err.Error()))
	os.Exit(1)
}

func (w *Webtee) RemoteLogObmondo(command []string, certname string) {
	app := &application{
		config: WebTeeConfig{w.obmondoAPIURL, true, command, certname, false},
	}
	connectToServer(app)
	// nolint: errcheck
	defer app.conn.Close()

	lines := make(chan logLine)

	app.wg.Add(1)
	go webTee(app, lines)

	// Prepare the command and create pipes for stderr and stdout.
	cmd := exec.Command("/bin/bash", "-c", strings.Join(app.config.Command(), " "))

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		w.failInstall(certname, "failed to connect to stdout pipe", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		w.failInstall(certname, "failed to connect to stderr pipe", err)
	}

	// Start command execution.
	if err := cmd.Start(); err != nil {
		w.failInstall(certname, "failed to start command", err)
	}

	// For each line in stdout & stderr, wrap it in an "echo" command and send it to webtee server.
	var pipeWg sync.WaitGroup
	pipeWg.Add(1)
	go readPipe(stderr, lines, pipeNameStderr, &pipeWg)
	pipeWg.Add(1)
	go readPipe(stdout, lines, pipeNameStdout, &pipeWg)

	// Now wait for the pipes to finish reading & sending to lines channel.
	pipeWg.Wait()

	err = cmd.Wait()

	// Don't complain if the command being run is puppet agent and the exit status is a
	// puppet success code. Else, check the error and complain about the same.
	if err != nil && !shouldIgnorePuppetAgentError(command, cmd.ProcessState.ExitCode()) {
		slog.Debug("command execution failed", slog.String("command", strings.Join(command, " ")), slog.String("error", err.Error()))
		//nolint:forbidigo, errcheck
		w.obmondoAPI.NotifyInstallScriptFailure(&api.InstallScriptInput{
			Certname: certname,
		})

		os.Exit(1)
	}

	// Close the lines channel.
	close(lines)

	// Wait for goroutines (like the grpc stream) to finish.
	app.wg.Wait()
}

func shouldIgnorePuppetAgentError(command []string, exitCode int) bool {
	return strings.Contains(strings.Join(command, " "), "puppet agent") && helper.IsPuppetSuccessExitCode(exitCode)
}

// readPipe reads a pipe and sends every line to the lines channel, tagged with
// the pipe's name. It should always be run in a separate goroutine because
// we decrement wg waitgroup after execution.
func readPipe(pipe io.ReadCloser, lines chan logLine, pipeName string, wg *sync.WaitGroup) {
	defer wg.Done()
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		lines <- logLine{
			line: scanner.Text(),
			pipe: pipeName,
		}
	}
}

func NewWebtee(obmondoAPI api.ObmondoClient) *Webtee {
	u, _ := url.Parse(api.GetObmondoURL())

	return &Webtee{
		obmondoAPIURL: fmt.Sprintf("%s:443", u.Hostname()),
		obmondoAPI:    obmondoAPI,
	}
}
