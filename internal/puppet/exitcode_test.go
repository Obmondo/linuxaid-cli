package puppet

import (
	"testing"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/system"
)

func TestSucceeded(t *testing.T) {
	tests := []struct {
		name     string
		osID     string
		exitCode int
		expected bool
	}{
		{name: "no changes", exitCode: ExitNoChanges, expected: true},
		{name: "changes", exitCode: ExitChanged, expected: true},
		{name: "changes and failed resources", exitCode: ExitChangedAndFailed, expected: true},
		{name: "a failed run", exitCode: ExitFailed, expected: false},
		{name: "failed resources", exitCode: ExitResourcesFailed, expected: false},
		{name: "failed resources on TurrisOS", osID: system.ConstDistributionNameTurrisOS, exitCode: ExitResourcesFailed, expected: true},
		{name: "a failed run on TurrisOS", osID: system.ConstDistributionNameTurrisOS, exitCode: ExitFailed, expected: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("ID", test.osID)

			if got := Succeeded(test.exitCode); got != test.expected {
				t.Errorf("expected %v for exit code %d, got %v", test.expected, test.exitCode, got)
			}
		})
	}
}
