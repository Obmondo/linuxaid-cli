package main

import (
	"log/slog"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/app"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/config"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/constant"

	"github.com/spf13/cobra"
)

const defaultSecurityExporterURL = "http://127.254.254.254:63396"

var securityExporterURLFlag string

var systemUpdateCmd = &cobra.Command{
	Use:     "system-update",
	Short:   "Execute system-update command",
	Long:    "A longer description of system-update command",
	Example: `$ linuxaid-cli system-update --certname web01.example --no-reboot`,
	PreRun: func(*cobra.Command, []string) {
		if config.ShouldSkipOpenvox() {
			slog.Info("Openvox-agent run will be skipped")
		}
	},
	SilenceUsage: true,
	RunE: func(*cobra.Command, []string) error {
		return app.SystemUpdate()
	},
}

func init() {
	rootCmd.AddCommand(systemUpdateCmd)

	systemUpdateCmd.Flags().BoolVar(&rebootFlag, constant.CobraFlagNoReboot, false, "Set this flag to prevent reboot (default will reboot)")
	systemUpdateCmd.Flags().BoolVar(&skipOpenvoxFlag, constant.CobraFlagSkipOpenvox, false, "Set this flag to prevent running openvox")
	systemUpdateCmd.Flags().StringVar(&securityExporterURLFlag, constant.CobraFlagSecurityExporterURL, defaultSecurityExporterURL, "Security exporter URL")

	// Bind flags to viper
	v := config.GetViperInstance()
	v.BindPFlag(constant.CobraFlagNoReboot, systemUpdateCmd.Flags().Lookup(constant.CobraFlagNoReboot))
	v.BindPFlag(constant.CobraFlagSkipOpenvox, systemUpdateCmd.Flags().Lookup(constant.CobraFlagSkipOpenvox))
	v.BindPFlag(constant.CobraFlagSecurityExporterURL, systemUpdateCmd.Flags().Lookup(constant.CobraFlagSecurityExporterURL))

	// Bind environment variables
	v.BindEnv(constant.CobraFlagNoReboot, "NO_REBOOT")
	v.BindEnv(constant.CobraFlagSkipOpenvox, "SKIP_OPENVOX")
	v.BindEnv(constant.CobraFlagSecurityExporterURL, "SECURITY_EXPORTER_URL")
}
