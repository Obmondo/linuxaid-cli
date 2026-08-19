package provisioner

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/constant"
	api "gitea.obmondo.com/EnableIT/linuxaid-cli/internal/obmondo"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/puppet"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/system"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/webtee"
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
func NewService(apiClient api.ObmondoClient, puppet *puppet.Service, webtee *webtee.Webtee, certname string) *Provisioner {
	return &Provisioner{
		apiClient: apiClient,
		puppet:    puppet,
		certName:  certname,
		webtee:    webtee,
	}
}

func (s *Provisioner) ProvisionPuppet() error {
	switch os.Getenv("ID") {
	case system.ConstDistributionNameUbuntu, system.ConstDistributionNameDebian:
		if err := s.provisionForDebian(); err != nil {
			return fmt.Errorf("failed to install puppet: %w", err)
		}
	case system.ConstDistributionNameSLES:
		if err := s.provisionForSuse(); err != nil {
			return fmt.Errorf("failed to install puppet: %w", err)
		}
	case system.ConstDistributionNameCentOS, system.ConstDistributionNameRHEL, system.ConstDistributionNameRocky, system.ConstDistributionNameOracleLinux:
		if err := s.provisionForRedHat(); err != nil {
			return fmt.Errorf("failed to install puppet: %w", err)
		}
	case system.ConstDistributionNameTurrisOS:
		if err := s.provisionForTurris(); err != nil {
			return fmt.Errorf("failed to install puppet: %w", err)
		}
	default:
		return errors.New("unknown distribution")
	}

	return nil
}

// provisionForDebian installs puppet-agent on Ubuntu/Debian systems
func (s *Provisioner) provisionForDebian() error {
	if err := system.RequireUbuntuCodeNameEnv(); err != nil {
		return err
	}

	codeName := os.Getenv("UBUNTU_CODENAME")
	if err := s.webtee.RemoteLogObmondo([]string{"apt update"}, s.certName); err != nil {
		return err
	}
	if err := s.webtee.RemoteLogObmondo([]string{"apt install -y iptables"}, s.certName); err != nil {
		return err
	}
	var ubuntuVersion string
	switch codeName {
	case "jammy":
		ubuntuVersion = "ubuntu22.04"
	case "noble":
		ubuntuVersion = "ubuntu24.04"
	case "resolute":
		ubuntuVersion = "ubuntu26.04"
	}

	switch runtime.GOARCH {
	case archAmd64, archArm64:
	default:
		return errors.New("unsupported system architecture")
	}

	fullPuppetVersion := fmt.Sprintf("%s-1+%s", constant.OpenvoxVersion, ubuntuVersion)
	packageName := fmt.Sprintf("openvox-agent_%s_%s.deb", fullPuppetVersion, runtime.GOARCH)
	downloadPath := filepath.Join(tmpDir, packageName)
	url := fmt.Sprintf("https://repos.obmondo.com/openvox/apt/pool/%s/o/openvox-agent/%s",
		constant.PuppetMajorVersion, packageName)
	if err := s.puppet.DownloadAgent(downloadPath, url); err != nil {
		return err
	}

	installCmd := []string{fmt.Sprintf("apt install -y %s", downloadPath)}
	if err := s.webtee.RemoteLogObmondo(installCmd, s.certName); err != nil {
		return err
	}

	return nil
}

// provisionForRedHat installs puppet-agent on RHEL/CentOS systems
func (s *Provisioner) provisionForRedHat() error {
	if err := s.webtee.RemoteLogObmondo([]string{"yum install -y iptables"}, s.certName); err != nil {
		return err
	}

	majRelease := system.GetMajorRelease()

	runtimeArch := runtime.GOARCH
	switch runtimeArch {
	case archAmd64:
		runtimeArch = "x86_64"
	case archArm64:
		runtimeArch = "aarch64"
	default:
		return errors.New("unsupported system architecture")
	}

	fullPuppetVersion := fmt.Sprintf("%s-1.el%s", constant.OpenvoxVersion, majRelease)
	packageName := fmt.Sprintf("openvox-agent-%s.%s", fullPuppetVersion, runtimeArch)
	downloadPath := filepath.Join(tmpDir, packageName+".rpm")
	url := fmt.Sprintf("https://repos.obmondo.com/openvox/yum/%s/el/%s/%s/%s.rpm",
		constant.PuppetMajorVersion, majRelease, runtimeArch, packageName)

	if err := s.puppet.DownloadAgent(downloadPath, url); err != nil {
		return err
	}

	installCmd := []string{fmt.Sprintf("yum install %s -y", downloadPath)}
	if err := s.webtee.RemoteLogObmondo(installCmd, s.certName); err != nil {
		return err
	}

	return nil
}

// provisionForSuse installs puppet-agent on SUSE systems
func (s *Provisioner) provisionForSuse() error {
	if err := s.webtee.RemoteLogObmondo([]string{"zypper install -y iptables"}, s.certName); err != nil {
		return err
	}

	majRelease := system.GetMajorRelease()

	runtimeArch := runtime.GOARCH
	switch runtimeArch {
	case archAmd64:
		runtimeArch = "x86_64"
	case archArm64:
		runtimeArch = "aarch64"
	default:
		return errors.New("unsupported system architecture")
	}

	fullPuppetVersion := fmt.Sprintf("%s-1.sles%s", constant.OpenvoxVersion, majRelease)
	packageName := fmt.Sprintf("openvox-agent-%s.%s", fullPuppetVersion, runtimeArch)
	downloadPath := filepath.Join(tmpDir, packageName+".rpm")
	url := fmt.Sprintf("https://repos.obmondo.com/openvox/sles/%s/%s/%s/%s.rpm",
		constant.PuppetMajorVersion, majRelease, runtimeArch, packageName)

	if err := s.puppet.DownloadAgent(downloadPath, url); err != nil {
		return err
	}

	installCmd := []string{fmt.Sprintf("rpm -ivh %s", downloadPath)}
	if err := s.webtee.RemoteLogObmondo(installCmd, s.certName); err != nil {
		return err
	}

	return nil
}

// provisionForTurris installs puppet via gem on TurrisOS
func (s *Provisioner) provisionForTurris() error {
	if err := s.webtee.RemoteLogObmondo([]string{"opkg update"}, s.certName); err != nil {
		return err
	}
	if err := s.webtee.RemoteLogObmondo([]string{"opkg install ruby ruby-stdlib ruby-dev ruby-gems"}, s.certName); err != nil {
		return err
	}

	installCmd := []string{fmt.Sprintf("gem install -v %s --no-document openvox", constant.OpenvoxVersion)}
	if err := s.webtee.RemoteLogObmondo(installCmd, s.certName); err != nil {
		return err
	}

	return nil
}
