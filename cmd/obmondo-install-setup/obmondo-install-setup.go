package main

import (
	"flag"
	"log/slog"
	"os"

	"go-scripts/pkg/disk"
	"go-scripts/pkg/prettyfmt"
	"go-scripts/pkg/puppet"
	"go-scripts/pkg/webtee"

	"go-scripts/constants"
	"go-scripts/utils"
	"go-scripts/utils/logger"
	osutil "go-scripts/utils/os"
)

var Version string

func main() {
	versionFlag := flag.Bool("version", false, "Print version and exit")
	debugFlag := flag.Bool("debug", false, "Enable debug logs")

	flag.Parse()

	if *versionFlag {
		slog.Info("obmondo-install-setup version", "version", Version)
		os.Exit(0)
	}

	logger.InitLogger(*debugFlag)

	utils.LoadOSReleaseEnv()

	utils.CheckUser()

	// Check required envs and OS
	utils.CheckCertNameEnv()
	utils.CheckOSNameEnv()
	utils.CheckOSVersionEnv()
	utils.SupportedOS()

	if err := disk.CheckDiskSize(); err != nil {
		prettyfmt.PrettyFmt(prettyfmt.FontRed("unable to check disk size", err.Error()))
	}

	certName := os.Getenv("CERTNAME")
	envErr := os.Setenv("PATH", constants.PuppetPath)
	if envErr != nil {
		slog.Error("failed to set the PATH env, exiting")
		os.Exit(1)
	}

	webtee.RemoteLogObmondo([]string{"echo Starting Obmondo Setup "}, certName)
	prettyfmt.PrettyFmt("\n ", prettyfmt.IconGear, " ", prettyfmt.FontWhite("Configuring Linuxaid on"), prettyfmt.FontYellow(certName), "\n")

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
