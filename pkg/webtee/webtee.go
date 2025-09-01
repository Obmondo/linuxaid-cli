package webtee

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// Pipenames
const (
	pipeNameStdout = "stdout"
	pipeNameStderr = "stderr"
)

func RemoteLogObmondo(command []string, certname string) {
	app := &application{
		config: WebTeeConfig{"api.obmondo.com:443", true, command, certname, false},
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
		slog.Error("failed to connect to stdout pipe", slog.String("error", err.Error()))
		os.Exit(1)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		slog.Error("failed to connect to stderr pipe", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Start command execution.
	err = cmd.Start()
	if err != nil {
		slog.Error("failed to start command", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// For each line in stdout & stderr, wrap it in an "echo" command and send it to webtee server.
	var pipeWg sync.WaitGroup
	pipeWg.Add(1)
	go readPipe(stderr, lines, false, &pipeWg)
	pipeWg.Add(1)
	go readPipe(stdout, lines, true, &pipeWg)

	// Now wait for the pipes to finish reading & sending to lines channel.
	pipeWg.Wait()

	err = cmd.Wait()
	if err != nil {
		slog.Debug("command execution failed", slog.String("command", strings.Join(command, " ")), slog.String("error", err.Error()))
		//nolint:forbidigo
		fmt.Printf("\nInstallation setup failed, please contact ops@obmondo.com\nDon't worry, obmondo has the failed logs to analyze it.\n") //nolint:revive
		os.Exit(1)
	}

	// Close the lines channel.
	close(lines)

	// Wait for goroutines (like the grpc stream) to finish.
	app.wg.Wait()
}

// readPipe reads a pipe, wraps every line in an "echo" command, prints it, and sends the line to
// the lines channel. It should always be run in a separate goroutine because
// we decrement wg waitgroup after execution.
func readPipe(pipe io.ReadCloser, lines chan logLine, isStdout bool, wg *sync.WaitGroup) {
	defer wg.Done()
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		m := scanner.Text()
		if isStdout {
			lines <- logLine{
				line: m,
				pipe: pipeNameStdout,
			}
		} else {
			lines <- logLine{
				line: m,
				pipe: pipeNameStderr,
			}
		}
	}
}
