package main

import (
	"log"
	"os"

	disk "go-scripts/pkg/disk"
	puppet "go-scripts/pkg/puppet"
	webtee "go-scripts/pkg/webtee"

	"github.com/schollz/progressbar/v3"
	constants "go-scripts/constants"
	util "go-scripts/util"
	os_util "go-scripts/util/os"
)

func main() {
	util.LoadOSReleaseEnv()

	util.CheckUser()

	// Check required envs and OS
	util.CheckCertNameEnv()
	util.CheckOSNameEnv()
	util.CheckOSVersionEnv()
	util.SupportedOS()

	disk.CheckDiskSize()

	certName := os.Getenv("CERTNAME")
	envErr := os.Setenv("PATH", constants.PuppetPath)
	if envErr != nil {
		log.Fatal("failed to set the PATH env, exiting")
	}

	webtee.RemoteLogObmondo([]string{"echo Starting Obmondo Setup "}, certName)

	// check if agent disable file exists
	if _, err := os.Stat(constants.AgentDisabledLockFile); err == nil {
		log.Println("Puppet has been disabled from the existing setup, can't proceed")
		log.Println("puppet agent --enable will enable the puppet agent")
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
		log.Fatal("Unknown distribution, exiting")
	}

	// Puppet agent setup
	bar := progressbar.Default(constants.BarProgressSize,
		"puppet-agent setup...")

	puppet.DisablePuppetAgentService()
	fiveErr := bar.Set(constants.BarSizeFive)
	if fiveErr != nil {
		log.Println("failed to set the progressbar size")
	}

	puppet.ConfigurePuppetAgent()
	tenErr := bar.Set(constants.BarSizeTen)
	if tenErr != nil {
		log.Println("failed to set the progressbar size")
	}

	puppet.FacterNewSetup()
	fifteenErr := bar.Set(constants.BarSizeFifteen)
	if fifteenErr != nil {
		log.Println("failed to set the progressbar size")
	}

	puppet.WaitForPuppetAgent()
	twentyErr := bar.Set(constants.BarSizeTwenty)
	if twentyErr != nil {
		log.Println("failed to set the progressbar size")
	}

	puppet.RunPuppetAgent(true, "noop")
	hundredErr := bar.Set(constants.BarSizeHundred)
	if hundredErr != nil {
		log.Println("failed to set the progressbar size")
	}

	finishErr := bar.Finish()
	if finishErr != nil {
		log.Println("failed to finish the progressbar size")
	}

	log.Println("\nInstallation succeeded. Please head to https://obmondo.com/server/" + certName + " to continue configuration.")
	webtee.RemoteLogObmondo([]string{"echo Finished Obmondo Setup "}, certName)
}
