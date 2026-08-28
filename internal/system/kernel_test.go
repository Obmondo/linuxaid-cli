package system

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/shell"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/shell/shelltest"
)

func TestGetInstalledKernel(t *testing.T) {
	testBootDirectory, err := filepath.Abs("../../test/boot/")
	if err != nil {
		t.Fatal(err)
	}
	expectedKernelOutput := "6.11.0-3-generic"

	latestKernel, err := getInstalledKernel(testBootDirectory)
	if err != nil {
		t.Fatal(err)
	}
	if latestKernel != expectedKernelOutput {
		t.Errorf("\n expected: %s\n actual: %s", expectedKernelOutput, latestKernel)
		t.FailNow()
	}
}

// TestGetInstalledKernelPicksNewestRHELKernel is the regression test for the bug where
// "find /boot/vmlinuz-* | sort -V | tail -1" returned the RHEL point release
// "4.18.0-553.el8_10" as newest and ranked every z-stream update below it, so a node booted
// on the point release never saw a reason to reboot.
func TestGetInstalledKernelPicksNewestRHELKernel(t *testing.T) {
	bootDir := writeBootDir(t,
		"4.18.0-553.el8_10.x86_64",
		"4.18.0-553.137.1.el8_10.x86_64",
		"4.18.0-553.157.1.el8_10.x86_64",
		"0-rescue-0123456789abcdef0123456789abcdef",
	)

	got, err := getInstalledKernel(bootDir)
	if err != nil {
		t.Fatal(err)
	}
	if want := "4.18.0-553.157.1.el8_10.x86_64"; got != want {
		t.Errorf("getInstalledKernel() = %q, want %q", got, want)
	}
}

func TestGetInstalledKernelEmptyBootDir(t *testing.T) {
	got, err := getInstalledKernel(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("getInstalledKernel() = %q, want empty string", got)
	}
}

func TestCheckKernelAndRebootIfNeeded(t *testing.T) {
	origBoot, origGrace := bootDirectory, rebootGracePeriod
	t.Cleanup(func() {
		bootDirectory = origBoot
		rebootGracePeriod = origGrace
	})
	rebootGracePeriod = time.Millisecond

	rhelBoot := []string{"4.18.0-553.el8_10.x86_64", "4.18.0-553.157.1.el8_10.x86_64"}

	t.Run("reboots when a newer kernel is installed", func(t *testing.T) {
		bootDirectory = writeBootDir(t, rhelBoot...)
		runner := &shelltest.Recorder{Results: map[string]shell.Result{
			"uname -r": {Output: "4.18.0-553.el8_10.x86_64\n"},
		}}

		// The fake runner never brings the machine down, so reboot reports that the grace
		// period elapsed. What matters is that the reboot command was issued.
		if err := CheckKernelAndRebootIfNeeded(runner, false); err == nil {
			t.Error("expected an error because the fake runner does not reboot the machine")
		}
		if !runner.Ran(rebootCommand) {
			t.Errorf("expected %q to run, got %v", rebootCommand, runner.Commands())
		}
	})

	t.Run("does not reboot when the running kernel is already newest", func(t *testing.T) {
		bootDirectory = writeBootDir(t, rhelBoot...)
		runner := &shelltest.Recorder{Results: map[string]shell.Result{
			"uname -r": {Output: "4.18.0-553.157.1.el8_10.x86_64\n"},
		}}

		if err := CheckKernelAndRebootIfNeeded(runner, false); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if runner.Ran("reboot") {
			t.Errorf("expected no reboot, got %v", runner.Commands())
		}
	})

	t.Run("does not reboot when reboot is disabled", func(t *testing.T) {
		bootDirectory = writeBootDir(t, rhelBoot...)
		runner := &shelltest.Recorder{Results: map[string]shell.Result{
			"uname -r": {Output: "4.18.0-553.el8_10.x86_64\n"},
		}}

		if err := CheckKernelAndRebootIfNeeded(runner, true); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if runner.Ran("reboot") {
			t.Errorf("expected no reboot when noReboot is set, got %v", runner.Commands())
		}
	})
}

// writeBootDir creates a temp directory holding an empty vmlinuz-<version> file per version.
func writeBootDir(t *testing.T, versions ...string) string {
	t.Helper()

	dir := t.TempDir()
	for _, version := range versions {
		file, err := os.Create(filepath.Join(dir, "vmlinuz-"+version))
		if err != nil {
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}

	return dir
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
