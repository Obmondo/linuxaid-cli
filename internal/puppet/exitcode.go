package puppet

import (
	"os"
	"slices"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/constant"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/system"
)

// Exit codes of a puppet run with --detailed-exitcodes.
// [Source: https://www.puppet.com/docs/puppet/7/man/agent.html#usage-notes]
const (
	// ExitNoChanges means the run succeeded and nothing needed changing.
	ExitNoChanges = 0
	// ExitFailed means the run failed, or was not attempted because another run was in progress.
	ExitFailed = 1
	// ExitChanged means the run succeeded and changed some resources.
	ExitChanged = 2
	// ExitResourcesFailed means the run succeeded, but some resources failed.
	ExitResourcesFailed = 4
	// ExitChangedAndFailed means the run succeeded, changed some resources, and some failed.
	ExitChangedAndFailed = 6
)

// Succeeded reports whether a puppet run with --detailed-exitcodes counts as successful: one of
// constant.PuppetSuccessExitCodes, or failed resources on TurrisOS, whose linuxaid support is still
// being updated.
func Succeeded(exitCode int) bool {
	if slices.Contains(constant.PuppetSuccessExitCodes, exitCode) {
		return true
	}

	return os.Getenv("ID") == system.ConstDistributionNameTurrisOS && exitCode == ExitResourcesFailed
}
