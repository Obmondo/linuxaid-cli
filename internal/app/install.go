package app

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"strings"

	api "gitea.obmondo.com/EnableIT/linuxaid-cli/internal/obmondo"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/certs"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/config"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/constant"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/disk"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/logger"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/prettyfmt"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/progress"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/provisioner"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/puppet"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/system"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/webtee"
)

func compatibilityCheck(puppetService *puppet.Service) error {
	// Sanity check
	if err := system.LoadOSReleaseEnv(); err != nil {
		return err
	}

	if err := system.RequireRootUser(); err != nil {
		return err
	}

	// Check required envs and OS
	if err := system.RequireOSNameEnv(); err != nil {
		return err
	}

	if err := system.RequireOSVersionEnv(); err != nil {
		return err
	}

	if _, err := system.IsSupportedOS(); err != nil {
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

// func shouldContinueAfterConfirmation determines if the installation process should continue after user confirmation.
// If the user provides no input (white spaces, newline, tab, etc), the same confirmation question is asked again.
//
// Inputs for continuation:
//   - y (case-insensitive)
//   - yes (case-insensitive)
//
// Anything other than this is considered as "no", and the program will exit.
func shouldContinueAfterConfirmation() bool {
	// I'm really not a fan of infinite loops, but just for this time I'll pretend I didn't wrote this.
	for {
		prettyfmt.PrettyPrintf(" %s Please confirm to continue (Yes/No)? ", prettyfmt.IconQuestion)

		// Accept user input for confirmation
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.ToLower(input)
		input = strings.TrimSpace(input)

		if input == "" {
			continue
		}

		if input != "y" && input != "yes" {
			prettyfmt.PrettyPrintf("\n Exiting the setup...\n")
			return false
		}

		// Dummy new line for better clarity of things
		prettyfmt.PrettyPrintln("")
		return true
	}
}

func Install(openvoxEnv string) error {
	// Re-initialise the logger with progressbar writer to not disturb the
	// progressbar if we print any logs. Everything is handled by progressbar's
	// Bprintf method under the hood.
	pbWriter := progress.InitProgressBar()
	logger.InitLogger(pbWriter, config.IsDebug())

	certname := certs.GetCertname()
	openvoxServer := config.GetOpenvoxServer()
	obmondoAPIURL := api.GetObmondoURL()
	obmondoAPI := api.NewObmondoClient(obmondoAPIURL, true)
	webtee := webtee.NewWebtee(obmondoAPI)
	puppetService := puppet.NewService(obmondoAPI, webtee)
	provisioner := provisioner.NewService(obmondoAPI, puppetService, webtee)

	//nolint:errcheck // a banner failing must not abort the install
	_ = webtee.RemoteLogObmondo([]string{"echo Starting Linuxaid Install Setup "}, certname)
	prettyfmt.PrettyPrintf(" %s  %s %s %s %s %s %s\n", prettyfmt.IconGear, prettyfmt.FontWhite("Configuring Linuxaid on"), prettyfmt.FontYellow(certname), prettyfmt.FontWhite("with Openvox Server"), prettyfmt.FontYellow(openvoxServer), prettyfmt.FontWhite("and environment"), prettyfmt.FontYellow(openvoxEnv))
	prettyfmt.PrettyPrintf(" %s  Running this tool will install and configure %s in your system.\n", prettyfmt.IconGear, prettyfmt.FontYellow("Openvox agent"))

	if !shouldContinueAfterConfirmation() {
		return nil
	}

	// Token is only required for Obmondo customers; opensource users can run
	// the setup without one.
	token, hasToken := os.LookupEnv(constant.InstallTokenEnv)
	if hasToken {
		if err := progress.NonDeterministicFunc("Verifying Token", func() error {
			input := &api.InstallScriptInput{
				Certname: certname,
				Token:    token,
			}

			return obmondoAPI.VerifyInstallToken(input)
		}); err != nil {
			return err
		}
	}

	// Remember the mode so linuxaid-cli knows to skip Obmondo API calls on
	// opensource nodes.
	if err := config.SetOpensourceMode(!hasToken); err != nil {
		slog.Warn("failed to record opensource mode", slog.Any("error", err))
	}

	if err := progress.NonDeterministicFunc("Checking Compatibility", func() error {
		return compatibilityCheck(puppetService)
	}); err != nil {
		return err
	}

	// check if agent disable file exists
	if _, err := os.Stat(constant.AgentDisabledLockFile); err == nil {
		prettyfmt.PrettyPrintln(prettyfmt.FontRed("Openvox has been disabled from the existing setup, can't proceed\npuppet agent --enable will enable the puppet agent\n"))
		//nolint:errcheck // a banner failing must not abort the install
		_ = webtee.RemoteLogObmondo([]string{"echo Exiting, openvox-agent is already installed and set to disabled"}, certs.GetCertname())

		// an already-disabled agent is not a failure: keep the exit 0 this always had
		return nil
	}

	if err := progress.NonDeterministicFunc("Installing Openvox", func() error {
		return provisioner.ProvisionPuppet()
	}); err != nil {
		return err
	}

	if err := progress.NonDeterministicFunc("Configuring Openvox", func() error {
		if err := puppetService.DisableAgentService(); err != nil {
			return err
		}

		if err := puppetService.ConfigureAgent(); err != nil {
			return err
		}

		return puppetService.FacterNewSetup()
	}); err != nil {
		return err
	}

	if err := progress.NonDeterministicFunc("Running Openvox", func() error {
		puppetService.WaitForAgent(constant.PuppetWaitForCertTimeOut)

		// the remote-logged run used to abort the whole process from inside webtee; keep it fatal
		if exitCode := puppetService.RunAgent(true, "noop", openvoxEnv); exitCode != 0 {
			return fmt.Errorf("the openvox run failed with exit code %d", exitCode)
		}

		if hasToken {
			// nolint:errcheck
			obmondoAPI.UpdatePuppetLastRunReport()
		}

		return nil
	}); err != nil {
		return err
	}

	//nolint:errcheck // a banner failing must not abort the install
	_ = webtee.RemoteLogObmondo([]string{"echo Finished Obmondo Setup "}, certname)
	prettyfmt.PrettyPrintln("\n ", prettyfmt.IconSuccess, prettyfmt.FontGreen("Success!"))
	prettyfmt.PrettyPrintf("\n %s %s %s\n", prettyfmt.FontWhite("Head to"), prettyfmt.FontBlue("https://obmondo.com/user/servers"), prettyfmt.FontWhite("to add role and subscription."))

	return nil
}
