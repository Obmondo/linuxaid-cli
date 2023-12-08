package util

import (
	"fmt"
	"os"

	constants "go-scripts/constants"
	"go-scripts/pkg/puppet"
	webtee "go-scripts/pkg/webtee"
	util "go-scripts/util"

	"github.com/bitfield/script"
)

func SusePuppetAgent() {
	certName := os.Getenv("CERTNAME")
	webtee.RemoteLogObmondo([]string{"zypper install -y epel-release iptables ca-certificates openssl"}, certName)

	majRelease := util.GetMajorRelease()
	tempDir := util.TempDir()

	defer os.RemoveAll(tempDir)
	fullPuppetVersion := fmt.Sprintf("%s.sles%s", constants.PuppetVersion, majRelease)
	packageName := fmt.Sprintf("puppet-agent-%s.x86_64", fullPuppetVersion)
	downloadPath := fmt.Sprintf("%s/%s.rpm", tempDir, packageName)
	url := fmt.Sprintf("https://repos.obmondo.com/puppetlabs/sles/puppet7/%s/x86_64/%s.rpm", majRelease, packageName)

	isPuppetInstalled := fmt.Sprintf("rpm -qa | grep -q %s", packageName)

	pipe := script.Exec(isPuppetInstalled)
	pipe.Wait()
	exitStatus := pipe.ExitStatus()

	if exitStatus != 0 {
		puppet.DownloadPuppetAgent(downloadPath, url)

		// Install the package
		installCmd := []string{fmt.Sprintf("zypper install -y %s", downloadPath)}
		webtee.RemoteLogObmondo(installCmd, certName)
	} else {
		webtee.RemoteLogObmondo([]string{"echo puppet-agent is already installed"}, certName)
	}

}
