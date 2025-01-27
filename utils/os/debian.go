package utils

import (
	"fmt"
	"os"

	constants "go-scripts/constants"
	puppet "go-scripts/pkg/puppet"
	webtee "go-scripts/pkg/webtee"
	utils "go-scripts/utils"
)

func DebianPuppetAgent() {
	utils.CheckUbuntuCodeNameEnv()

	certName := os.Getenv("CERTNAME")
	codeName := os.Getenv("UBUNTU_CODENAME")
	webtee.RemoteLogObmondo([]string{"apt update"}, certName)
	webtee.RemoteLogObmondo([]string{"apt install -y iptables"}, certName)

	tempDir := utils.TempDir()

	defer os.RemoveAll(tempDir)
	fullPuppetVersion := fmt.Sprintf("%s%s", constants.PuppetVersion, codeName)
	packageName := fmt.Sprintf("puppet-agent_%s_amd64.deb", fullPuppetVersion)
	downloadPath := fmt.Sprintf("%s/%s", tempDir, packageName)
	url := fmt.Sprintf("https://repos.obmondo.com/puppetlabs/apt/pool/%s/puppet8/p/puppet-agent/%s", codeName, packageName)

	puppet.DownloadPuppetAgent(downloadPath, url)

	// Install the package
	installCmd := []string{fmt.Sprintf("apt install %s", downloadPath)}
	webtee.RemoteLogObmondo(installCmd, certName)

}
