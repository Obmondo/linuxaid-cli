package system

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/bitfield/script"
)

// UpdateSystem performs a system update based on the specified Linux distribution.
//
// This function accepts a `distribution` string representing the type of Linux distribution that needs
// to be updated. Depending on the distribution provided, it will invoke the appropriate update function.
func UpdateSystem(distribution string) error {
	switch distribution {
	case "ubuntu", "debian":
		return updateDebian()
	case "sles":
		return updateSUSE()
	case "centos", "rhel", "rocky":
		return updateRedHat()
	default:
		slog.Error("unknown distribution", slog.String("distribution", distribution))
		return nil
	}
}

func updateDebian() error {
	slog.Info("running apt update/upgrade/autoremove")
	enverr := os.Setenv("DEBIAN_FRONTEND", "noninteractive")
	if enverr != nil {
		slog.Error(enverr.Error())
		os.Exit(1)
	}

	if err := script.Exec("apt-get update").Wait(); err != nil {
		slog.Error("failed to update all repositories", slog.String("error", err.Error()))
	}
	pipe := script.Exec("apt-get --with-new-pkgs upgrade -y")
	_, err := pipe.Stdout()
	if err != nil {
		slog.Error("failed to upgrade all packages", slog.String("error", err.Error()))
		return err
	}

	exitStatus := pipe.ExitStatus()
	if exitStatus != 0 {
		slog.Error("exiting, apt update failed")
		return fmt.Errorf("apt-get update and upgrade failed: exit status %d", exitStatus)
	}

	pipe = script.Exec("apt-get autoremove -y")
	_, err = pipe.Stdout()
	if err != nil {
		slog.Error("failed to remove unused dependencies", slog.String("error", err.Error()))
		return err
	}

	return nil
}

func updateSUSE() error {
	slog.Info("running zypper refresh/update")
	if err := script.Exec("zypper refresh").Wait(); err != nil {
		slog.Error("failed to refresh all repositories", slog.String("error", err.Error()))
	}

	pipe := script.Exec("zypper update -y")
	_, err := pipe.Stdout()
	if err != nil {
		slog.Error("failed to update all repositories", slog.String("error", err.Error()))
		return err
	}

	exitStatus := pipe.ExitStatus()
	if exitStatus != 0 {
		slog.Error("exiting, suse update failed")
		return fmt.Errorf("suse update failed: exit status %d", exitStatus)
	}

	return nil
}

func updateRedHat() error {
	slog.Info("running yum repolist/update")
	if err := script.Exec("yum repolist").Wait(); err != nil {
		slog.Error("failed to fetch all repositories", slog.String("error", err.Error()))
	}

	pipe := script.Exec("yum update -y")
	_, err := pipe.Stdout()
	if err != nil {
		slog.Error("failed to update all packages", slog.String("error", err.Error()))
		return err
	}

	exitStatus := pipe.ExitStatus()
	if exitStatus != 0 {
		slog.Error("exiting, yum update failed")
		return fmt.Errorf("yum update failed: exit status %d", exitStatus)
	}

	return nil
}
