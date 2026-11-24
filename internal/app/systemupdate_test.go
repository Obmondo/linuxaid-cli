package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/shell"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/shell/shelltest"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/system"
)

// agentControlledRunner returns a recorder and the distribution commands for an Ubuntu host.
func agentControlledRunner(t *testing.T, results map[string]shell.Result) (*shelltest.Recorder, system.CertificateManagerCommands) {
	t.Helper()
	t.Setenv("ID", system.ConstDistributionNameUbuntu)

	runner := &shelltest.Recorder{Results: results}
	cmds, err := system.IsSupportedOS(runner)
	if err != nil {
		t.Fatalf("IsSupportedOS: %v", err)
	}

	return runner, cmds
}

// TestAgentControlledWritesResultAndNeverReboots: in a kubeaid-agent run the CLI updates the
// packages, writes whether a reboot is needed and leaves the reboot to people. The function
// takes no Obmondo client, so it cannot touch the service window.
func TestAgentControlledWritesResultAndNeverReboots(t *testing.T) {
	runner, cmds := agentControlledRunner(t, nil)
	resultFile := filepath.Join(t.TempDir(), "run", "node-update.json")

	if err := runAgentControlled(runner, cmds, system.ConstDistributionNameUbuntu, resultFile); err != nil {
		t.Fatalf("runAgentControlled: %v", err)
	}

	raw, err := os.ReadFile(resultFile)
	if err != nil {
		t.Fatalf("result file not written: %v", err)
	}
	var got system.RebootCheck
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("result file is not RebootCheck JSON: %v (%s)", err, raw)
	}
	for _, key := range []string{`"reboot_required"`, `"reason"`, `"running_kernel"`, `"installed_kernel"`} {
		if !strings.Contains(string(raw), key) {
			t.Errorf("result %s lacks %s", raw, key)
		}
	}

	if runner.Ran("reboot") {
		t.Errorf("agent-controlled run must never reboot, ran %v", runner.Commands())
	}
	if !runner.Ran("upgrade") {
		t.Errorf("expected the package upgrade to run, got %v", runner.Commands())
	}
}

// TestAgentControlledFailsOnAptError: kubeaid-agent stops its run on a non-zero exit, so an
// apt failure must surface as an error and leave no result behind.
func TestAgentControlledFailsOnAptError(t *testing.T) {
	results := map[string]shell.Result{}
	for _, command := range upgradeCommandsFor(t) {
		results[command] = shell.Result{ExitCode: 100}
	}
	runner, cmds := agentControlledRunner(t, results)
	resultFile := filepath.Join(t.TempDir(), "node-update.json")

	if err := runAgentControlled(runner, cmds, system.ConstDistributionNameUbuntu, resultFile); err == nil {
		t.Fatal("expected an apt failure to be returned")
	}
	if _, err := os.Stat(resultFile); !os.IsNotExist(err) {
		t.Errorf("expected no result file after a failed update, stat err = %v", err)
	}
}

// TestAgentControlledRequiresResultFile: without a result file kubeaid-agent cannot learn the
// outcome, so the run refuses to start rather than update blind.
func TestAgentControlledRequiresResultFile(t *testing.T) {
	runner, cmds := agentControlledRunner(t, nil)

	if err := runAgentControlled(runner, cmds, system.ConstDistributionNameUbuntu, ""); err == nil {
		t.Fatal("expected an error without a result file")
	}
	if len(runner.Commands()) != 0 {
		t.Errorf("expected nothing to run, got %v", runner.Commands())
	}
}

// upgradeCommandsFor returns the apt upgrade command UpdateSystem issues on Ubuntu.
func upgradeCommandsFor(t *testing.T) []string {
	t.Helper()
	probe := &shelltest.Recorder{}
	if err := system.UpdateSystem(probe, system.ConstDistributionNameUbuntu); err != nil {
		t.Fatalf("probe UpdateSystem: %v", err)
	}
	var out []string
	for _, command := range probe.Commands() {
		if strings.Contains(command, " upgrade") {
			out = append(out, command)
		}
	}
	return out
}
