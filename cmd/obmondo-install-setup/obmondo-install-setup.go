package main

import (
	"fmt"
	"log/slog"
	"os"

	disk "go-scripts/pkg/disk"
	puppet "go-scripts/pkg/puppet"
	webtee "go-scripts/pkg/webtee"

	constants "go-scripts/constants"
	util "go-scripts/util"
	"go-scripts/util/logger"
	os_util "go-scripts/util/os"

	"github.com/schollz/progressbar/v3"
)

func main() {
	debug := true
	logger.InitLogger(debug)

	util.LoadOSReleaseEnv()

	util.CheckUser()

	// Check required envs and OS
	util.CheckCertNameEnv()
	util.CheckOSNameEnv()
	util.CheckOSVersionEnv()
	util.SupportedOS()

	if err := disk.CheckDiskSize(); err != nil {
		slog.Error("unable to check disk size", slog.String("error", err.Error()))
	}

	certName := os.Getenv("CERTNAME")
	envErr := os.Setenv("PATH", constants.PuppetPath)
	if envErr != nil {
		slog.Error("failed to set the PATH env, exiting")
		os.Exit(1)
	}

	webtee.RemoteLogObmondo([]string{"echo Starting Obmondo Setup "}, certName)

	// check if agent disable file exists
	if _, err := os.Stat(constants.AgentDisabledLockFile); err == nil {
		slog.Warn("puppet has been disabled from the existing setup, can't proceed\npuppet agent --enable will enable the puppet agent")
		webtee.RemoteLogObmondo([]string{"echo Exiting, puppet-agent is already installed and set to disabled"}, certName)
		os.Exit(0)
	}

	// Pre-requisites
	distribution := os.Getenv("ID")
	switch distribution {
	case "ubuntu", "debian":
		os_util.DebianPuppetAgent()
	case "sles":
		os_util.SusePuppetAgent()
	case "centos", "rhel":
		os_util.RedHatPuppetAgent()
	default:
		slog.Error("unknown distribution, exiting")
		os.Exit(1)
	}

	// Puppet agent setup
	bar := progressbar.Default(constants.BarProgressSize,
		"puppet-agent setup...")

	puppet.DisablePuppetAgentService()
	fiveErr := bar.Set(constants.BarSizeFive)
	if fiveErr != nil {
		slog.Error("failed to set the progressbar size")
	}

	puppet.ConfigurePuppetAgent()
	tenErr := bar.Set(constants.BarSizeTen)
	if tenErr != nil {
		slog.Error("failed to set the progressbar size")
	}

	puppet.FacterNewSetup()
	fifteenErr := bar.Set(constants.BarSizeFifteen)
	if fifteenErr != nil {
		slog.Error("failed to set the progressbar size")
	}

	puppet.WaitForPuppetAgent()
	twentyErr := bar.Set(constants.BarSizeTwenty)
	if twentyErr != nil {
		slog.Error("failed to set the progressbar size")
	}

	puppet.RunPuppetAgent(true, "noop")
	hundredErr := bar.Set(constants.BarSizeHundred)
	if hundredErr != nil {
		slog.Error("failed to set the progressbar size")
	}

	finishErr := bar.Finish()
	if finishErr != nil {
		slog.Error("failed to finish the progressbar size")
	}

	slog.Info("\ninstallation succeeded. To continue configuration please, please head to", slog.String("web", fmt.Sprintf("https://obmondo.com/server/%s", certName)))
	webtee.RemoteLogObmondo([]string{"echo Finished Obmondo Setup "}, certName)
}
