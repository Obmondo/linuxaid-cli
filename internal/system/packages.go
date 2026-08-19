package system

import (
	"fmt"
	"log/slog"
	"os"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/shell"
)

// UpdateSystem performs a system update based on the specified Linux distribution.
//
// This function accepts a `distribution` string representing the type of Linux distribution that needs
// to be updated. Depending on the distribution provided, it will invoke the appropriate update function.
func UpdateSystem(runner shell.Runner, distribution string) error {
	switch distribution {
	case ConstDistributionNameUbuntu, ConstDistributionNameDebian:
		return updateDebian(runner)
	case ConstDistributionNameSLES:
		return updateSUSE(runner)
	case ConstDistributionNameCentOS, ConstDistributionNameRHEL, ConstDistributionNameRocky:
		return updateRedHat(runner)
	default:
		slog.Error("unknown distribution", slog.String("distribution", distribution))
		return nil
	}
}

func updateDebian(runner shell.Runner) error {
	slog.Info("running apt update/upgrade/autoremove")

	if err := os.Setenv("DEBIAN_FRONTEND", "noninteractive"); err != nil {
		return fmt.Errorf("could not set DEBIAN_FRONTEND: %w", err)
	}

	if result := runner.Quiet("apt-get update"); result.Err != nil {
		slog.Error("failed to update all repositories", slog.Any("error", result.Err))
	}

	upgrade := runner.Run("apt-get --with-new-pkgs upgrade -y")
	if upgrade.Err != nil {
		slog.Error("failed to upgrade all packages", slog.Any("error", upgrade.Err))
		return upgrade.Err
	}

	if upgrade.ExitCode != 0 {
		slog.Error("exiting, apt update failed")
		return fmt.Errorf("apt-get update and upgrade failed: exit status %d", upgrade.ExitCode)
	}

	if autoremove := runner.Run("apt-get autoremove -y"); autoremove.Err != nil {
		slog.Error("failed to remove unused dependencies", slog.Any("error", autoremove.Err))
		return autoremove.Err
	}

	return nil
}

func updateSUSE(runner shell.Runner) error {
	slog.Info("running zypper refresh/update")

	if result := runner.Quiet("zypper refresh"); result.Err != nil {
		slog.Error("failed to refresh all repositories", slog.Any("error", result.Err))
	}

	update := runner.Run("zypper update -y")
	if update.Err != nil {
		slog.Error("failed to update all repositories", slog.Any("error", update.Err))
		return update.Err
	}

	if update.ExitCode != 0 {
		slog.Error("exiting, suse update failed")
		return fmt.Errorf("suse update failed: exit status %d", update.ExitCode)
	}

	return nil
}

func updateRedHat(runner shell.Runner) error {
	slog.Info("running yum repolist/update")

	if result := runner.Quiet("yum repolist"); result.Err != nil {
		slog.Error("failed to fetch all repositories", slog.Any("error", result.Err))
	}

	update := runner.Run("yum update -y")
	if update.Err != nil {
		slog.Error("failed to update all packages", slog.Any("error", update.Err))
		return update.Err
	}

	if update.ExitCode != 0 {
		slog.Error("exiting, yum update failed")
		return fmt.Errorf("yum update failed: exit status %d", update.ExitCode)
	}

	return nil
}
