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
	versionFlag     bool
	debugFlag       bool
	rebootFlag      bool
	certnameFlag    string
	skipOpenvoxFlag bool
)

var rootCmd = &cobra.Command{
	Use:     "linuxaid-cli",
	Short:   "A brief description of my-cli",
	Long:    "A longer description of my-cli application",
	Example: `  # linuxaid-cli --certname web01.customerid`,
	PreRunE: func(*cobra.Command, []string) error {
		logger.InitLogger(config.IsDebug())

		// Handle version flag first
		if versionFlag {
			slog.Info("system-update", "version", Version)
			os.Exit(0)
		}

		// Get certname from viper (cert, flag, or env)
		if helper.GetCertname() == "" {
			slog.Error("failed to fetch the certname")
			os.Exit(1)
		}

		return nil
	},

	// Run: func(*cobra.Command, []string) {
	// 	obmondoSystemUpdate()
	// },
}

func init() {
	viperConfig := config.Initialize()

	rootCmd.PersistentFlags().BoolVarP(&versionFlag, "version", "v", false, "Print version and exit")
	rootCmd.PersistentFlags().BoolVar(&debugFlag, constant.CobraFlagDebug, false, "Enable debug logs")
	rootCmd.PersistentFlags().StringVar(&certnameFlag, constant.CobraFlagCertname, "", "Certificate name (required)")
	rootCmd.PersistentFlags().BoolVar(&rebootFlag, constant.CobraFlagReboot, true, "Set this flag false to prevent reboot")
	rootCmd.PersistentFlags().BoolVar(&skipOpenvoxFlag, constant.CobraFlagSkipOpenvox, false, "Set this flag to prevent running openvox")

	// Bind flags to viper
	viperConfig.BindPFlag(constant.CobraFlagDebug, rootCmd.PersistentFlags().Lookup(constant.CobraFlagDebug))
	viperConfig.BindPFlag(constant.CobraFlagReboot, rootCmd.PersistentFlags().Lookup(constant.CobraFlagReboot))
	viperConfig.BindPFlag(constant.CobraFlagSkipOpenvox, rootCmd.PersistentFlags().Lookup(constant.CobraFlagSkipOpenvox))

	// Bind environment variables
	viperConfig.BindEnv(constant.CobraFlagCertname, "CERTNAME")
	viperConfig.BindEnv(constant.CobraFlagSkipOpenvox, "SKIP_OPENVOX")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
