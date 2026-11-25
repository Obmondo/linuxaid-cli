package system

import (
	"strings"
	"testing"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/shell/shelltest"
)

// TestAptCommandsWaitForDpkgLock: a second apt run (a leftover timer, apt-daily, an operator)
// holding the dpkg lock must make the update wait, not fail at once.
func TestAptCommandsWaitForDpkgLock(t *testing.T) {
	runner := &shelltest.Recorder{}

	if err := UpdateSystem(runner, ConstDistributionNameUbuntu); err != nil {
		t.Fatalf("UpdateSystem returned %v", err)
	}

	for _, verb := range []string{" update", " upgrade", " autoremove", " clean"} {
		found := false
		for _, command := range runner.Commands() {
			if !strings.HasPrefix(command, "apt-get ") || !strings.Contains(command, verb) {
				continue
			}
			found = true
			if !strings.Contains(command, "-o DPkg::Lock::Timeout=600") {
				t.Errorf("%q does not wait for the dpkg lock", command)
			}
		}
		if !found {
			t.Errorf("no apt-get%s command ran, got %v", verb, runner.Commands())
		}
	}
}
