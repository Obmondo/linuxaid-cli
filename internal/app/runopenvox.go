package app

import (
	"log/slog"

	api "gitea.obmondo.com/EnableIT/linuxaid-cli/internal/obmondo"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/shell"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/checkconnectivity"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/config"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/constant"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/puppet"
)

// Run the puppet agent in noop mode for now
func runOpenvoxAgent(runner shell.Runner, cfg config.Config, environment string) error {
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

	agentCmd := "/opt/puppetlabs/bin/puppet agent -t --noop"
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

	slog.Info("executing the puppet agent command")
	result := runner.Run(agentCmd)
	if result.Err != nil {
		// When encountering status code 1, consider it as an error, and return.
		if result.ExitCode == statusCodeFailed {
			slog.Error("puppet agent command execution failed", slog.Any("status", result.Err))
			return result.Err
		}

		// When encountering status codes other than 2, just log it as a warning.
		if result.ExitCode != statusCodeSucceededWithChanges {
			slog.Warn("puppet agent run succeeded, but with failures", slog.Any("status", result.Err))
		}
	}

	slog.Info("completed the puppet agent command execution")
	return nil
}

// Entry point
// RunOpenvox performs one puppet agent run. As with SystemUpdate, nil means "nothing more to do"
// rather than "everything succeeded": the paths that give up quietly keep their exit 0.
func RunOpenvox(cfg config.Config, environmentFlag string) error {
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

	environment := resolveOpenvoxEnvironment(obmondoAPI, certname, environmentFlag, opensource)
	slog.Info("resolved puppet environment", slog.String("environment", environment))

	// Need to have case here later in future, when we migrate the endpoints in go-api
	if err := runOpenvoxAgent(shell.New(), cfg, environment); err != nil {
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
