package app

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	api "gitea.obmondo.com/EnableIT/linuxaid-cli/internal/obmondo"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/shell"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/checkconnectivity"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/config"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/constant"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/puppet"
)

// OpenvoxRun carries the per-run flags of the run-openvox command.
type OpenvoxRun struct {
	// Environment is the puppet environment for this run only: empty asks the API in
	// agent mode, and uses the default in apply mode.
	Environment string
	// Tags restricts the run to a subset of the catalog.
	Tags string
	// Enforce applies changes (--no-noop) instead of the default report-only (--noop).
	Enforce bool
	// Apply compiles the catalog locally from a cloned control-repo instead of asking
	// a puppetserver.
	Apply bool
	// EnvironmentPath holds the cloned puppet environments (apply mode).
	EnvironmentPath string
	// HieraConfig is the global-layer hiera.yaml (apply mode); empty uses the
	// environment's own hierarchy only.
	HieraConfig string
}

// runPuppet executes a puppet command and maps its exit status. Shared by agent and apply mode.
func runPuppet(runner shell.Runner, puppetCmd string) error {
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

	statusCodeFailed := 1
	statusCodeSucceededWithChanges := 2

	slog.Info("executing the puppet command", slog.String("command", puppetCmd))
	result := runner.Run(puppetCmd)
	if result.Err != nil {
		// When encountering status code 1, consider it as an error, and return.
		if result.ExitCode == statusCodeFailed {
			slog.Error("puppet command execution failed", slog.Any("status", result.Err))
			return result.Err
		}

		// When encountering status codes other than 2, just log it as a warning.
		if result.ExitCode != statusCodeSucceededWithChanges {
			slog.Warn("puppet run succeeded, but with failures", slog.Any("status", result.Err))
		}
	}

	slog.Info("completed the puppet command execution")
	return nil
}

// runOpenvoxAgent runs the puppet agent against the puppetserver: report-only (--noop),
// or applying changes when enforce is set.
func runOpenvoxAgent(runner shell.Runner, cfg config.Config, environment, tags string, enforce bool) error {
	// Report-only by default; enforce applies changes and asks for detailed exit codes.
	agentCmd := "/opt/puppetlabs/bin/puppet agent -t --noop"
	if enforce {
		agentCmd = "/opt/puppetlabs/bin/puppet agent -t --no-noop --detailed-exitcodes"
	}
	// The agent decides the environment: the ENC no longer sends one, and puppet.conf no longer
	// pins one, so an environment missing here would leave the run on puppet's own default.
	if environment != "" {
		agentCmd += " --environment " + environment
	}
	// An explicit --puppet-server/PUPPET_SERVER override must reach the agent
	// too, otherwise it keeps using the server from puppet.conf.
	if server := cfg.OpenvoxServer; server != "" {
		if h := extractHostname(server); h != "" {
			server = h
		}
		agentCmd += " --server " + server
	}
	// --tag restricts the run to a subset of the catalog; without it the agent applies everything.
	if tags != "" {
		agentCmd += " --tags " + tags
	}

	return runPuppet(runner, agentCmd)
}

// applyOpenvox runs masterless puppet apply against the environment cloned under
// environmentPath. The environment's own environment.conf and hiera.yaml drive the
// modulepath and hierarchy, exactly as on the puppetserver; hieraConfig injects the
// Helm-values data as the (higher-precedence) global hiera layer.
func applyOpenvox(runner shell.Runner, enforce bool, environmentPath, environment, hieraConfig string) error {
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
	return runPuppet(runner, strings.Join(args, " "))
}

