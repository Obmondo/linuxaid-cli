package main

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"gitea.obmondo.com/go-scripts/config"
	"gitea.obmondo.com/go-scripts/constants"
	"gitea.obmondo.com/go-scripts/helper/logger"
)

var Version string

var (
	versionFlag  bool
	debugFlag    bool
	rebootFlag   bool
	certNameFlag string
)

var rootCmd = &cobra.Command{
	Use:     "obmondo-system-update",
	Example: `  # obmondo-system-update --certname web01.customerid`,
	PreRunE: func(*cobra.Command, []string) error {
		// Handle version flag first
		if versionFlag {
			slog.Info("obmondo-system-update", "version", Version)
			os.Exit(0)
		}

		logger.InitLogger(config.GetDebug())
		return nil
	},

	Run: func(*cobra.Command, []string) {
		obmondoSystemUpdate()
	},
}

func init() {

	viperConfig := config.Initialize()

	rootCmd.Flags().BoolVarP(&versionFlag, "version", "v", false, "Print version and exit")
	rootCmd.Flags().BoolVar(&debugFlag, constants.CobraFlagDebug, false, "Enable debug logs")
	rootCmd.Flags().StringVar(&certNameFlag, constants.CobraFlagCertName, "", "Certificate name (required)")
	rootCmd.Flags().BoolVar(&rebootFlag, constants.CobraFlagReboot, true, "Set this flag false to prevent reboot")

	// Bind flags to viper
	viperConfig.BindPFlag(constants.CobraFlagDebug, rootCmd.Flags().Lookup(constants.CobraFlagDebug))
	viperConfig.BindPFlag(constants.CobraFlagReboot, rootCmd.Flags().Lookup(constants.CobraFlagReboot))

	// Bind environment variables
	viperConfig.BindEnv(constants.CobraFlagCertName, "CERTNAME")
}
