package utils

import (
	"fmt"
	"log/slog"
	"os"

	constants "go-scripts/constants"
	"go-scripts/pkg/puppet"
	webtee "go-scripts/pkg/webtee"
	utils "go-scripts/utils"

	"github.com/bitfield/script"
)

func RedHatPuppetAgent() {
	certName := os.Getenv("CERTNAME")
	webtee.RemoteLogObmondo([]string{"yum install -y iptables"}, certName)

	majRelease := utils.GetMajorRelease()
	tempDir := utils.TempDir()

	defer os.RemoveAll(tempDir)
	fullPuppetVersion := fmt.Sprintf("%s.el%s", constants.PuppetVersion, majRelease)
	packageName := fmt.Sprintf("puppet-agent-%s.x86_64", fullPuppetVersion)
	downloadPath := fmt.Sprintf("%s/%s.rpm", tempDir, packageName)
	url := fmt.Sprintf("https://repos.obmondo.com/puppetlabs/yum/puppet7/el/%s/x86_64/%s.rpm", majRelease, packageName)

	isPuppetInstalled := fmt.Sprintf("rpm -q %s", packageName)

	pipe := script.Exec(isPuppetInstalled)
	if err := pipe.Wait(); err != nil {
		slog.Error("failed to verify if puppet is installed", slog.String("error", err.Error()))
	}
	exitStatus := pipe.ExitStatus()

	if exitStatus != 0 {
		puppet.DownloadPuppetAgent(downloadPath, url)

		// Install the package
		installCmd := []string{fmt.Sprintf("rpm -ivh %s", downloadPath)}
		webtee.RemoteLogObmondo(installCmd, certName)
	} else {
		webtee.RemoteLogObmondo([]string{"echo puppet-agent is already installed"}, certName)
	}

}
