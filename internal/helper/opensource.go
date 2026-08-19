package helper

import (
	"os"
	"path/filepath"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/constant"
)

// SetOpensourceMode records whether this node was set up without an Obmondo
// token, so later linuxaid-cli runs know to skip Obmondo API calls.
func SetOpensourceMode(enabled bool) error {
	if !enabled {
		if err := os.Remove(constant.OpensourceModeFile); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}

	// nolint: mnd
	if err := os.MkdirAll(filepath.Dir(constant.OpensourceModeFile), 0o755); err != nil {
		return err
	}

	// nolint: mnd
	return os.WriteFile(constant.OpensourceModeFile, []byte("This node was set up without an Obmondo token.\n"), 0o644)
}

// IsOpensourceMode reports whether the node was set up without an Obmondo
// token. On such nodes the Obmondo API rejects requests, so callers skip them.
func IsOpensourceMode() bool {
	_, err := os.Stat(constant.OpensourceModeFile)
	return err == nil
}
