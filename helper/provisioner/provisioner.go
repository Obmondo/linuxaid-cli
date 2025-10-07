package provisioner

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"gitea.obmondo.com/EnableIT/go-scripts/config"
	"gitea.obmondo.com/EnableIT/go-scripts/constant"
	"gitea.obmondo.com/EnableIT/go-scripts/helper"
	api "gitea.obmondo.com/EnableIT/go-scripts/pkg/obmondo"
	"gitea.obmondo.com/EnableIT/go-scripts/pkg/puppet"
	"gitea.obmondo.com/EnableIT/go-scripts/pkg/webtee"
)

const tmpDir = "/tmp"

type Provisioner struct {
	webtee    *webtee.Webtee
	apiClient api.ObmondoClient
	puppet    *puppet.Service
	certName  string
}

// NewService creates a new Puppet installer service.
func NewService(apiClient api.ObmondoClient, puppet *puppet.Service) *Provisioner {
	return &Provisioner{
		apiClient: apiClient,
		puppet:    puppet,
		certName:  config.GetCertName(),
	}
}

func (s *Provisioner) ProvisionPuppet() {
	switch os.Getenv("ID") {
	case "ubuntu", "debian":
		if err := s.provisionForDebian(); err != nil {
			slog.Error("failed to install puppet", slog.Any("error", err))
			os.Exit(1)
		}
	case "sles":
		if err := s.provisionForSuse(); err != nil {
			slog.Error("failed to install puppet", slog.Any("error", err))
			os.Exit(1)
		}
	case "centos", "rhel":
		if err := s.provisionForRedHat(); err != nil {
			slog.Error("failed to install puppet", slog.Any("error", err))
			os.Exit(1)
		}
	default:
		slog.Error("unknown distribution, exiting")
		os.Exit(1)
	}
}

// provisionForDebian installs puppet-agent on Ubuntu/Debian systems
func (s *Provisioner) provisionForDebian() error {
	helper.RequireUbuntuCodeNameEnv()

	codeName := os.Getenv("UBUNTU_CODENAME")
	s.webtee.RemoteLogObmondo([]string{"apt update"}, s.certName)
	s.webtee.RemoteLogObmondo([]string{"apt install -y iptables"}, s.certName)
	var ubuntuVersion string
	switch codeName {
	case "jammy":
		ubuntuVersion = "ubuntu22.04"
	case "noble":
		ubuntuVersion = "ubuntu24.04"
	}

	fullPuppetVersion := fmt.Sprintf("%s-1+%s", constant.PuppetVersion, ubuntuVersion)
	packageName := fmt.Sprintf("openvox-agent_%s_amd64.deb", fullPuppetVersion)
	downloadPath := filepath.Join(tmpDir, packageName)
	url := fmt.Sprintf("https://repos.obmondo.com/openvox/apt/pool/%s/o/openvox-agent/%s",
		constant.PuppetMajorVersion, packageName)
	if err := s.puppet.DownloadAgent(downloadPath, url); err != nil {
		return err
	}

	installCmd := []string{fmt.Sprintf("apt install -y %s", downloadPath)}
	s.webtee.RemoteLogObmondo(installCmd, s.certName)

	return nil
}

// provisionForRedHat installs puppet-agent on RHEL/CentOS systems
func (s *Provisioner) provisionForRedHat() error {
	s.webtee.RemoteLogObmondo([]string{"yum install -y iptables"}, s.certName)

	majRelease := helper.GetMajorRelease()

	fullPuppetVersion := fmt.Sprintf("%s-1.el%s", constant.PuppetVersion, majRelease)
	packageName := fmt.Sprintf("openvox-agent-%s.x86_64", fullPuppetVersion)
	downloadPath := filepath.Join(tmpDir, packageName+".rpm")
	url := fmt.Sprintf("https://repos.obmondo.com/openvox/yum/%s/el/%s/x86_64/%s.rpm",
		constant.PuppetMajorVersion, majRelease, packageName)

	if err := s.puppet.DownloadAgent(downloadPath, url); err != nil {
		return err
	}

	installCmd := []string{fmt.Sprintf("yum install %s -y", downloadPath)}
	s.webtee.RemoteLogObmondo(installCmd, s.certName)

	return nil
}

// provisionForSuse installs puppet-agent on SUSE systems
func (s *Provisioner) provisionForSuse() error {
	s.webtee.RemoteLogObmondo([]string{"zypper install -y iptables"}, s.certName)

	majRelease := helper.GetMajorRelease()

	fullPuppetVersion := fmt.Sprintf("%s-1.sles%s", constant.PuppetVersion, majRelease)
	packageName := fmt.Sprintf("openvox-agent-%s.x86_64", fullPuppetVersion)
	downloadPath := filepath.Join(tmpDir, packageName+".rpm")
	url := fmt.Sprintf("https://repos.obmondo.com/openvox/sles/%s/%s/x86_64/%s.rpm",
		constant.PuppetMajorVersion, majRelease, packageName)

	if err := s.puppet.DownloadAgent(downloadPath, url); err != nil {
		return err
	}

	installCmd := []string{fmt.Sprintf("rpm -ivh %s", downloadPath)}
	s.webtee.RemoteLogObmondo(installCmd, s.certName)

	return nil
}
