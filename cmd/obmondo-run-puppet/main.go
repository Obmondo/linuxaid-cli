package main

import (
	"flag"
	"log/slog"
	"os"

	"gitea.obmondo.com/EnableIT/go-scripts/helper"
	"gitea.obmondo.com/EnableIT/go-scripts/pkg/checkconnectivity"
	api "gitea.obmondo.com/EnableIT/go-scripts/pkg/obmondo"
	"github.com/bitfield/script"
)

var Version string

// Run the puppet agent in noop mode for now
func runPuppet() error {
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

	slog.Info("executing the puppet agent command")
	cmdPipe := script.Exec("/opt/puppetlabs/bin/puppet agent -t --noop")
	_, err := cmdPipe.Stdout()
	if err != nil {
		// When encountering status code 1, consider it as an error, and return.
		if cmdPipe.ExitStatus() == statusCodeFailed {
			slog.Error("puppet agent command execution failed", slog.String("status", err.Error()))
			return err
		}

		// When encountering status codes other than 2, just log it as a warning.
		if cmdPipe.ExitStatus() != statusCodeSucceededWithChanges {
			slog.Warn("puppet agent run succeeded, but with failures", slog.String("status", err.Error()))
		}
	}

	slog.Info("completed the puppet agent command execution")
	return nil
}

// Entry point
func main() {
	versionFlag := flag.Bool("version", false, "Print version and exit")

	flag.Parse()

	if *versionFlag {
		slog.Info("run_puppet", "version", Version)
		os.Exit(0)
	}

	helper.LoadPuppetEnv()

	slog.Info("run_puppet", "version", Version)

	allAPIReachable := checkconnectivity.CheckTCPConnection()
	if !allAPIReachable {
		slog.Error("unable to connect to obmondo api, aborting", slog.String("error", "api not accessible"))
		return
	}

	obmondoAPI := api.NewObmondoClient(false)

	// nolint:errcheck
	obmondoAPI.ServerPing()

	// Need to have case here later in future, when we migrate the endpoints in go-api
	if err := runPuppet(); err != nil {
		slog.Error("unable to run the puppet agent", slog.String("error", err.Error()))
	}

	// nolint:errcheck
	obmondoAPI.UpdatePuppetLastRunReport()
}

