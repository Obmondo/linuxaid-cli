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

func DebianPuppetAgent(obmondoAPI api.ObmondoClient) {
	helper.RequireUbuntuCodeNameEnv()

	certName := config.GetCertName()
	codeName := os.Getenv("UBUNTU_CODENAME")
	webtee.RemoteLogObmondo(obmondoAPI, []string{"apt update"}, certName)
	webtee.RemoteLogObmondo(obmondoAPI, []string{"apt install -y iptables"}, certName)

	tempDir := helper.TempDir()

	// nolint: errcheck
	defer os.RemoveAll(tempDir)
	fullPuppetVersion := fmt.Sprintf("%s%s", constant.PuppetVersion, codeName)
	packageName := fmt.Sprintf("puppet-agent_%s_amd64.deb", fullPuppetVersion)
	downloadPath := fmt.Sprintf("%s/%s", tempDir, packageName)
	url := fmt.Sprintf("https://repos.obmondo.com/puppetlabs/apt/pool/%s/%s/p/puppet-agent/%s", codeName, constant.PuppetMajorVersion, packageName)

	puppet.DownloadPuppetAgent(obmondoAPI, downloadPath, url)

	// Install the package
	installCmd := []string{fmt.Sprintf("apt install -y %s", downloadPath)}
	webtee.RemoteLogObmondo(obmondoAPI, installCmd, certName)
}
