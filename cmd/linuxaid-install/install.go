package main

import (
	"bufio"
	"log/slog"
	"os"
	"strings"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/helper/logger"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/helper/progress"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/helper/provisioner"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/pkg/disk"
	api "gitea.obmondo.com/EnableIT/linuxaid-cli/pkg/obmondo"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/pkg/prettyfmt"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/pkg/puppet"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/pkg/webtee"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/config"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/constant"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/helper"
)

func compatibilityCheck(puppetService *puppet.Service) error {
	// Sanity check
	helper.LoadOSReleaseEnv()
	helper.RequireRootUser()

	// Check required envs and OS
	helper.RequireOSNameEnv()
	helper.RequireOSVersionEnv()
	if _, err := helper.IsSupportedOS(); err != nil {
		slog.Error("OS not supported", slog.String("err", err.Error()))
		return err
	}

	if err := disk.CheckDiskSize(); err != nil {
		prettyfmt.PrettyPrintln(prettyfmt.FontRed("check disk size failed: ", err.Error()))
		return err
	}

	// Check if Puppetserver is alive and active
	if err := puppetService.CheckServerStatus(); err != nil {
		slog.Error("puppet server check failed", slog.Any("error", err))
		return err
	}

	if err := os.Setenv("PATH", constant.PuppetPath); err != nil {
		slog.Error("failed to set the PATH env, exiting")
		return err
	}

	return nil
}

func Install() {
	// Re-initialise the logger with progressbar writer to not disturb the
	// progressbar if we print any logs. Everything is handled by progressbar's
	// Bprintf method under the hood.
	pbWriter := progress.InitProgressBar()
	logger.InitLogger(pbWriter, config.IsDebug())

	certname := helper.GetCertname()
	puppetServer := config.GetPupeptServer()
	obmondoAPIURL := api.GetObmondoURL()
	obmondoAPI := api.NewObmondoClient(obmondoAPIURL, true)
	webtee := webtee.NewWebtee(obmondoAPI)
	puppetService := puppet.NewService(obmondoAPI, webtee)
	provisioner := provisioner.NewService(obmondoAPI, puppetService, webtee)

	webtee.RemoteLogObmondo([]string{"echo Starting Linuxaid Install Setup "}, certname)
	prettyfmt.PrettyPrintf(" %s  %s %s %s %s\n", prettyfmt.IconGear, prettyfmt.FontWhite("Configuring Linuxaid on"), prettyfmt.FontYellow(certname), prettyfmt.FontWhite("with puppetserver"), prettyfmt.FontYellow(puppetServer))
	prettyfmt.PrettyPrintf(" %s  Running this tool will install and configure %s in your system.\n %s Please confirm to continue (Yes/No)? ", prettyfmt.IconGear, prettyfmt.FontYellow("Openvox agent"), prettyfmt.IconQuestion)

	// Accept user input for confirmation
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input != "y" && input != "yes" {
		prettyfmt.PrettyPrintf("\n Exiting the setup...\n")
		return
	}

	// Dummy new line for better clarity of things
	prettyfmt.PrettyPrintln("")

	if err := progress.NonDeterministicFunc("Verifying Token", func() error {
		input := &api.InstallScriptInput{
			Certname: certname,
			Token:    os.Getenv(constant.InstallTokenEnv),
		}

		return obmondoAPI.VerifyInstallToken(input)
	}); err != nil {
		os.Exit(1)
	}

	if err := progress.NonDeterministicFunc("Checking Compatibility", func() error {
		return compatibilityCheck(puppetService)
	}); err != nil {
		os.Exit(1)
	}

	// check if agent disable file exists
	if _, err := os.Stat(constant.AgentDisabledLockFile); err == nil {
		prettyfmt.PrettyPrintln(prettyfmt.FontRed("Openvox has been disabled from the existing setup, can't proceed\npuppet agent --enable will enable the puppet agent\n"))
		webtee.RemoteLogObmondo([]string{"echo Exiting, openvox-agent is already installed and set to disabled"}, helper.GetCertname())
		os.Exit(0)
	}

	// nolint: errcheck
	progress.NonDeterministicFunc("Installing Openvox", func() error {
		provisioner.ProvisionPuppet()
		return nil
	})

	// nolint: errcheck
	progress.NonDeterministicFunc("Configuring Openvox", func() error {
		puppetService.DisableAgentService()
		puppetService.ConfigureAgent()
		puppetService.FacterNewSetup()
		return nil
	})

	// nolint: errcheck
	progress.NonDeterministicFunc("Running Openvox", func() error {
		puppetService.WaitForAgent(constant.PuppetWaitForCertTimeOut)
		puppetService.RunAgent(true, "noop")
		// nolint:errcheck
		obmondoAPI.UpdatePuppetLastRunReport()
		return nil
	})

	webtee.RemoteLogObmondo([]string{"echo Finished Obmondo Setup "}, certname)
	prettyfmt.PrettyPrintln("\n ", prettyfmt.IconSuccess, prettyfmt.FontGreen("Success!"))
	prettyfmt.PrettyPrintf("\n %s %s %s\n", prettyfmt.FontWhite("Head to"), prettyfmt.FontBlue("https://obmondo.com/user/servers"), prettyfmt.FontWhite("to add role and subscription."))
}
