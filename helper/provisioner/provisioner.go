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
	var err error
	switch os.Getenv("ID") {
	case helper.ConstDistributionNameUbuntu, helper.ConstDistributionNameDebian:
		err = s.provisionForDebian()
	case helper.ConstDistributionNameSLES:
		err = s.provisionForSuse()
	case helper.ConstDistributionNameCentOS, helper.ConstDistributionNameRHEL, helper.ConstDistributionNameRocky, helper.ConstDistributionNameOracleLinux:
		err = s.provisionForRedHat()
	case helper.ConstDistributionNameTurrisOS:
		s.provisionForTurris()
	default:
		slog.Error("unknown distribution, exiting")
		os.Exit(1)
	}

	if err != nil {
		slog.Error("failed to install puppet", slog.Any("error", err))
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
	s.webtee.RemoteLogObmondo(installCmd, s.certName)

	return nil
}

// rpmSpec captures how an RPM-based distribution names, downloads, and
// installs the openvox-agent package.
type rpmSpec struct {
	prereqCmd     string // installs iptables before the agent
	versionFmt    string // verbs: openvox version, major release
	repoURLFmt    string // verbs: puppet major version, major release, arch, package name
	installCmdFmt string // verb: download path
}

// provisionForRedHat installs puppet-agent on RHEL/CentOS systems
func (s *Provisioner) provisionForRedHat() error {
	return s.provisionRPMBased(rpmSpec{
		prereqCmd:     "yum install -y iptables",
		versionFmt:    "%s-1.el%s",
		repoURLFmt:    "https://repos.obmondo.com/openvox/yum/%s/el/%s/%s/%s.rpm",
		installCmdFmt: "yum install %s -y",
	})
}

// provisionForSuse installs puppet-agent on SUSE systems
func (s *Provisioner) provisionForSuse() error {
	return s.provisionRPMBased(rpmSpec{
		prereqCmd:     "zypper install -y iptables",
		versionFmt:    "%s-1.sles%s",
		repoURLFmt:    "https://repos.obmondo.com/openvox/sles/%s/%s/%s/%s.rpm",
		installCmdFmt: "rpm -ivh %s",
	})
}

func (s *Provisioner) provisionRPMBased(spec rpmSpec) error {
	s.webtee.RemoteLogObmondo([]string{spec.prereqCmd}, s.certName)

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

	fullPuppetVersion := fmt.Sprintf(spec.versionFmt, constant.OpenvoxVersion, majRelease)
	packageName := fmt.Sprintf("openvox-agent-%s.%s", fullPuppetVersion, runtimeArch)
	downloadPath := filepath.Join(tmpDir, packageName+".rpm")
	url := fmt.Sprintf(spec.repoURLFmt, constant.PuppetMajorVersion, majRelease, runtimeArch, packageName)

	if err := s.puppet.DownloadAgent(downloadPath, url); err != nil {
		return err
	}

	installCmd := []string{fmt.Sprintf(spec.installCmdFmt, downloadPath)}
	s.webtee.RemoteLogObmondo(installCmd, s.certName)

	return nil
}

// provisionForTurris installs puppet via gem on TurrisOS
func (s *Provisioner) provisionForTurris() {
	s.webtee.RemoteLogObmondo([]string{"opkg update"}, s.certName)
	s.webtee.RemoteLogObmondo([]string{"opkg install ruby ruby-stdlib ruby-dev ruby-gems"}, s.certName)

	installCmd := []string{fmt.Sprintf("gem install -v %s --no-document openvox", constant.OpenvoxVersion)}
	s.webtee.RemoteLogObmondo(installCmd, s.certName)
}
