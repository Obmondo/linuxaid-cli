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
	rootCmd.Flags().BoolVar(&debugFlag, constant.CobraFlagDebug, false, "Enable debug logs")
	rootCmd.Flags().StringVar(&certNameFlag, constant.CobraFlagCertName, "", "Certificate name (required)")
	rootCmd.Flags().BoolVar(&rebootFlag, constant.CobraFlagReboot, true, "Set this flag false to prevent reboot")

	// Bind flags to viper
	viperConfig.BindPFlag(constant.CobraFlagDebug, rootCmd.Flags().Lookup(constant.CobraFlagDebug))
	viperConfig.BindPFlag(constant.CobraFlagReboot, rootCmd.Flags().Lookup(constant.CobraFlagReboot))

	// Bind environment variables
	viperConfig.BindEnv(constant.CobraFlagCertName, "CERTNAME")
}
