package utils

import (
	"fmt"
	"os"

	"gitea.obmondo.com/go-scripts/config"
	"gitea.obmondo.com/go-scripts/constants"
	"gitea.obmondo.com/go-scripts/pkg/puppet"
	"gitea.obmondo.com/go-scripts/pkg/webtee"
	"gitea.obmondo.com/go-scripts/utils"
)

func DebianPuppetAgent() {
	utils.RequireUbuntuCodeNameEnv()

	certName := config.GetCertName()
	codeName := os.Getenv("UBUNTU_CODENAME")
	webtee.RemoteLogObmondo([]string{"apt update"}, certName)
	webtee.RemoteLogObmondo([]string{"apt install -y iptables"}, certName)

	tempDir := utils.TempDir()

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
