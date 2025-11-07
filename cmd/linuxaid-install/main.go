package main

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/config"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/constant"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/helper"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/helper/logger"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/pkg/prettyfmt"
)

var Version string

var (
	debugFlag        bool
	certNameFlag     string
	puppetServerFlag string
)

var rootCmd = &cobra.Command{
	Use:     "linuxaid-install",
	Short:   "A brief description of linuxaid-cli",
	Long:    "A longer description of linuxaid-cli application",
	Example: `$ linuxaid-install --certname web01.example --puppet-server your.openvoxserver.com`,
	Version: Version,
	CompletionOptions: cobra.CompletionOptions{
		HiddenDefaultCmd: true,
	},
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		logger.InitLogger(config.IsDebug())

		// Print version first
		slog.Info("linuxaid-cli", slog.String("version", cmd.Root().Version))

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
		Install()
	},
}

func init() {
	defaultServer := constant.DefaultPuppetServerCustomerID + constant.DefaultPuppetServerDomainSuffix

	rootCmd.Flags().BoolVar(&debugFlag, "debug", false, "Enable debug logs")
	rootCmd.Flags().StringVar(&certNameFlag, constant.CobraFlagCertname, "", "Certificate name (required)")
	rootCmd.Flags().StringVar(&puppetServerFlag, constant.CobraFlagPuppetServer, defaultServer, "Puppet server hostname")

	// Bind flags to viper
	v := config.GetViperInstance()
	v.BindPFlag(constant.CobraFlagDebug, rootCmd.Flags().Lookup(constant.CobraFlagDebug))
	v.BindPFlag(constant.CobraFlagCertname, rootCmd.Flags().Lookup(constant.CobraFlagCertname))
	v.BindPFlag(constant.CobraFlagPuppetServer, rootCmd.Flags().Lookup(constant.CobraFlagPuppetServer))

	// Bind environment variables
	v.BindEnv(constant.CobraFlagDebug)
	v.BindEnv(constant.CobraFlagCertname)
	v.BindEnv(constant.CobraFlagPuppetServer, "PUPPET_SERVER")

	// Set default values
	v.SetDefault(constant.CobraFlagPuppetServer, defaultServer)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
