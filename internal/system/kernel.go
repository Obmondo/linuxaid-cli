package system

import (
	"fmt"
	"log/slog"
	"strings"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/disk"
	"github.com/bitfield/script"
)

const bootDirectory = "/boot"

// CheckKernelAndRebootIfNeeded checks if a new kernel is installed and reboots if necessary.
func CheckKernelAndRebootIfNeeded(noReboot bool) error {
	// Get installed kernel of the system
	// If kernel is installed, then only we will try to reboot.
	// In lxc kernel wont be present
	installedKernel, err := getInstalledKernel(bootDirectory)
	if err != nil {
		slog.Error("error occurred while trying to find kernel", slog.String("error", err.Error()))
		return err
	}
	if installedKernel == "" {
		slog.Warn("looks like no kernel is installed on the node")
		return nil
	}

	// Get running kernel of the system
	runningKernel, err := script.Exec("uname -r").String()
	if err != nil {
		slog.Error("Failed to fetch Running Kernel", slog.String("error", err.Error()))
		return err
	}
	runningKernel = strings.TrimSpace(runningKernel)

	// Check the disk size
	if err := disk.CheckDiskSize(); err != nil {
		slog.Error("unable to check disk size", slog.String("error", err.Error()))
		return err
	}

	// Reboot the node, if we have installed a new kernel
	if installedKernel != runningKernel && !noReboot {
		slog.Info("looks like newer kernel is installed, so going ahead with reboot now")
		script.Exec("reboot --force")
	}

	return nil
}

// getInstalledKernel returns the installed Kernel
func getInstalledKernel(bootDirectory string) (string, error) {
	formatedBashCommand := fmt.Sprintf("find %s/vmlinuz-* | sort -V | tail -n 1 | sed 's|.*vmlinuz-||'", bootDirectory)
	installedKernel, err := script.Exec(fmt.Sprintf("/bin/bash -c \"%s\"", formatedBashCommand)).String()
	installedKernel = strings.TrimSpace(installedKernel)
	return installedKernel, err
}
