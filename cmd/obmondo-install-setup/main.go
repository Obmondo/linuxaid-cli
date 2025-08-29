package main

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"gitea.obmondo.com/go-scripts/config"
	"gitea.obmondo.com/go-scripts/constant"
	"gitea.obmondo.com/go-scripts/helper/logger"
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
	PreRunE: func(*cobra.Command, []string) error {
		// Handle version flag first
		if versionFlag {
			slog.Info("obmondo-install-setup", "version", Version)
			os.Exit(0)
		}

		logger.InitLogger(config.GetDebug())
		return nil
	},

	Run: func(cmd *cobra.Command, _ []string) {
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

	defaultServer := constant.DefaultPuppetServerCustomerID + "." + constant.DefaultPuppetServerDomain

	rootCmd.Flags().BoolVarP(&versionFlag, "version", "v", false, "Print version and exit")
	rootCmd.Flags().BoolVar(&debugFlag, "debug", false, "Enable debug logs")
	rootCmd.Flags().StringVar(&certNameFlag, constant.CobraFlagCertName, "", "Certificate name (required)")
	rootCmd.Flags().StringVar(&puppetServerFlag, constant.CobraFlagPuppetServer, defaultServer, "Puppet server hostname")

	// Bind flags to viper
	viperConfig.BindPFlag(constant.CobraFlagDebug, rootCmd.Flags().Lookup(constant.CobraFlagDebug))
	viperConfig.BindPFlag(constant.CobraFlagCertName, rootCmd.Flags().Lookup(constant.CobraFlagCertName))
	viperConfig.BindPFlag(constant.CobraFlagPuppetServer, rootCmd.Flags().Lookup(constant.CobraFlagPuppetServer))

	// Bind environment variables
	viperConfig.BindEnv(constant.CobraFlagCertName, "CERTNAME")
	viperConfig.BindEnv(constant.CobraFlagPuppetServer, "PUPPET_SERVER")

	// Set default values
	viperConfig.SetDefault(constant.CobraFlagPuppetServer, defaultServer)
}

func main() {

	if err := rootCmd.Execute(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
