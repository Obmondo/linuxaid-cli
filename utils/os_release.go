package utils

import (
	"errors"
	"log/slog"
	"os"
	"strings"

	"github.com/bitfield/script"
)

const (
	constDistributionNameUbuntu = "ubuntu"
	constDistributionNameDebian = "debian"
	constDistributionNameSLES   = "sles"
	constDistributionNameCentOS = "centos"
	constDistributionNameRHEL   = "rhel"

	constDistributionDebianUpdateRepoListCmd = "apt update"
	constDistributionSLESUpdateRepoListCmd   = "zypper refresh"
	constDistributionRHELUpdateRepoListCmd   = "yum repolist"

	constDistributionDebianCheckCACertificatesCmd = "dpkg-query -W ca-certificates openssl"
	constDistributionSLESCheckCACertificatesCmd   = "rpm -q ca-certificates openssl ca-certificates-cacert ca-certificates-mozilla"
	constDistributionRHELCheckCACertificatesCmd   = "rpm -q ca-certificates openssl"

	constDistributionDebianInstallCACertificatesCmd = "apt install -y ca-certificates"
	constDistributionSLESInstallCACertificatesCmd   = "zypper install -y ca-certificates openssl ca-certificates-cacert ca-certificates-mozilla"
	constDistributionRHELInstallCACertificatesCmd   = "yum install -y ca-certificates openssl"
)

type certificateManagerCommands struct {
	updateRepoListCmd        string
	checkCACertificatesCmd   string
	installCACertificatesCmd string
}

func GetMajorRelease() string {
	osVersion, _, _ := strings.Cut(os.Getenv("VERSION_ID"), ".")
	return osVersion
}

// List of Supported OS
func SupportedOS() {
	commands, err := getCommandsForInstallingCACertificates()
	if err != nil {
		slog.Error("can't get commands for installing ca certificates", slog.String("error", err.Error()))
		os.Exit(1)
	}
	updateRepositoryList(commands.updateRepoListCmd)
	checkAndInstallCaCertificates(commands.checkCACertificatesCmd, commands.installCACertificatesCmd)
}

// getCommandsForInstallingCACertificates returns the following for any distribution
// 1. command to update repository list
// 2. command to check if CA certificates are installed
// 3. command to install CA certificates
func getCommandsForInstallingCACertificates() (certificateManagerCommands, error) {
	switch os.Getenv("ID") {
	case constDistributionNameUbuntu, constDistributionNameDebian:
		return certificateManagerCommands{
			updateRepoListCmd:        constDistributionDebianUpdateRepoListCmd,
			checkCACertificatesCmd:   constDistributionDebianCheckCACertificatesCmd,
			installCACertificatesCmd: constDistributionDebianInstallCACertificatesCmd,
		}, nil
	case constDistributionNameSLES:
		return certificateManagerCommands{
			updateRepoListCmd:        constDistributionSLESUpdateRepoListCmd,
			checkCACertificatesCmd:   constDistributionSLESCheckCACertificatesCmd,
			installCACertificatesCmd: constDistributionSLESInstallCACertificatesCmd,
		}, nil
	case constDistributionNameCentOS, constDistributionNameRHEL:
		return certificateManagerCommands{
			updateRepoListCmd:        constDistributionRHELUpdateRepoListCmd,
			checkCACertificatesCmd:   constDistributionRHELCheckCACertificatesCmd,
			installCACertificatesCmd: constDistributionRHELInstallCACertificatesCmd,
		}, nil
	default:
		err := errors.New("unknown distribution")
		return certificateManagerCommands{}, err
	}
}

// updateRepositoryList updates repository list for any distribution
func updateRepositoryList(updateCommand string) {
	pipe := script.Exec(updateCommand)
	if err := pipe.Wait(); err != nil {
		slog.Error("failed to update all repositories", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

// checkAndInstallCaCertificates handles the installation of CA certificates for any distribution
func checkAndInstallCaCertificates(checkCommand, installCommand string) {
	isInstalled := IsCaCertificateInstalled(checkCommand)
	if !isInstalled {
		pipe := script.Exec(installCommand)
		if err := pipe.Wait(); err != nil {
			slog.Error("failed to install ca-certificates", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}
}
