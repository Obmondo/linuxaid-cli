package main

import (
	"log/slog"
	"os"

	"gitea.obmondo.com/EnableIT/go-scripts/helper/provisioner"
	"gitea.obmondo.com/EnableIT/go-scripts/pkg/disk"
	api "gitea.obmondo.com/EnableIT/go-scripts/pkg/obmondo"
	"gitea.obmondo.com/EnableIT/go-scripts/pkg/prettyfmt"
	"gitea.obmondo.com/EnableIT/go-scripts/pkg/puppet"
	"gitea.obmondo.com/EnableIT/go-scripts/pkg/webtee"

	"gitea.obmondo.com/EnableIT/go-scripts/config"
	"gitea.obmondo.com/EnableIT/go-scripts/constant"
	"gitea.obmondo.com/EnableIT/go-scripts/helper"
)

func obmondoInstallSetup() {
	certName := config.GetCertName()
	puppetServer := config.GetPupeptServer()

	obmondoAPI := api.NewObmondoClient(true)
	webtee := webtee.NewWebtee(obmondoAPI)
	puppetService := puppet.NewService(obmondoAPI, webtee)
	provisioner := provisioner.NewService(obmondoAPI, puppetService)

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

	// Check if Puppetserver is alive and active
	if err := puppetService.CheckServerStatus(); err != nil {
		slog.Error("puppet server check failed", slog.Any("error", err))
		os.Exit(1)
	}

	envErr := os.Setenv("PATH", constant.PuppetPath)
	if envErr != nil {
		slog.Error("failed to set the PATH env, exiting")
		os.Exit(1)
	}

	webtee.RemoteLogObmondo([]string{"echo Starting Obmondo Setup "}, certName)
	prettyfmt.PrettyFmt("\n ", prettyfmt.IconGear, " ", prettyfmt.FontWhite("Configuring Linuxaid on"), prettyfmt.FontYellow(certName), prettyfmt.FontWhite("with puppetserver"), prettyfmt.FontYellow(puppetServer), "\n")

	// check if agent disable file exists
	if _, err := os.Stat(constant.AgentDisabledLockFile); err == nil {
		prettyfmt.PrettyFmt(prettyfmt.FontRed("puppet has been disabled from the existing setup, can't proceed\npuppet agent --enable will enable the puppet agent"), "\n")
		webtee.RemoteLogObmondo([]string{"echo Exiting, puppet-agent is already installed and set to disabled"}, certName)
		os.Exit(0)
	}

	prettyfmt.PrettyFmt("  ", prettyfmt.FontGreen(prettyfmt.IconCheck), " ", prettyfmt.FontWhite("Compatibility Check Successful"))

	provisioner.ProvisionPuppet()

	prettyfmt.PrettyFmt("  ", prettyfmt.FontGreen(prettyfmt.IconCheck), " ", prettyfmt.FontWhite("Successfully Installed Puppet"))

	puppetService.DisableAgentService()
	puppetService.ConfigureAgent()
	puppetService.FacterNewSetup()

	prettyfmt.PrettyFmt("  ", prettyfmt.FontGreen(prettyfmt.IconCheck), " ", prettyfmt.FontWhite("Successfully Configured Puppet"))

	puppetService.WaitForAgent(constant.PuppetWaitForCertTimeOut)
	puppetService.RunAgent(true, "noop")
	// nolint:errcheck
	obmondoAPI.UpdatePuppetLastRunReport()

	prettyfmt.PrettyFmt("  ", prettyfmt.FontGreen(prettyfmt.IconCheck), " ", prettyfmt.FontWhite("Puppet Ran Successfully"))

	prettyfmt.PrettyFmt("\n  ", prettyfmt.IconIceCream, prettyfmt.FontGreen("Success!"))

	webtee.RemoteLogObmondo([]string{"echo Finished Obmondo Setup "}, certName)

	prettyfmt.PrettyFmt(prettyfmt.FontWhite("\n    Head to "), prettyfmt.FontBlue("https://obmondo.com/user/servers"), prettyfmt.FontWhite("to add role and subscription."), "\n")
}
