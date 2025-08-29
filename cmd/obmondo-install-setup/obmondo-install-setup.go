package main

import (
	"log/slog"
	"os"

	"gitea.obmondo.com/go-scripts/pkg/disk"
	"gitea.obmondo.com/go-scripts/pkg/prettyfmt"
	"gitea.obmondo.com/go-scripts/pkg/puppet"
	"gitea.obmondo.com/go-scripts/pkg/webtee"

	"gitea.obmondo.com/go-scripts/config"
	"gitea.obmondo.com/go-scripts/constants"
	"gitea.obmondo.com/go-scripts/helper"
	osutil "gitea.obmondo.com/go-scripts/helper/os"
)

func obmondoInstallSetup() {
	certName := config.GetCertName()
	puppetServer := config.GetPupeptServer()

	// Sanity check
	helper.LoadOSReleaseEnv()
	helper.RequireRootUser()

	// Check required envs and OS
	helper.RequireOSNameEnv()
	helper.RequireOSVersionEnv()
	if _, err := helper.IsSupportedOS(); err != nil {
		slog.Error("OS not supported", slog.String("err", err.Error()))
		os.Exit(1)
	}

	if err := disk.CheckDiskSize(); err != nil {
		prettyfmt.PrettyFmt(prettyfmt.FontRed("check disk size failed: ", err.Error()))
	}

	envErr := os.Setenv("PATH", constants.PuppetPath)
	if envErr != nil {
		slog.Error("failed to set the PATH env, exiting")
		os.Exit(1)
	}

	webtee.RemoteLogObmondo([]string{"echo Starting Obmondo Setup "}, certName)
	prettyfmt.PrettyFmt("\n ", prettyfmt.IconGear, " ", prettyfmt.FontWhite("Configuring Linuxaid on"), prettyfmt.FontYellow(certName), prettyfmt.FontWhite("with puppetserver"), prettyfmt.FontYellow(puppetServer), "\n")

	// check if agent disable file exists
	if _, err := os.Stat(constants.AgentDisabledLockFile); err == nil {
		prettyfmt.PrettyFmt(prettyfmt.FontRed("puppet has been disabled from the existing setup, can't proceed\npuppet agent --enable will enable the puppet agent"), "\n")
		webtee.RemoteLogObmondo([]string{"echo Exiting, puppet-agent is already installed and set to disabled"}, certName)
		os.Exit(0)
	}

	prettyfmt.PrettyFmt("  ", prettyfmt.FontGreen(prettyfmt.IconCheck), " ", prettyfmt.FontWhite("Compatibility Check Successful"))

	// Pre-requisites
	distribution := os.Getenv("ID")
	switch distribution {
	case "ubuntu", "debian":
		osutil.DebianPuppetAgent()
	case "sles":
		osutil.SusePuppetAgent()
	case "centos", "rhel":
		osutil.RedHatPuppetAgent()
	default:
		slog.Error("unknown distribution, exiting")
		os.Exit(1)
	}

	prettyfmt.PrettyFmt("  ", prettyfmt.FontGreen(prettyfmt.IconCheck), " ", prettyfmt.FontWhite("Successfully Installed Puppet"))

	puppet.DisablePuppetAgentService()
	puppet.ConfigurePuppetAgent()
	puppet.FacterNewSetup()

	prettyfmt.PrettyFmt("  ", prettyfmt.FontGreen(prettyfmt.IconCheck), " ", prettyfmt.FontWhite("Successfully Configured Puppet"))

	puppet.WaitForPuppetAgent()
	puppet.RunPuppetAgent(true, "noop")

	prettyfmt.PrettyFmt("  ", prettyfmt.FontGreen(prettyfmt.IconCheck), " ", prettyfmt.FontWhite("Puppet Ran Successfully"))

	prettyfmt.PrettyFmt("\n  ", prettyfmt.IconIceCream, prettyfmt.FontGreen("Success!"))

	webtee.RemoteLogObmondo([]string{"echo Finished Obmondo Setup "}, certName)

	prettyfmt.PrettyFmt(prettyfmt.FontWhite("\n    Head to "), prettyfmt.FontBlue("https://obmondo.com/user/servers"), prettyfmt.FontWhite("to add role and subscription."), "\n")
}
