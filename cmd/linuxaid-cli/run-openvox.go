package main

import (
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/app"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/constant"

	"github.com/spf13/cobra"
)

var openvoxEnvFlag string

var runOpenvoxCmd = &cobra.Command{
	Use:     "run-openvox",
	Short:   "Execute run-openvox command",
	Long:    "A longer description of run-openvox command",
	Example: `$ linuxaid-cli run-openvox --certname web01.example --environment testing`,
	Run: func(*cobra.Command, []string) {
		app.RunOpenvox(openvoxEnvFlag)
	},
}

func init() {
	rootCmd.AddCommand(runOpenvoxCmd)

	// no default here: an unset flag means "ask the API", which is what a normal run does
	runOpenvoxCmd.Flags().StringVarP(&openvoxEnvFlag, constant.CobraFlagEnvironment, constant.CobraFlagEnvironmentShorthand, "", "Puppet environment for this run only (defaults to the environment set in Obmondo)")
}
