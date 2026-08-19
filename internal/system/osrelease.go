package system

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/shell"
)

const (
	ConstDistributionNameUbuntu      = "ubuntu"
	ConstDistributionNameDebian      = "debian"
	ConstDistributionNameSLES        = "sles"
	ConstDistributionNameCentOS      = "centos"
	ConstDistributionNameRHEL        = "rhel"
	ConstDistributionNameRocky       = "rocky"
	ConstDistributionNameOracleLinux = "ol"
	ConstDistributionNameTurrisOS    = "turrisos"

	constDistributionDebianUpdateRepoListCmd  = "apt update"
	constDistributionSLESUpdateRepoListCmd    = "zypper refresh"
	constDistributionRHELUpdateRepoListCmd    = "yum repolist"
	constDistributionOpenWRTUpdateRepoListCmd = "opkg update"

	constDistributionDebianCheckCACertificatesCmd  = "dpkg-query -W ca-certificates openssl"
	constDistributionSLESCheckCACertificatesCmd    = "rpm -q ca-certificates openssl ca-certificates-cacert ca-certificates-mozilla"
	constDistributionRHELCheckCACertificatesCmd    = "rpm -q ca-certificates openssl"
	constDistributionOpenWRTCheckCACertificatesCmd = "opkg list-installed ca-certificates openssl"

	constDistributionDebianInstallCACertificatesCmd  = "apt install -y ca-certificates"
	constDistributionSLESInstallCACertificatesCmd    = "zypper install -y ca-certificates openssl ca-certificates-cacert ca-certificates-mozilla"
	constDistributionRHELInstallCACertificatesCmd    = "yum install -y ca-certificates openssl"
	constDistributionOpenWRTInstallCACertificatesCmd = "opkg install libopenssl openssl-util libopenssl-conf"
)

type CertificateManagerCommands struct {
	runner                   shell.Runner
	updateRepoListCmd        string
	checkCACertificatesCmd   string
	installCACertificatesCmd string
}

func GetMajorRelease() string {
	osVersion, _, _ := strings.Cut(os.Getenv("VERSION_ID"), ".")
	return osVersion
}

// List of Supported OS
func IsSupportedOS(runner shell.Runner) (CertificateManagerCommands, error) {
	commands, err := getCommandsForInstallingCACertificates()
	if err != nil {
		return commands, fmt.Errorf("failed determining the os distribution: %w", err)
	}

	commands.runner = runner

	return commands, nil
}

// getCommandsForInstallingCACertificates returns the following for any distribution
// 1. command to update repository list
// 2. command to check if CA certificates are installed
// 3. command to install CA certificates
func getCommandsForInstallingCACertificates() (CertificateManagerCommands, error) {
	switch os.Getenv("ID") {
	case ConstDistributionNameUbuntu, ConstDistributionNameDebian:
		return CertificateManagerCommands{
			updateRepoListCmd:        constDistributionDebianUpdateRepoListCmd,
			checkCACertificatesCmd:   constDistributionDebianCheckCACertificatesCmd,
			installCACertificatesCmd: constDistributionDebianInstallCACertificatesCmd,
		}, nil
	case ConstDistributionNameSLES:
		return CertificateManagerCommands{
			updateRepoListCmd:        constDistributionSLESUpdateRepoListCmd,
			checkCACertificatesCmd:   constDistributionSLESCheckCACertificatesCmd,
			installCACertificatesCmd: constDistributionSLESInstallCACertificatesCmd,
		}, nil
	case ConstDistributionNameCentOS, ConstDistributionNameRHEL, ConstDistributionNameRocky, ConstDistributionNameOracleLinux:
		return CertificateManagerCommands{
			updateRepoListCmd:        constDistributionRHELUpdateRepoListCmd,
			checkCACertificatesCmd:   constDistributionRHELCheckCACertificatesCmd,
			installCACertificatesCmd: constDistributionRHELInstallCACertificatesCmd,
		}, nil
	case ConstDistributionNameTurrisOS:
		return CertificateManagerCommands{
			updateRepoListCmd:        constDistributionOpenWRTUpdateRepoListCmd,
			checkCACertificatesCmd:   constDistributionOpenWRTCheckCACertificatesCmd,
			installCACertificatesCmd: constDistributionOpenWRTInstallCACertificatesCmd,
		}, nil
	}
	return CertificateManagerCommands{}, errors.New("unknown distribution")
}

// UpdateRepositoryList updates repository list for any distribution
func (c *CertificateManagerCommands) UpdateRepositoryList() error {
	return c.runner.Quiet(c.updateRepoListCmd).Err
}

// CheckAndInstallCaCertificates handles the installation of CA certificates for any distribution
func (c *CertificateManagerCommands) CheckAndInstallCaCertificates() error {
	if c.isCaCertificateInstalled() {
		return nil
	}

	return c.runner.Quiet(c.installCACertificatesCmd).Err
}

// isCaCertificateInstalled reports whether the distribution's CA certificate packages are already
// installed, by running the distribution's own query command.
func (c *CertificateManagerCommands) isCaCertificateInstalled() bool {
	result := c.runner.Quiet(c.checkCACertificatesCmd)
	if result.Err != nil {
		slog.Error("failed to determine if ca-certificates is installed", slog.Any("error", result.Err))
	}

	return result.ExitCode == 0
}
