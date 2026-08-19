package system

import (
	"fmt"
	"log/slog"
	"strings"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/disk"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/shell"
)

const bootDirectory = "/boot"

// CheckKernelAndRebootIfNeeded checks if a new kernel is installed and reboots if necessary.
func CheckKernelAndRebootIfNeeded(runner shell.Runner, noReboot bool) error {
	// Get installed kernel of the system
	// If kernel is installed, then only we will try to reboot.
	// In lxc kernel wont be present
	installedKernel, err := getInstalledKernel(runner, bootDirectory)
	if err != nil {
		slog.Error("error occurred while trying to find kernel", slog.String("error", err.Error()))
		return err
	}
	if installedKernel == "" {
		slog.Warn("looks like no kernel is installed on the node")
		return nil
	}

	// Get running kernel of the system
	running := runner.Capture("uname -r")
	if running.Err != nil {
		slog.Error("Failed to fetch Running Kernel", slog.Any("error", running.Err))
		return running.Err
	}
	runningKernel := strings.TrimSpace(running.Output)

	// Check the disk size
	if err := disk.CheckDiskSize(); err != nil {
		slog.Error("unable to check disk size", slog.String("error", err.Error()))
		return err
	}

	// Reboot the node, if we have installed a new kernel
	if installedKernel != runningKernel && !noReboot {
		slog.Info("looks like newer kernel is installed, so going ahead with reboot now")
		runner.Run("reboot --force")
	}

	return nil
}

// getInstalledKernel returns the installed Kernel
func getInstalledKernel(runner shell.Runner, bootDirectory string) (string, error) {
	formatedBashCommand := fmt.Sprintf("find %s/vmlinuz-* | sort -V | tail -n 1 | sed 's|.*vmlinuz-||'", bootDirectory)
	result := runner.Capture(fmt.Sprintf("/bin/bash -c \"%s\"", formatedBashCommand))

	return strings.TrimSpace(result.Output), result.Err
}
