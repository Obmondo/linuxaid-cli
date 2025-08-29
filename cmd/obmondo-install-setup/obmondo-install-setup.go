package main

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

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

var (
	versionFlag      bool
	debugFlag        bool
	certNameFlag     string
	puppetServerFlag string
)

var rootCmd = &cobra.Command{
	Use:     "obmondo-install-setup",
	Short:   "An Obmondo linuxaid install script to setup on a linux node",
	Example: `  # obmondo-install-setup --certname web01.customerid`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// Handle version flag first
		if versionFlag {
			slog.Info("obmondo-install-setup", "version", Version)
			os.Exit(0)
		}

		// Get certname from viper (flag or env)
		certName := viper.GetString("certname")
		if certName == "" {
			slog.Error("certname is required. Provide via --certname flag or CERTNAME environment variable")
			os.Exit(1)
		}

		logger.InitLogger(debugFlag)
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {

	defaultServer := constants.DefaultPuppetServerCustomerID + "." + constants.DefaultPuppetServerDomain

	rootCmd.Flags().BoolVarP(&versionFlag, "version", "v", false, "Print version and exit")
	rootCmd.Flags().BoolVar(&debugFlag, "debug", false, "Enable debug logs")
	rootCmd.Flags().StringVar(&certNameFlag, constants.CobraFlagCertName, "", "Certificate name (required)")
	rootCmd.Flags().StringVar(&puppetServerFlag, constants.CobraFlagPuppetServer, defaultServer, "Puppet server hostname")

	// Bind flags to viper
	viper.BindPFlag(constants.CobraFlagDebug, rootCmd.Flags().Lookup(constants.CobraFlagDebug))
	viper.BindPFlag(constants.CobraFlagCertName, rootCmd.Flags().Lookup(constants.CobraFlagCertName))
	viper.BindPFlag(constants.CobraFlagPuppetServer, rootCmd.Flags().Lookup(constants.CobraFlagPuppetServer))

	// Bind environment variables
	viper.BindEnv(constants.CobraFlagCertName, "CERTNAME")
	viper.BindEnv(constants.CobraFlagPuppetServer, "PUPPET_SERVER")

	// Set default values
	viper.SetDefault(constants.CobraFlagPuppetServer, defaultServer)
}

func main() {

	if err := rootCmd.Execute(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	// Sanity check
	utils.LoadOSReleaseEnv()
	utils.RequireRootUser()

	// Check required envs and OS
	utils.RequireOSNameEnv()
	utils.RequireOSVersionEnv()
	if _, err := utils.IsSupportedOS(); err != nil {
		slog.Error("OS not supported", slog.String("err", err.Error()))
		os.Exit(1)
	}

	if err := disk.CheckDiskSize(); err != nil {
		prettyfmt.PrettyFmt(prettyfmt.FontRed("check disk size failed: ", err.Error()))
	}

	certName := viper.GetString("certName")
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
