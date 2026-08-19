// Package shell is the single place where this codebase runs commands on the host. Everything
// that shells out goes through a Runner, so a test can substitute one instead of needing a real
// machine with apt, puppet and a kernel on it.
package shell

import (
	"github.com/bitfield/script"
)

// Result is the outcome of one command.
type Result struct {
	// Output holds the captured stdout, and is only set by Capture.
	Output string
	// ExitCode is the command's exit status.
	ExitCode int
	// Err is set when the command could not be run or exited non-zero.
	Err error
}

// Succeeded reports whether the command ran and exited zero.
func (r Result) Succeeded() bool {
	return r.Err == nil && r.ExitCode == 0
}

// Runner runs commands on the host. The three methods differ only in what happens to the
// command's output, which matters because some of these commands are long-running installs whose
// progress the operator watches.
type Runner interface {
	// Run streams the command's output to this process's stdout.
	Run(command string) Result
	// Capture returns the command's stdout instead of streaming it.
	Capture(command string) Result
	// Quiet runs the command and discards its output.
	Quiet(command string) Result
}

type scriptRunner struct{}

// New returns the Runner that actually executes commands on the host.
func New() Runner {
	return scriptRunner{}
}

func (scriptRunner) Run(command string) Result {
	pipe := script.Exec(command)
	_, err := pipe.Stdout()

	return Result{ExitCode: pipe.ExitStatus(), Err: err}
}

func (scriptRunner) Capture(command string) Result {
	pipe := script.Exec(command)
	out, err := pipe.String()

	return Result{Output: out, ExitCode: pipe.ExitStatus(), Err: err}
}

func (scriptRunner) Quiet(command string) Result {
	pipe := script.Exec(command)
	err := pipe.Wait()

	return Result{ExitCode: pipe.ExitStatus(), Err: err}
}
