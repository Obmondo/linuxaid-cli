package helper

import (
	"os"
	"slices"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/constant"
)

// IsPuppetSuccessExitCode reports whether a puppet agent exit code counts as
// success on this node.
//
// We're patching the error handling for turrisos for now, since we're still
// updating linuxaid support: it additionally tolerates exit code 4. Once
// done, we'll remove this special handling.
func IsPuppetSuccessExitCode(exitCode int) bool {
	if os.Getenv("ID") == ConstDistributionNameTurrisOS && exitCode == 4 { // nolint: mnd
		return true
	}
	return slices.Contains(constant.PuppetSuccessExitCodes, exitCode)
}