// Entry point
// RunOpenvox performs one puppet agent run. As with SystemUpdate, nil means "nothing more to do"
// rather than "everything succeeded": the paths that give up quietly keep their exit 0.
func RunOpenvox(cfg config.Config, opts OpenvoxRun) error {
	if err := puppet.LoadPuppetEnv(); err != nil {
		return err
	}

	obmondoAPI := api.NewObmondoClient(api.GetObmondoURL(), false, cfg.Certname)

	// Opensource nodes are not registered with Obmondo, so the API would
	// reject every call; skip them instead of logging errors on each run.
	opensource := cfg.Opensource
	if opensource {
		slog.Info("opensource mode, skipping Obmondo API calls")
	}

	certname := cfg.Certname

	// Masterless mode: the catalog is compiled locally from the cloned repo, so there is no
	// puppetserver to resolve or reach, and the environment is the cloned directory rather
	// than something the API decides.
	if opts.Apply {
		environment := opts.Environment
		if environment == "" {
			environment = constant.DefaultOpenvoxEnv
		}

		if !opensource {
			// nolint:errcheck
			obmondoAPI.ServerPing()
		}

		if err := applyOpenvox(shell.New(), opts.Enforce, opts.EnvironmentPath, environment, opts.HieraConfig); err != nil {
			slog.Error("unable to run puppet apply", slog.String("error", err.Error()))
		}

		if !opensource {
			// nolint:errcheck
			obmondoAPI.UpdatePuppetLastRunReport()
		}

		return nil
	}

	prometheusHost, puppetServerHost := resolveCustomerURLs(obmondoAPI, cfg)
	slog.Info("resolved customer URLs",
		slog.String("prometheus", prometheusHost),
		slog.String("puppet_server", puppetServerHost))

	// The connectivity check targets Obmondo hosts (api, prometheus), which
	// are only relevant for nodes registered with a token.
	if !opensource {
		allAPIReachable := checkconnectivity.CheckTCPConnection(prometheusHost, puppetServerHost)
		if !allAPIReachable {
			slog.Error("unable to connect to required hosts, aborting",
				slog.String("prometheus", prometheusHost),
				slog.String("puppet_server", puppetServerHost))
			return nil
		}

		// nolint:errcheck
		obmondoAPI.ServerPing()
	}

	environment := resolveOpenvoxEnvironment(obmondoAPI, certname, opts.Environment, opensource)
	slog.Info("resolved puppet environment", slog.String("environment", environment))

	// Need to have case here later in future, when we migrate the endpoints in go-api
	if err := runOpenvoxAgent(shell.New(), cfg, environment, opts.Tags, opts.Enforce); err != nil {
		slog.Error("unable to run the puppet agent", slog.String("error", err.Error()))
	}

	if !opensource {
		// nolint:errcheck
		obmondoAPI.UpdatePuppetLastRunReport()
	}

	return nil
}

// resolveOpenvoxEnvironment decides which puppet environment this run uses:
//
//  1. --environment/-E, for a one-off run against another branch or tag. It is never sent to the
//     API, so it applies to this run only. The flag is read straight from cobra rather than
//     through viper, whose AutomaticEnv would otherwise let a stray ENVIRONMENT variable pin
//     every run.
//  2. the environment the API resolved for this certname: the pinned override, or the latest
//     linuxaid release.
//  3. the default environment, when the flag is unset and the API cannot be reached.
func resolveOpenvoxEnvironment(obmondoAPI api.ObmondoClient, certname, flagEnvironment string, opensource bool) string {
	if flagEnvironment != "" {
		slog.Info("using the environment given on the command line", slog.String("environment", flagEnvironment))
		return flagEnvironment
	}

	if opensource {
		return constant.DefaultOpenvoxEnv
	}

	environment, err := obmondoAPI.GetServerEnvironment(certname)
	if err != nil {
		slog.Warn("could not fetch the environment from Obmondo, falling back to the default",
			slog.Any("error", err), slog.String("environment", constant.DefaultOpenvoxEnv))
		return constant.DefaultOpenvoxEnv
	}

	if environment == "" {
		slog.Warn("Obmondo returned no environment, falling back to the default",
			slog.String("environment", constant.DefaultOpenvoxEnv))
		return constant.DefaultOpenvoxEnv
	}

	return environment
}
