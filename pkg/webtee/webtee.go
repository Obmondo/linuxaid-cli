package webtee

import (
	"bufio"
	"io"
	"log"
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
	defer app.conn.Close()

	lines := make(chan logLine)

	app.wg.Add(1)
	go webTee(app, lines)

	// Prepare the command and create pipes for stderr and stdout.
	cmd := exec.Command("/bin/bash", "-c", strings.Join(app.config.Command(), " "))

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalln("Failed to connect to stdout pipe", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatalln("Failed to connect to stderr pipe", err)
	}

	// Start command execution.
	err = cmd.Start()
	if err != nil {
		log.Fatalln("Failed to start command", err)
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
		log.Println("Installation Setup Failed, please contact ops@obmondo.com")
		log.Println("Don't worry, Obmondo has the failed logs to analyze it")
		log.Fatal("Command exited with ", err)
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
