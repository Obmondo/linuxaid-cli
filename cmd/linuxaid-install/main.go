package main

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"gitea.obmondo.com/EnableIT/go-scripts/config"
	"gitea.obmondo.com/EnableIT/go-scripts/constant"
	"gitea.obmondo.com/EnableIT/go-scripts/helper"
	"gitea.obmondo.com/EnableIT/go-scripts/helper/logger"
	"gitea.obmondo.com/EnableIT/go-scripts/pkg/prettyfmt"
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
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		// Handle version flag first
		if versionFlag {
			slog.Info("obmondo-install-setup", "version", Version)
			os.Exit(0)
		}

		logger.InitLogger(config.IsDebug())

		// Get certname from viper (cert, flag, or env)
		certName := helper.GetCertname()
		if certName == "" {
			errMsg := "Uh ho. I couldn't figure out the certname, please provide one as an ENV"
			prettyfmt.PrettyFmt("\n  ", prettyfmt.IconCheckFail, " ", prettyfmt.FontWhite(errMsg))

			slog.Debug("certname is required. Provide via --certname flag or CERTNAME environment variable")
			cmd.Help()
			os.Exit(1)
		}

		return nil
	},

	Run: func(*cobra.Command, []string) {
		obmondoInstallSetup()
	},
}

func init() {

	viperConfig := config.Initialize()

	defaultServer := constant.DefaultPuppetServerCustomerID + constant.DefaultPuppetServerDomainSuffix

	rootCmd.Flags().BoolVarP(&versionFlag, "version", "v", false, "Print version and exit")
	rootCmd.Flags().BoolVar(&debugFlag, "debug", false, "Enable debug logs")
	rootCmd.Flags().StringVar(&certNameFlag, constant.CobraFlagCertname, "", "Certificate name (required)")
	rootCmd.Flags().StringVar(&puppetServerFlag, constant.CobraFlagPuppetServer, defaultServer, "Puppet server hostname")

	// Bind flags to viper
	viperConfig.BindPFlag(constant.CobraFlagDebug, rootCmd.Flags().Lookup(constant.CobraFlagDebug))
	viperConfig.BindPFlag(constant.CobraFlagCertname, rootCmd.Flags().Lookup(constant.CobraFlagCertname))
	viperConfig.BindPFlag(constant.CobraFlagPuppetServer, rootCmd.Flags().Lookup(constant.CobraFlagPuppetServer))

	// Bind environment variables
	viperConfig.BindEnv(constant.CobraFlagCertname, "CERTNAME")
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
