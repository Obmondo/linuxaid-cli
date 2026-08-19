// Package shelltest provides a shell.Runner for tests: it records the commands it was asked to
// run and replies from a table instead of touching the host.
package shelltest

import (
	"strings"
	"sync"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/shell"
)

// Recorder is a shell.Runner that records commands and returns canned results.
type Recorder struct {
	// Results maps a command to the result it should produce. A command with no entry
	// succeeds with empty output, which keeps the happy path free of setup.
	Results map[string]shell.Result

	mu       sync.Mutex
	commands []string
}

// Commands returns the commands the runner was asked to run, in order.
func (r *Recorder) Commands() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]string(nil), r.commands...)
}

// Ran reports whether any recorded command contains the given substring.
func (r *Recorder) Ran(substring string) bool {
	for _, command := range r.Commands() {
		if strings.Contains(command, substring) {
			return true
		}
	}

	return false
}

func (r *Recorder) record(command string) shell.Result {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.commands = append(r.commands, command)

	return r.Results[command]
}

func (r *Recorder) Run(command string) shell.Result     { return r.record(command) }
func (r *Recorder) Capture(command string) shell.Result { return r.record(command) }
func (r *Recorder) Quiet(command string) shell.Result   { return r.record(command) }
