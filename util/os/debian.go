package util

import (
	"fmt"
	"os"

	constants "go-scripts/constants"
	puppet "go-scripts/pkg/puppet"
	webtee "go-scripts/pkg/webtee"
	util "go-scripts/util"

	"github.com/bitfield/script"
)

func DebianPuppetAgent() {
	util.CheckUbuntuCodeNameEnv()

	certName := os.Getenv("CERTNAME")
	codeName := os.Getenv("UBUNTU_CODENAME")
	webtee.RemoteLogObmondo([]string{"apt update"}, certName)
	webtee.RemoteLogObmondo([]string{"apt install -y ca-certificates openssl"}, certName)

	tempDir := util.TempDir()

	defer os.RemoveAll(tempDir)
	fullPuppetVersion := fmt.Sprintf("%s%s", constants.PuppetVersion, codeName)
	packageName := fmt.Sprintf("puppet-agent_%s_amd64.deb", fullPuppetVersion)
	downloadPath := fmt.Sprintf("%s/%s", tempDir, packageName)
	url := fmt.Sprintf("https://repos.obmondo.com/puppetlabs/apt/pool/%s/puppet7/p/puppet-agent/%s", codeName, packageName)

	isPuppetInstalled := fmt.Sprintf("dpkg-query -Wf '${Status}\t${Version}\n' puppet-agent | cut -f2 | grep -q %s", fullPuppetVersion)

	pipe := script.Exec(isPuppetInstalled)
	pipe.Wait()
	exitStatus := pipe.ExitStatus()

	if exitStatus != 0 {
		puppet.DownloadPuppetAgent(downloadPath, url)

		// Install the package
		installCmd := []string{fmt.Sprintf("dpkg -i %s", downloadPath)}
		webtee.RemoteLogObmondo(installCmd, certName)
	} else {
		puppet.PuppetAgentIsInstalled()
	}

}
