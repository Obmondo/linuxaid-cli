package system

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/disk"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/shell"
)

const (
	rebootCommand       = "reboot --force"
	turrisRebootCommand = "reboot"

	// rebootGraceMinutes is how many minutes reboot waits for the kernel to bring the
	// machine down before it concludes the reboot request was lost.
	rebootGraceMinutes = 5
)

// bootDirectory and rebootGracePeriod are vars, not consts, so tests can point them at a
// temporary directory and shorten the wait.
var (
	bootDirectory     = "/boot"
	rebootGracePeriod = rebootGraceMinutes * time.Minute
)

func getRebootCommand(distribution string) string {
	switch distribution {
	case ConstDistributionNameTurrisOS:
		return turrisRebootCommand
	default:
		return rebootCommand
	}
}

// CheckKernelAndRebootIfNeeded reboots the node when a newer kernel than the running one is
// installed, unless noReboot is set. It always logs which branch it took: a silent no-op here
// is what previously hid nodes that never rebooted after an update.
func CheckKernelAndRebootIfNeeded(runner shell.Runner, distribution string, noReboot bool) error {
	var installedKernel string
	var err error

	switch distribution {
	case ConstDistributionNameTurrisOS:
		installedKernel, err = getTurrisInstalledKernel(runner)
		if err == nil {
			if idx := strings.IndexAny(installedKernel, "~-+"); idx != -1 {
				installedKernel = installedKernel[:idx]
			}
		}
	default:
		// In an LXC container there is no kernel in /boot and nothing to reboot into.
		installedKernel, err = getInstalledKernel(bootDirectory)
	}

	if err != nil {
		slog.Error("error occurred while trying to find kernel", slog.String("error", err.Error()))
		return err
	}
	if installedKernel == "" {
		slog.Warn("looks like no kernel is installed on the node")
		return nil
	}

	running := runner.Capture("uname -r")
	if running.Err != nil {
		slog.Error("Failed to fetch Running Kernel", slog.Any("error", running.Err))
		return running.Err
	}
	runningKernel := strings.TrimSpace(running.Output)
	if runningKernel == "" {
		slog.Error("empty output from uname -r")
		return errors.New("empty output from uname -r")
	}

	compareInstalled := installedKernel

	if compareInstalled == runningKernel {
		slog.Info("running kernel is already the newest installed kernel, no reboot needed",
			slog.String("kernel", runningKernel))
		return nil
	}

	if noReboot {
		slog.Info("a newer kernel is installed but reboot is disabled, not rebooting",
			slog.String("installed_kernel", installedKernel),
			slog.String("running_kernel", runningKernel))
		return nil
	}

	if err := disk.CheckDiskSize(); err != nil {
		slog.Error("unable to check disk size", slog.String("error", err.Error()))
		return err
	}

	return reboot(runner, distribution, installedKernel, runningKernel)
}

// reboot asks PID 1 to reboot and then blocks until the machine goes down. The request is
// asynchronous, so this process must not return: obmondo-system-update.service is Type=oneshot,
// and once its main process exits systemd tears down the unit's cgroup, which can kill the
// pending reboot before the kernel goes down. If the machine is still up after the grace
// period the reboot did not take effect, so report it rather than hang the service forever.
func reboot(runner shell.Runner, distribution, installedKernel, runningKernel string) error {
	slog.Info("a newer kernel is installed, rebooting now",
		slog.String("installed_kernel", installedKernel),
		slog.String("running_kernel", runningKernel))

	rebootCmd := getRebootCommand(distribution)
	result := runner.Run(rebootCmd)
	if result.Err != nil {
		slog.Error("failed to trigger reboot", slog.Any("error", result.Err))
		return fmt.Errorf("running %q: %w", rebootCmd, result.Err)
	}
	if result.ExitCode != 0 {
		slog.Error("reboot command exited non-zero", slog.Int("exit_code", result.ExitCode))
		return fmt.Errorf("%q exited with status %d", rebootCmd, result.ExitCode)
	}

	slog.Info("reboot requested, waiting for the system to shut down")
	time.Sleep(rebootGracePeriod)

	slog.Error("still running after the reboot request, reboot did not take effect")
	return errors.New("system did not reboot within the grace period")
}

// getInstalledKernel returns the newest kernel version found in bootDirectory (the part after
// "vmlinuz-"), or "" when the directory holds no kernel image. The GRUB rescue image is not a
// real kernel candidate and is skipped. Ordering uses compareKernelVersions rather than
// "sort -V"; see that function for why.
func getInstalledKernel(bootDirectory string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(bootDirectory, "vmlinuz-*"))
	if err != nil {
		return "", err
	}

	var newest string
	for _, match := range matches {
		version := strings.TrimPrefix(filepath.Base(match), "vmlinuz-")
		if strings.Contains(version, "rescue") {
			continue
		}
		if newest == "" || compareKernelVersions(version, newest) > 0 {
			newest = version
		}
	}

	return newest, nil
}

// getTurrisInstalledKernel returns the installed kernel package version parsed from opkg,
// or an error if the query fails or output is empty.
func getTurrisInstalledKernel(runner shell.Runner) (string, error) {
	opkgResult := runner.Capture("opkg list-installed kernel")
	if opkgResult.Err != nil {
		return "", fmt.Errorf("failed to list installed kernel package: %w", opkgResult.Err)
	}
	output := strings.TrimSpace(opkgResult.Output)
	if output == "" {
		return "", errors.New("empty output from opkg list-installed kernel")
	}

	// Example output: kernel - 6.6.88~030160133b7136452d44a9fedbe72889-r3
	name, pkgVersion, found := strings.Cut(output, " - ")
	if !found || strings.TrimSpace(name) != "kernel" {
		return "", fmt.Errorf("unexpected opkg output format: %q", output)
	}

	pkgVersion = strings.TrimSpace(pkgVersion)
	if pkgVersion == "" {
		return "", errors.New("empty kernel package version from opkg")
	}

	return pkgVersion, nil
}
