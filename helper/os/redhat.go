package helper

import (
	"fmt"
	"os"

	"gitea.obmondo.com/go-scripts/config"
	"gitea.obmondo.com/go-scripts/constant"
	"gitea.obmondo.com/go-scripts/helper"
	"gitea.obmondo.com/go-scripts/pkg/puppet"
	"gitea.obmondo.com/go-scripts/pkg/webtee"
)

func RedHatPuppetAgent() {
	certName := config.GetCertName()
	webtee.RemoteLogObmondo([]string{"yum install -y iptables"}, certName)

	majRelease := helper.GetMajorRelease()
	tempDir := helper.TempDir()

	// nolint: errcheck
	defer os.RemoveAll(tempDir)
	fullPuppetVersion := fmt.Sprintf("%s.el%s", constant.PuppetVersion, majRelease)
	packageName := fmt.Sprintf("puppet-agent-%s.x86_64", fullPuppetVersion)
	downloadPath := fmt.Sprintf("%s/%s.rpm", tempDir, packageName)
	url := fmt.Sprintf("https://repos.obmondo.com/puppetlabs/yum/%s/el/%s/x86_64/%s.rpm", constant.PuppetMajorVersion, majRelease, packageName)
	puppet.DownloadPuppetAgent(downloadPath, url)

	// Install the package
	installCmd := []string{fmt.Sprintf("yum install %s -y", downloadPath)}
	webtee.RemoteLogObmondo(installCmd, certName)

}
