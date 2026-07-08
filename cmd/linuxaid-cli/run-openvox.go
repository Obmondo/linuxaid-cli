package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/constant"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/helper"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/pkg/checkconnectivity"
	api "gitea.obmondo.com/EnableIT/linuxaid-cli/pkg/obmondo"
	"github.com/bitfield/script"
	"github.com/spf13/cobra"
)

var (
	// runOpenvoxEnforce applies changes (puppet --no-noop) instead of the default report-only (--noop).
	runOpenvoxEnforce bool
	// runOpenvoxApply compiles the catalog locally (puppet apply) from a cloned
	// control-repo instead of asking a puppetserver (puppet agent).
	runOpenvoxApply     bool
	runOpenvoxEnvPath   string
	runOpenvoxEnvName   string
	runOpenvoxHieraConf string
)

var runOpenvoxCmd = &cobra.Command{
	Use:     "run-openvox",
	Short:   "Execute run-openvox command",
	Long:    "A longer description of run-openvox command",
	Example: `$ linuxaid-cli run-openvox --certname web01.example`,
	Run: func(*cobra.Command, []string) {
		RunOpenvox(runOpenvoxEnforce)
	},
}

// runPuppet executes a puppet command and maps its exit status.
//
// Puppet run execution returns total 5 status codes
//
// 0: The run succeeded with no changes or failures; the system was already in the desired state.
// 1: The run failed, or wasn't attempted due to another run already in progress.
// 2: The run succeeded, and some resources were changed.
// 4: The run succeeded, and some resources failed.
// 6: The run succeeded, and included both changes and failures.
// [Source: https://www.puppet.com/docs/puppet/7/man/agent.html#usage-notes]
//
// We throw error at status code 1, and return.
// Status codes other than 2 are considered as warning.
// Status code 0 doesn't count as error, so no need to handle it.
func runPuppet(puppetCmd string) error {
	statusCodeFailed := 1
	statusCodeSucceededWithChanges := 2

	slog.Info("executing the puppet command", slog.String("command", puppetCmd))
	cmdPipe := script.Exec(puppetCmd)
	_, err := cmdPipe.Stdout()
	if err != nil {
		// When encountering status code 1, consider it as an error, and return.
		if cmdPipe.ExitStatus() == statusCodeFailed {
			slog.Error("puppet command execution failed", slog.String("status", err.Error()))
			return err
		}

		// When encountering status codes other than 2, just log it as a warning.
		if cmdPipe.ExitStatus() != statusCodeSucceededWithChanges {
			slog.Warn("puppet run succeeded, but with failures", slog.String("status", err.Error()))
		}
	}

	slog.Info("completed the puppet command execution")
	return nil
}

// runOpenvoxAgent runs the puppet agent against the puppetserver: report-only (--noop)
// by default, or applying changes (--no-noop --detailed-exitcodes) when enforce is set.
func runOpenvoxAgent(enforce bool) error {
	puppetCmd := "/opt/puppetlabs/bin/puppet agent -t --noop"
	if enforce {
		puppetCmd = "/opt/puppetlabs/bin/puppet agent -t --no-noop --detailed-exitcodes"
	}

	slog.Info("executing the puppet agent command", slog.Bool("enforce", enforce))
	return runPuppet(puppetCmd)
}

// applyOpenvox runs masterless puppet apply against the environment cloned under
// environmentPath. The environment's own environment.conf and hiera.yaml drive the
// modulepath and hierarchy, exactly as on the puppetserver; hieraConfig injects the
// Helm-values data as the (higher-precedence) global hiera layer.
func applyOpenvox(enforce bool, environmentPath, environment, hieraConfig string) error {
	sitePP := filepath.Join(environmentPath, environment, "manifests", "site.pp")
	if _, err := os.Stat(sitePP); err != nil {
		return fmt.Errorf("site.pp not found at %s, is the control-repo cloned: %w", sitePP, err)
	}

	args := []string{
		"/opt/puppetlabs/bin/puppet", "apply",
		"--detailed-exitcodes",
		"--environmentpath", environmentPath,
		"--environment", environment,
	}
	if hieraConfig != "" {
		args = append(args, "--hiera_config", hieraConfig)
	}
	if !enforce {
		args = append(args, "--noop")
	}
	args = append(args, sitePP)

	slog.Info("executing the puppet apply command", slog.Bool("enforce", enforce), slog.String("environment", environment))
	return runPuppet(strings.Join(args, " "))
}

// Entry point
func RunOpenvox(enforce bool) {
	helper.LoadPuppetEnv()

	obmondoAPI := api.NewObmondoClient(api.GetObmondoURL(), false)

	certname := helper.GetCertname()

	// Masterless mode: the catalog is compiled locally from the cloned repo, so
	// there is no puppetserver to resolve or reach.
	if runOpenvoxApply {
		// nolint:errcheck
		obmondoAPI.ServerPing()

		if err := applyOpenvox(enforce, runOpenvoxEnvPath, runOpenvoxEnvName, runOpenvoxHieraConf); err != nil {
			slog.Error("unable to run puppet apply", slog.String("error", err.Error()))
		}

		// nolint:errcheck
		obmondoAPI.UpdatePuppetLastRunReport()
		return
	}

	prometheusHost, puppetServerHost := resolveCustomerURLs(obmondoAPI, certname)
	slog.Info("resolved customer URLs",
		slog.String("prometheus", prometheusHost),
		slog.String("puppet_server", puppetServerHost))

	allAPIReachable := checkconnectivity.CheckTCPConnection(prometheusHost, puppetServerHost)
	if !allAPIReachable {
		slog.Error("unable to connect to required hosts, aborting",
			slog.String("prometheus", prometheusHost),
			slog.String("puppet_server", puppetServerHost))
		return
	}

	// nolint:errcheck
	obmondoAPI.ServerPing()

	// Need to have case here later in future, when we migrate the endpoints in go-api
	if err := runOpenvoxAgent(enforce); err != nil {
		slog.Error("unable to run the puppet agent", slog.String("error", err.Error()))
	}

	// nolint:errcheck
	obmondoAPI.UpdatePuppetLastRunReport()
}

func init() {
	f := runOpenvoxCmd.Flags()
	f.BoolVar(&runOpenvoxEnforce, constant.CobraFlagEnforce, false,
		"Apply changes by running puppet with --no-noop (default is report-only --noop)")
	f.BoolVar(&runOpenvoxApply, constant.CobraFlagApply, false,
		"Run masterless puppet apply against a locally cloned control-repo instead of puppet agent")
	f.StringVar(&runOpenvoxEnvPath, constant.CobraFlagEnvironmentPath, constant.DefaultEnvironmentPath,
		"Path holding the cloned puppet environments (apply mode)")
	f.StringVar(&runOpenvoxEnvName, constant.CobraFlagOpenvoxEnv, constant.DefaultOpenvoxEnv,
		"LinuxAid/OpenVox environment to apply (apply mode)")
	f.StringVar(&runOpenvoxHieraConf, constant.CobraFlagHieraConfig, constant.DefaultHieraConfig,
		"Global-layer hiera.yaml for puppet apply, empty to use the environment hierarchy only (apply mode)")
	rootCmd.AddCommand(runOpenvoxCmd)
}
