package utils

import (
	"fmt"
	"os"

	"go-scripts/config"
	"go-scripts/constants"
	"go-scripts/pkg/puppet"
	"go-scripts/pkg/webtee"
	"go-scripts/utils"
)

func SusePuppetAgent() {
	certName := config.GetCertName()
	webtee.RemoteLogObmondo([]string{"zypper install -y iptables"}, certName)

	majRelease := utils.GetMajorRelease()
	tempDir := utils.TempDir()

	defer os.RemoveAll(tempDir)
	fullPuppetVersion := fmt.Sprintf("%s.sles%s", constants.PuppetVersion, majRelease)
	packageName := fmt.Sprintf("puppet-agent-%s.x86_64", fullPuppetVersion)
	downloadPath := fmt.Sprintf("%s/%s.rpm", tempDir, packageName)
	url := fmt.Sprintf("https://repos.obmondo.com/puppetlabs/sles/%s/%s/x86_64/%s.rpm", constants.PuppetMajorVersion, majRelease, packageName)

	puppet.DownloadPuppetAgent(downloadPath, url)

	// Install the package
	installCmd := []string{fmt.Sprintf("rpm -ivh %s", downloadPath)}
	webtee.RemoteLogObmondo(installCmd, certName)

}
