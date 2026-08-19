package system

import (
	"path/filepath"
	"testing"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/shell"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/shell/shelltest"
)

func TestGetInstalledKernel(t *testing.T) {
	testBootDirectory, err := filepath.Abs("../../test/boot/")
	if err != nil {
		t.Fatal(err)
	}
	expectedKernelOutput := "6.11.0-3-generic"

	latestKernel, err := getInstalledKernel(shell.New(), testBootDirectory)
	if err != nil {
		t.Fatal(err)
	}
	if latestKernel != expectedKernelOutput {
		t.Errorf("\n expected: %s\n actual: %s", expectedKernelOutput, latestKernel)
		t.FailNow()
	}

}

// TestUpdateSystemRunsDistributionCommands pins which commands each distribution actually runs,
// which was untestable while every call shelled out directly.
func TestUpdateSystemRunsDistributionCommands(t *testing.T) {
	tests := []struct {
		distribution string
		want         string
	}{
		{distribution: ConstDistributionNameUbuntu, want: "apt-get --with-new-pkgs upgrade -y"},
		{distribution: ConstDistributionNameDebian, want: "apt-get --with-new-pkgs upgrade -y"},
		{distribution: ConstDistributionNameSLES, want: "zypper update -y"},
		{distribution: ConstDistributionNameRHEL, want: "yum update -y"},
		{distribution: ConstDistributionNameRocky, want: "yum update -y"},
	}

	for _, test := range tests {
		t.Run(test.distribution, func(t *testing.T) {
			runner := &shelltest.Recorder{}

			if err := UpdateSystem(runner, test.distribution); err != nil {
				t.Fatalf("UpdateSystem(%q) returned %v", test.distribution, err)
			}

			if !runner.Ran(test.want) {
				t.Errorf("expected %q to run %q, got %v", test.distribution, test.want, runner.Commands())
			}
		})
	}
}

// TestUpdateSystemReportsFailure covers the path where the package manager exits non-zero.
func TestUpdateSystemReportsFailure(t *testing.T) {
	runner := &shelltest.Recorder{
		Results: map[string]shell.Result{
			"apt-get --with-new-pkgs upgrade -y": {ExitCode: 100},
		},
	}

	if err := UpdateSystem(runner, ConstDistributionNameDebian); err == nil {
		t.Error("expected a failing apt-get upgrade to be reported, got nil")
	}
}

// TestUpdateSystemIgnoresUnknownDistribution keeps the existing behaviour: an unrecognised
// distribution is logged and skipped rather than treated as a failure.
func TestUpdateSystemIgnoresUnknownDistribution(t *testing.T) {
	runner := &shelltest.Recorder{}

	if err := UpdateSystem(runner, "plan9"); err != nil {
		t.Errorf("expected an unknown distribution to be skipped, got %v", err)
	}

	if len(runner.Commands()) != 0 {
		t.Errorf("expected no commands to run, got %v", runner.Commands())
	}
}
