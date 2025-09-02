package helper

import (
	"fmt"
	"os"

	"gitea.obmondo.com/EnableIT/go-scripts/config"
	"gitea.obmondo.com/EnableIT/go-scripts/constant"
	"gitea.obmondo.com/EnableIT/go-scripts/helper"
	api "gitea.obmondo.com/EnableIT/go-scripts/pkg/obmondo"
	"gitea.obmondo.com/EnableIT/go-scripts/pkg/puppet"
	"gitea.obmondo.com/EnableIT/go-scripts/pkg/webtee"
)

func RedHatPuppetAgent(obmondoAPI api.ObmondoClient) {
	certName := config.GetCertName()
	webtee.RemoteLogObmondo(obmondoAPI, []string{"yum install -y iptables"}, certName)

	majRelease := helper.GetMajorRelease()
	tempDir := helper.TempDir()

	// nolint: errcheck
	defer os.RemoveAll(tempDir)
	fullPuppetVersion := fmt.Sprintf("%s.el%s", constant.PuppetVersion, majRelease)
	packageName := fmt.Sprintf("puppet-agent-%s.x86_64", fullPuppetVersion)
	downloadPath := fmt.Sprintf("%s/%s.rpm", tempDir, packageName)
	url := fmt.Sprintf("https://repos.obmondo.com/puppetlabs/yum/%s/el/%s/x86_64/%s.rpm", constant.PuppetMajorVersion, majRelease, packageName)
	puppet.DownloadPuppetAgent(obmondoAPI, downloadPath, url)

	// Install the package
	installCmd := []string{fmt.Sprintf("yum install %s -y", downloadPath)}
	webtee.RemoteLogObmondo(obmondoAPI, installCmd, certName)

}
