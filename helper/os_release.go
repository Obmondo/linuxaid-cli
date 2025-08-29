package helper

import (
	"errors"
	"fmt"
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
func IsSupportedOS() (certificateManagerCommands, error) {
	commands, err := getCommandsForInstallingCACertificates()
	if err != nil {
		return commands, fmt.Errorf("failed determining the os distribution: %w", err)
	}

	return commands, nil
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
	}
	return certificateManagerCommands{}, errors.New("unknown distribution")
}

// UpdateRepositoryList updates repository list for any distribution
func (c *certificateManagerCommands) UpdateRepositoryList() error {
	pipe := script.Exec(c.updateRepoListCmd)
	if err := pipe.Wait(); err != nil {
		return err
	}

	return nil
}

// CheckAndInstallCaCertificates handles the installation of CA certificates for any distribution
func (c *certificateManagerCommands) CheckAndInstallCaCertificates() error {
	isInstalled := IsCaCertificateInstalled(c.checkCACertificatesCmd)
	if !isInstalled {
		pipe := script.Exec(c.installCACertificatesCmd)
		if err := pipe.Wait(); err != nil {
			return err
		}
	}

	return nil
}
