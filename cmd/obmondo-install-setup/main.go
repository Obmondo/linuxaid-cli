package main

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"go-scripts/config"
	"go-scripts/constants"
	"go-scripts/utils/logger"
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
	Example: `  # obmondo-install-setup --certname web01.customerid`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// Handle version flag first
		if versionFlag {
			slog.Info("obmondo-install-setup", "version", Version)
			os.Exit(0)
		}

		logger.InitLogger(config.GetDebug())
		return nil
	},

	Run: func(cmd *cobra.Command, args []string) {
		// Get certname from viper (flag or env)
		certName := config.GetCertName()
		if certName == "" {
			slog.Debug("certname is required. Provide via --certname flag or CERTNAME environment variable")
			cmd.Help()
			os.Exit(1)
		}

		obmondoInstallSetup()
	},
}

func init() {

	viperConfig := config.Initialize()

	defaultServer := constants.DefaultPuppetServerCustomerID + "." + constants.DefaultPuppetServerDomain

	rootCmd.Flags().BoolVarP(&versionFlag, "version", "v", false, "Print version and exit")
	rootCmd.Flags().BoolVar(&debugFlag, "debug", false, "Enable debug logs")
	rootCmd.Flags().StringVar(&certNameFlag, constants.CobraFlagCertName, "", "Certificate name (required)")
	rootCmd.Flags().StringVar(&puppetServerFlag, constants.CobraFlagPuppetServer, defaultServer, "Puppet server hostname")

	// Bind flags to viper
	viperConfig.BindPFlag(constants.CobraFlagDebug, rootCmd.Flags().Lookup(constants.CobraFlagDebug))
	viperConfig.BindPFlag(constants.CobraFlagCertName, rootCmd.Flags().Lookup(constants.CobraFlagCertName))
	viperConfig.BindPFlag(constants.CobraFlagPuppetServer, rootCmd.Flags().Lookup(constants.CobraFlagPuppetServer))

	// Bind environment variables
	viperConfig.BindEnv(constants.CobraFlagCertName, "CERTNAME")
	viperConfig.BindEnv(constants.CobraFlagPuppetServer, "PUPPET_SERVER")

	// Set default values
	viperConfig.SetDefault(constants.CobraFlagPuppetServer, defaultServer)
}

func main() {

	if err := rootCmd.Execute(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
