package helper

import (
	"fmt"
	"os"

	"gitea.obmondo.com/go-scripts/config"
	"gitea.obmondo.com/go-scripts/constants"
	"gitea.obmondo.com/go-scripts/helper"
	"gitea.obmondo.com/go-scripts/pkg/puppet"
	"gitea.obmondo.com/go-scripts/pkg/webtee"
)

func DebianPuppetAgent() {
	helper.RequireUbuntuCodeNameEnv()

	certName := config.GetCertName()
	codeName := os.Getenv("UBUNTU_CODENAME")
	webtee.RemoteLogObmondo([]string{"apt update"}, certName)
	webtee.RemoteLogObmondo([]string{"apt install -y iptables"}, certName)

	tempDir := helper.TempDir()

	defer os.RemoveAll(tempDir)
	fullPuppetVersion := fmt.Sprintf("%s%s", constants.PuppetVersion, codeName)
	packageName := fmt.Sprintf("puppet-agent_%s_amd64.deb", fullPuppetVersion)
	downloadPath := fmt.Sprintf("%s/%s", tempDir, packageName)
	url := fmt.Sprintf("https://repos.obmondo.com/puppetlabs/apt/pool/%s/%s/p/puppet-agent/%s", codeName, constants.PuppetMajorVersion, packageName)

	puppet.DownloadPuppetAgent(downloadPath, url)

	// Install the package
	installCmd := []string{fmt.Sprintf("apt install -y %s", downloadPath)}
	webtee.RemoteLogObmondo(installCmd, certName)
}
