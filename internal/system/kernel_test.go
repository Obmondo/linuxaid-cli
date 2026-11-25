package system

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/shell"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/shell/shelltest"
)

const (
	testRhelKernelPoint = "4.18.0-553.el8_10.x86_64"
	testRhelKernelZ     = "4.18.0-553.157.1.el8_10.x86_64"
	testUnameCmd        = "uname -r"
	testOpkgCmd         = "opkg list-installed kernel"
	testUbuntuUpgrade   = aptUpgradeCmd
	testTurrisUnameOut  = "6.6.88\n"
	testUbuntuKernel    = "6.8.0-134-generic"
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
		testRhelKernelPoint,
		"4.18.0-553.137.1.el8_10.x86_64",
		testRhelKernelZ,
		"0-rescue-0123456789abcdef0123456789abcdef",
	)

	got, err := getInstalledKernel(bootDir)
	if err != nil {
		t.Fatal(err)
	}
	if want := testRhelKernelZ; got != want {
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

	rhelBoot := []string{testRhelKernelPoint, testRhelKernelZ}

	t.Run("reboots when a newer kernel is installed", func(t *testing.T) {
		bootDirectory = writeBootDir(t, rhelBoot...)
		runner := &shelltest.Recorder{Results: map[string]shell.Result{
			testUnameCmd: {Output: testRhelKernelPoint + "\n"},
		}}

		// The fake runner never brings the machine down, so reboot reports that the grace
		// period elapsed. What matters is that the reboot command was issued.
		if err := CheckKernelAndRebootIfNeeded(runner, ConstDistributionNameDebian, false); err == nil {
			t.Error("expected an error because the fake runner does not reboot the machine")
		}
		if !runner.Ran(rebootCommand) {
			t.Errorf("expected %q to run, got %v", rebootCommand, runner.Commands())
		}
	})

	t.Run("does not reboot when the running kernel is already newest", func(t *testing.T) {
		bootDirectory = writeBootDir(t, rhelBoot...)
		runner := &shelltest.Recorder{Results: map[string]shell.Result{
			testUnameCmd: {Output: testRhelKernelZ + "\n"},
		}}

		if err := CheckKernelAndRebootIfNeeded(runner, ConstDistributionNameDebian, false); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if runner.Ran("reboot") {
			t.Errorf("expected no reboot, got %v", runner.Commands())
		}
	})

	t.Run("does not reboot when reboot is disabled", func(t *testing.T) {
		bootDirectory = writeBootDir(t, rhelBoot...)
		runner := &shelltest.Recorder{Results: map[string]shell.Result{
			testUnameCmd: {Output: testRhelKernelPoint + "\n"},
		}}

		if err := CheckKernelAndRebootIfNeeded(runner, ConstDistributionNameDebian, true); err != nil {
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
		{distribution: ConstDistributionNameUbuntu, want: testUbuntuUpgrade},
		{distribution: ConstDistributionNameDebian, want: testUbuntuUpgrade},
		{distribution: ConstDistributionNameSLES, want: "zypper update -y"},
		{distribution: ConstDistributionNameRHEL, want: "yum update -y"},
		{distribution: ConstDistributionNameRocky, want: "yum update -y"},
		{distribution: ConstDistributionNameTurrisOS, want: "pkgupdate --batch --no-immediate-reboot"},
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
			testUbuntuUpgrade: {ExitCode: 100},
		},
	}

	if err := UpdateSystem(runner, ConstDistributionNameDebian); err == nil {
		t.Error("expected a failing apt-get upgrade to be reported, got nil")
	}
}

// TestUpdateDebianKeepsLocalConffiles is the regression test for hosts where LinuxAid edits
// /etc/systemd/resolved.conf: dpkg's conffile question aborted the upgrade and left
// systemd-resolved half-configured. Pending packages must be configured first, and both
// dpkg and apt must keep the local config file without asking.
func TestUpdateDebianKeepsLocalConffiles(t *testing.T) {
	runner := &shelltest.Recorder{}

	if err := UpdateSystem(runner, ConstDistributionNameUbuntu); err != nil {
		t.Fatalf("UpdateSystem returned %v", err)
	}

	commands := runner.Commands()
	configure, upgrade := slices.Index(commands, dpkgConfigureCmd), slices.Index(commands, aptUpgradeCmd)
	if configure == -1 || upgrade == -1 || configure > upgrade {
		t.Fatalf("expected %q before %q, got %v", dpkgConfigureCmd, aptUpgradeCmd, commands)
	}

	for _, command := range []string{dpkgConfigureCmd, aptUpgradeCmd} {
		if !strings.Contains(command, "--force-confdef") || !strings.Contains(command, "--force-confold") {
			t.Errorf("expected %q to keep local conffiles", command)
		}
	}
}

// TestUpdateDebianStopsWhenDpkgConfigureFails makes sure a broken dpkg state is reported
// instead of running an upgrade that cannot succeed.
func TestUpdateDebianStopsWhenDpkgConfigureFails(t *testing.T) {
	runner := &shelltest.Recorder{
		Results: map[string]shell.Result{
			dpkgConfigureCmd: {ExitCode: 1},
		},
	}

	if err := UpdateSystem(runner, ConstDistributionNameDebian); err == nil {
		t.Error("expected a failing dpkg --configure -a to be reported, got nil")
	}

	if runner.Ran(aptUpgradeCmd) {
		t.Errorf("expected no upgrade after dpkg failed, got %v", runner.Commands())
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

func TestCheckTurrisKernelAndRebootIfNeeded(t *testing.T) {
	origGrace := rebootGracePeriod
	t.Cleanup(func() {
		rebootGracePeriod = origGrace
	})
	rebootGracePeriod = time.Millisecond

	t.Run("reboots when newer kernel package installed on Turris with real opkg format", func(t *testing.T) {
		runner := &shelltest.Recorder{Results: map[string]shell.Result{
			testUnameCmd: {Output: testTurrisUnameOut},
			testOpkgCmd:  {Output: "kernel - 6.6.89-r1\n"},
		}}

		if err := CheckKernelAndRebootIfNeeded(runner, ConstDistributionNameTurrisOS, false); err == nil {
			t.Error("expected an error because fake runner does not reboot")
		}
		cmds := runner.Commands()
		if !slices.Contains(cmds, turrisRebootCommand) {
			t.Errorf("expected 'reboot' to run, got %v", cmds)
		}
		if runner.Ran(rebootCommand) {
			t.Errorf("expected plain 'reboot' for Turris, got 'reboot --force'")
		}
	})

	t.Run("does not reboot when Turris kernel is up to date with hash suffix", func(t *testing.T) {
		runner := &shelltest.Recorder{Results: map[string]shell.Result{
			testUnameCmd: {Output: testTurrisUnameOut},
			testOpkgCmd:  {Output: "kernel - 6.6.88~030160133b7136452d44a9fedbe72889-r3\n"},
		}}

		if err := CheckKernelAndRebootIfNeeded(runner, ConstDistributionNameTurrisOS, false); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if runner.Ran("reboot") {
			t.Errorf("expected no reboot, got %v", runner.Commands())
		}
	})

	t.Run("reboots when prefix collision occurs (running 6.6.8 vs installed 6.6.88)", func(t *testing.T) {
		runner := &shelltest.Recorder{Results: map[string]shell.Result{
			testUnameCmd: {Output: "6.6.8\n"},
			testOpkgCmd:  {Output: "kernel - 6.6.88-r1\n"},
		}}

		if err := CheckKernelAndRebootIfNeeded(runner, ConstDistributionNameTurrisOS, false); err == nil {
			t.Error("expected an error because fake runner does not reboot")
		}
		if !runner.Ran("reboot") {
			t.Errorf("expected 'reboot' to run, got %v", runner.Commands())
		}
	})

	t.Run("reports error when opkg fails or returns empty", func(t *testing.T) {
		runner := &shelltest.Recorder{Results: map[string]shell.Result{
			testUnameCmd: {Output: testTurrisUnameOut},
			testOpkgCmd:  {Output: "\n"},
		}}

		if err := CheckKernelAndRebootIfNeeded(runner, ConstDistributionNameTurrisOS, false); err == nil {
			t.Error("expected error when opkg output is empty, got nil")
		}
	})
}

func TestCheckReboot(t *testing.T) {
	origBoot, origFile := bootDirectory, rebootRequiredFile
	t.Cleanup(func() {
		bootDirectory = origBoot
		rebootRequiredFile = origFile
	})

	const installed, running = "7.0.0-43-generic", "7.0.0-22-generic"

	t.Run("kernel newer", func(t *testing.T) { // TestCheckRebootKernelNewer
		bootDirectory = writeBootDir(t, running, installed)
		rebootRequiredFile = filepath.Join(t.TempDir(), "reboot-required")
		runner := &shelltest.Recorder{Results: map[string]shell.Result{testUnameCmd: {Output: running + "\n"}}}

		got, err := CheckReboot(runner, ConstDistributionNameUbuntu)
		if err != nil {
			t.Fatalf("CheckReboot: %v", err)
		}
		want := RebootCheck{Required: true, Reason: RebootReasonKernel, RunningKernel: running, InstalledKernel: installed}
		if got != want {
			t.Errorf("CheckReboot() = %+v, want %+v", got, want)
		}
		if runner.Ran("reboot") {
			t.Errorf("CheckReboot must never reboot, ran %v", runner.Commands())
		}
	})

	t.Run("reboot-required file", func(t *testing.T) { // TestCheckRebootRebootRequiredFile
		bootDirectory = writeBootDir(t, installed)
		rebootRequiredFile = filepath.Join(t.TempDir(), "reboot-required")
		if err := os.WriteFile(rebootRequiredFile, []byte("*** System restart required ***\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		runner := &shelltest.Recorder{Results: map[string]shell.Result{testUnameCmd: {Output: installed + "\n"}}}

		got, err := CheckReboot(runner, ConstDistributionNameUbuntu)
		if err != nil {
			t.Fatalf("CheckReboot: %v", err)
		}
		if !got.Required || got.Reason != RebootReasonFile {
			t.Errorf("CheckReboot() = %+v, want Required with reason %q", got, RebootReasonFile)
		}
	})

	t.Run("nothing to do", func(t *testing.T) { // TestCheckRebootNone
		bootDirectory = writeBootDir(t, installed)
		rebootRequiredFile = filepath.Join(t.TempDir(), "reboot-required")
		runner := &shelltest.Recorder{Results: map[string]shell.Result{testUnameCmd: {Output: installed + "\n"}}}

		got, err := CheckReboot(runner, ConstDistributionNameUbuntu)
		if err != nil {
			t.Fatalf("CheckReboot: %v", err)
		}
		if got.Required || got.Reason != "" {
			t.Errorf("CheckReboot() = %+v, want no reboot", got)
		}
	})
}
