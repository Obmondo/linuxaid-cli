package main

import (
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/app"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/constant"

	"github.com/spf13/cobra"
)

var (
	openvoxEnvFlag string
	openvoxTagFlag string
	// runOpenvoxEnforce applies changes (puppet --no-noop) instead of the default report-only (--noop).
	runOpenvoxEnforce bool
	// runOpenvoxApply compiles the catalog locally (puppet apply) from a cloned
	// control-repo instead of asking a puppetserver (puppet agent).
	runOpenvoxApply     bool
	runOpenvoxEnvPath   string
	runOpenvoxHieraConf string
)

var runOpenvoxCmd = &cobra.Command{
	Use:           "run-openvox",
	Short:         "Execute run-openvox command",
	Long:          "A longer description of run-openvox command",
	Example:       `$ linuxaid-cli run-openvox --certname web01.example --environment testing --tag nginx`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(*cobra.Command, []string) error {
		return app.RunOpenvox(cfg, app.OpenvoxRun{
			Environment:     openvoxEnvFlag,
			Tags:            openvoxTagFlag,
			Enforce:         runOpenvoxEnforce,
			Apply:           runOpenvoxApply,
			EnvironmentPath: runOpenvoxEnvPath,
			HieraConfig:     runOpenvoxHieraConf,
		})
	},
}

func init() {
	f := runOpenvoxCmd.Flags()
	f.BoolVar(&runOpenvoxEnforce, constant.CobraFlagEnforce, false,
		"Apply changes by running puppet with --no-noop (default is report-only --noop)")
	f.BoolVar(&runOpenvoxApply, constant.CobraFlagApply, false,
		"Run masterless puppet apply against a locally cloned control-repo instead of puppet agent")
	f.StringVar(&runOpenvoxEnvPath, constant.CobraFlagEnvironmentPath, constant.DefaultEnvironmentPath,
		"Path holding the cloned puppet environments (apply mode)")
	f.StringVar(&runOpenvoxHieraConf, constant.CobraFlagHieraConfig, constant.DefaultHieraConfig,
		"Global-layer hiera.yaml for puppet apply, empty to use the environment hierarchy only (apply mode)")

	// no default here: an unset flag means "ask the API", which is what a normal run does
	f.StringVarP(&openvoxEnvFlag, constant.CobraFlagEnvironment, constant.CobraFlagEnvironmentShorthand, "", "Puppet environment for this run only (defaults to the environment set in Obmondo, or to the default in apply mode)")
	// an unset flag runs the full catalog; a value restricts the run to those openvox tags
	f.StringVar(&openvoxTagFlag, constant.CobraFlagTag, "", "Restrict this run to the given openvox tags (comma-separated), like puppet agent --tags")

	rootCmd.AddCommand(runOpenvoxCmd)
}
