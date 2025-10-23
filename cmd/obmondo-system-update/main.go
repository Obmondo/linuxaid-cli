package main

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"gitea.obmondo.com/EnableIT/go-scripts/config"
	"gitea.obmondo.com/EnableIT/go-scripts/constant"
	"gitea.obmondo.com/EnableIT/go-scripts/helper"
	"gitea.obmondo.com/EnableIT/go-scripts/helper/logger"
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

		// Get certname from viper (cert, flag, or env)
		if helper.GetCertname() == "" {
			slog.Error("failed to fetch the certname")
			os.Exit(1)
		}
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
	rootCmd.Flags().StringVar(&certNameFlag, constant.CobraFlagCertname, "", "Certificate name (required)")
	rootCmd.Flags().BoolVar(&rebootFlag, constant.CobraFlagReboot, true, "Set this flag false to prevent reboot")
	logger.InitLogger(debugFlag)

	// Bind flags to viper
	viperConfig.BindPFlag(constant.CobraFlagDebug, rootCmd.Flags().Lookup(constant.CobraFlagDebug))
	viperConfig.BindPFlag(constant.CobraFlagReboot, rootCmd.Flags().Lookup(constant.CobraFlagReboot))

	// Bind environment variables
	viperConfig.BindEnv(constant.CobraFlagCertname, "CERTNAME")
}

func main() {

	if err := rootCmd.Execute(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
