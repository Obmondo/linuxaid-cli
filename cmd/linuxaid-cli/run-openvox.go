package main

import (
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/app"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/constant"

	"github.com/spf13/cobra"
)

var (
	openvoxEnvFlag string
	openvoxTagFlag string
)

var runOpenvoxCmd = &cobra.Command{
	Use:           "run-openvox",
	Short:         "Execute run-openvox command",
	Long:          "A longer description of run-openvox command",
	Example:       `$ linuxaid-cli run-openvox --certname web01.example --environment testing --tag nginx`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(*cobra.Command, []string) error {
		return app.RunOpenvox(cfg, openvoxEnvFlag, openvoxTagFlag)
	},
}

func init() {
	rootCmd.AddCommand(runOpenvoxCmd)

	// no default here: an unset flag means "ask the API", which is what a normal run does
	runOpenvoxCmd.Flags().StringVarP(&openvoxEnvFlag, constant.CobraFlagEnvironment, constant.CobraFlagEnvironmentShorthand, "", "Puppet environment for this run only (defaults to the environment set in Obmondo)")
	// an unset flag runs the full catalog; a value restricts the run to those openvox tags
	runOpenvoxCmd.Flags().StringVar(&openvoxTagFlag, constant.CobraFlagTag, "", "Restrict this run to the given openvox tags (comma-separated), like puppet agent --tags")
}
