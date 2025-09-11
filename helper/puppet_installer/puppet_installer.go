package puppetinstaller

import (
	"fmt"
	"os"
	"path/filepath"

	"gitea.obmondo.com/EnableIT/go-scripts/config"
	"gitea.obmondo.com/EnableIT/go-scripts/constant"
	"gitea.obmondo.com/EnableIT/go-scripts/helper"
	api "gitea.obmondo.com/EnableIT/go-scripts/pkg/obmondo"
	"gitea.obmondo.com/EnableIT/go-scripts/pkg/puppet"
	"gitea.obmondo.com/EnableIT/go-scripts/pkg/webtee"
)

type InstallerService struct {
	webtee    *webtee.Webtee
	apiClient api.ObmondoClient
	puppet    *puppet.Service
	certName  string
}

const tmpDir = "/tmp"

// NewInstallerService creates a new Puppet installer service.
func NewInstallerService(apiClient api.ObmondoClient, puppet *puppet.Service) *InstallerService {
	return &InstallerService{
		apiClient: apiClient,
		puppet:    puppet,
		certName:  config.GetCertName(),
	}
}

// InstallDebian installs puppet-agent on Ubuntu/Debian systems
func (s *InstallerService) InstallDebian() error {
	helper.RequireUbuntuCodeNameEnv()

	codeName := os.Getenv("UBUNTU_CODENAME")
	s.webtee.RemoteLogObmondo([]string{"apt update"}, s.certName)
	s.webtee.RemoteLogObmondo([]string{"apt install -y iptables"}, s.certName)

	fullPuppetVersion := fmt.Sprintf("%s%s", constant.PuppetVersion, codeName)
	packageName := fmt.Sprintf("puppet-agent_%s_amd64.deb", fullPuppetVersion)
	downloadPath := filepath.Join(tmpDir, packageName)
	url := fmt.Sprintf("https://repos.obmondo.com/puppetlabs/apt/pool/%s/%s/p/puppet-agent/%s",
		codeName, constant.PuppetMajorVersion, packageName)

	if err := s.puppet.DownloadAgent(downloadPath, url); err != nil {
		return err
	}

	installCmd := []string{fmt.Sprintf("apt install -y %s", downloadPath)}
	s.webtee.RemoteLogObmondo(installCmd, s.certName)

	return nil
}

// InstallRedHat installs puppet-agent on RHEL/CentOS systems
func (s *InstallerService) InstallRedHat() error {
	s.webtee.RemoteLogObmondo([]string{"yum install -y iptables"}, s.certName)

	majRelease := helper.GetMajorRelease()

	fullPuppetVersion := fmt.Sprintf("%s.el%s", constant.PuppetVersion, majRelease)
	packageName := fmt.Sprintf("puppet-agent-%s.x86_64", fullPuppetVersion)
	downloadPath := filepath.Join(tmpDir, packageName+".rpm")
	url := fmt.Sprintf("https://repos.obmondo.com/puppetlabs/yum/%s/el/%s/x86_64/%s.rpm",
		constant.PuppetMajorVersion, majRelease, packageName)

	if err := s.puppet.DownloadAgent(downloadPath, url); err != nil {
		return err
	}

	installCmd := []string{fmt.Sprintf("yum install %s -y", downloadPath)}
	s.webtee.RemoteLogObmondo(installCmd, s.certName)

	return nil
}

// InstallSuse installs puppet-agent on SUSE systems
func (s *InstallerService) InstallSuse() error {
	s.webtee.RemoteLogObmondo([]string{"zypper install -y iptables"}, s.certName)

	majRelease := helper.GetMajorRelease()

	fullPuppetVersion := fmt.Sprintf("%s.sles%s", constant.PuppetVersion, majRelease)
	packageName := fmt.Sprintf("puppet-agent-%s.x86_64", fullPuppetVersion)
	downloadPath := filepath.Join(tmpDir, packageName+".rpm")
	url := fmt.Sprintf("https://repos.obmondo.com/puppetlabs/sles/%s/%s/x86_64/%s.rpm",
		constant.PuppetMajorVersion, majRelease, packageName)

	if err := s.puppet.DownloadAgent(downloadPath, url); err != nil {
		return err
	}

	installCmd := []string{fmt.Sprintf("rpm -ivh %s", downloadPath)}
	s.webtee.RemoteLogObmondo(installCmd, s.certName)

	return nil
}
