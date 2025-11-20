package provisioner

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/constant"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/helper"
	api "gitea.obmondo.com/EnableIT/linuxaid-cli/pkg/obmondo"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/pkg/puppet"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/pkg/webtee"
)

const (
	tmpDir = "/tmp"

	archAmd64 = "amd64"
	archArm64 = "arm64"
)

type Provisioner struct {
	webtee    *webtee.Webtee
	apiClient api.ObmondoClient
	puppet    *puppet.Service
	certName  string
}

// NewService creates a new Puppet installer service.
func NewService(apiClient api.ObmondoClient, puppet *puppet.Service, webtee *webtee.Webtee) *Provisioner {
	return &Provisioner{
		apiClient: apiClient,
		puppet:    puppet,
		certName:  helper.GetCertname(),
		webtee:    webtee,
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
	case "turrisos":
		s.provisionForTurris()
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

	switch runtime.GOARCH {
	case archAmd64, archArm64:
	default:
		return errors.New("unsupported system architecture")
	}

	fullPuppetVersion := fmt.Sprintf("%s-1+%s", constant.PuppetVersion, ubuntuVersion)
	packageName := fmt.Sprintf("openvox-agent_%s_%s.deb", fullPuppetVersion, runtime.GOARCH)
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

	runtimeArch := runtime.GOARCH
	switch runtimeArch {
	case archAmd64:
		runtimeArch = "x86_64"
	case archArm64:
		runtimeArch = "aarch64"
	default:
		return errors.New("unsupported system architecture")
	}

	fullPuppetVersion := fmt.Sprintf("%s-1.el%s", constant.PuppetVersion, majRelease)
	packageName := fmt.Sprintf("openvox-agent-%s.%s", fullPuppetVersion, runtimeArch)
	downloadPath := filepath.Join(tmpDir, packageName+".rpm")
	url := fmt.Sprintf("https://repos.obmondo.com/openvox/yum/%s/el/%s/%s/%s.rpm",
		constant.PuppetMajorVersion, majRelease, runtimeArch, packageName)

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

	runtimeArch := runtime.GOARCH
	switch runtimeArch {
	case archAmd64:
		runtimeArch = "x86_64"
	case archArm64:
		runtimeArch = "aarch64"
	default:
		return errors.New("unsupported system architecture")
	}

	fullPuppetVersion := fmt.Sprintf("%s-1.sles%s", constant.PuppetVersion, majRelease)
	packageName := fmt.Sprintf("openvox-agent-%s.%s", fullPuppetVersion, runtimeArch)
	downloadPath := filepath.Join(tmpDir, packageName+".rpm")
	url := fmt.Sprintf("https://repos.obmondo.com/openvox/sles/%s/%s/%s/%s.rpm",
		constant.PuppetMajorVersion, majRelease, runtimeArch, packageName)

	if err := s.puppet.DownloadAgent(downloadPath, url); err != nil {
		return err
	}

	installCmd := []string{fmt.Sprintf("rpm -ivh %s", downloadPath)}
	s.webtee.RemoteLogObmondo(installCmd, s.certName)

	return nil
}

// provisionForTurris installs puppet via gem on TurrisOS
func (s *Provisioner) provisionForTurris() {
	s.webtee.RemoteLogObmondo([]string{"opkg update"}, s.certName)
	s.webtee.RemoteLogObmondo([]string{"opkg install ruby ruby-stdlib ruby-dev ruby-gems"}, s.certName)

	installCmd := []string{fmt.Sprintf("gem install -v %s --no-document openvox", constant.OpenvoxAgentVersionTurris)}
	s.webtee.RemoteLogObmondo(installCmd, s.certName)
}
