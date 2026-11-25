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
	case ConstDistributionNameTurrisOS:
		return updateTurris(runner)
	default:
		slog.Error("unknown distribution", slog.String("distribution", distribution))
		return nil
	}
}

// DEBIAN_FRONTEND=noninteractive only covers debconf questions. dpkg asks its own
// question when a package ships a new version of a config file that was changed
// locally (e.g. /etc/systemd/resolved.conf managed by LinuxAid), and with no stdin
// that question aborts the whole upgrade. confdef/confold keep the local file.
const (
	dpkgConfigureCmd = "dpkg --force-confdef --force-confold --configure -a"
	aptUpgradeCmd    = "apt-get " + aptLockTimeout + " -o Dpkg::Options::=--force-confdef -o Dpkg::Options::=--force-confold --with-new-pkgs upgrade -y"
	aptUpdateCmd     = "apt-get " + aptLockTimeout + " update"
	aptAutoremoveCmd = "apt-get " + aptLockTimeout + " autoremove -y"
	aptCleanCmd      = "apt-get " + aptLockTimeout + " clean"

	// aptLockTimeout makes apt wait up to 10 minutes for the dpkg lock instead of failing at
	// once when another apt run (apt-daily, an operator, an older update job) holds it.
	aptLockTimeout = "-o DPkg::Lock::Timeout=600"
)

func updateDebian(runner shell.Runner) error {
	slog.Info("running apt update/upgrade/autoremove")

	if err := os.Setenv("DEBIAN_FRONTEND", "noninteractive"); err != nil {
		return fmt.Errorf("could not set DEBIAN_FRONTEND: %w", err)
	}

	// Finish any package left half-configured by an earlier interrupted run,
	// otherwise apt refuses to upgrade.
	if result := runner.Run(dpkgConfigureCmd); result.Err != nil || result.ExitCode != 0 {
		slog.Error("failed to configure pending packages", slog.Any("error", result.Err), slog.Int("exit_code", result.ExitCode))
		return fmt.Errorf("dpkg --configure -a failed: exit status %d", result.ExitCode)
	}

	if result := runner.Quiet(aptUpdateCmd); result.Err != nil {
		slog.Error("failed to update all repositories", slog.Any("error", result.Err))
	}

	upgrade := runner.Run(aptUpgradeCmd)
	if upgrade.Err != nil {
		slog.Error("failed to upgrade all packages", slog.Any("error", upgrade.Err))
		return upgrade.Err
	}

	if upgrade.ExitCode != 0 {
		slog.Error("exiting, apt update failed")
		return fmt.Errorf("apt-get update and upgrade failed: exit status %d", upgrade.ExitCode)
	}

	if autoremove := runner.Run(aptAutoremoveCmd); autoremove.Err != nil {
		slog.Error("failed to remove unused dependencies", slog.Any("error", autoremove.Err))
		return autoremove.Err
	}

	if clean := runner.Run(aptCleanCmd); clean.Err != nil {
		slog.Error("failed to clean apt cache", slog.Any("error", clean.Err))
		return clean.Err
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

	if clean := runner.Run("zypper clean --all"); clean.Err != nil {
		slog.Error("failed to clean zypper cache", slog.Any("error", clean.Err))
		return clean.Err
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

	if clean := runner.Run("yum clean all"); clean.Err != nil {
		slog.Error("failed to clean yum cache", slog.Any("error", clean.Err))
		return clean.Err
	}

	return nil
}

func updateTurris(runner shell.Runner) error {
	slog.Info("running pkgupdate --batch --no-immediate-reboot")

	update := runner.Run("pkgupdate --batch --no-immediate-reboot")
	if update.Err != nil {
		slog.Error("failed to update turris os packages", slog.Any("error", update.Err))
		return update.Err
	}

	if update.ExitCode != 0 {
		slog.Error("exiting, pkgupdate failed")
		return fmt.Errorf("pkgupdate failed: exit status %d", update.ExitCode)
	}

	return nil
}
