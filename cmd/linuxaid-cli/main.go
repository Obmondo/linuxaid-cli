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
	debugFlag       bool
	rebootFlag      bool
	certnameFlag    string
	skipOpenvoxFlag bool
)

var rootCmd = &cobra.Command{
	Use:   "linuxaid-cli",
	Short: "A brief description of linuxaid-cli",
	Long:  "A longer description of linuxaid-cli application",
	Example: `
	$ linuxaid-cli run-openvox --certname web01.customerid
	$ linuxaid-cli system-update --certname web01.customerid --reboot
	`,
	Version: Version,
	PersistentPreRun: func(cmd *cobra.Command, _ []string) {
		logger.InitLogger(config.IsDebug())

		// Print version first
		slog.Info("linuxaid-cli", slog.String("version", cmd.Root().Version))

		// Get certname from viper (cert, flag, or env)
		if helper.GetCertname() == "" {
			slog.Error("failed to fetch the certname")
			cmd.Help()
			os.Exit(1)
		}

	},
}

func init() {
	v := config.GetViperInstance()

	rootCmd.PersistentFlags().BoolVar(&debugFlag, constant.CobraFlagDebug, false, "Enable debug logs")
	rootCmd.PersistentFlags().StringVar(&certnameFlag, constant.CobraFlagCertname, "", "Certificate name (required)")

	// Bind flags to viper
	v.BindPFlag(constant.CobraFlagDebug, rootCmd.PersistentFlags().Lookup(constant.CobraFlagDebug))
	v.BindPFlag(constant.CobraFlagCertname, rootCmd.PersistentFlags().Lookup(constant.CobraFlagCertname))

	// Bind environment variables
	v.BindEnv(constant.CobraFlagDebug)
	v.BindEnv(constant.CobraFlagCertname)

}

func main() {
	//nolint:reassign
	// By default, parent's PersistentPreRun gets overridden by a child's PersistentPreRun.
	// We want to disable this overriding behaviour and chain all the PersistentPreRuns.
	// REFERENCE : https://github.com/spf13/cobra/pull/2044.
	cobra.EnableTraverseRunHooks = true

	if err := rootCmd.Execute(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
