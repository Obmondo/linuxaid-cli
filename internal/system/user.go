package system

import (
	"fmt"
	"os/user"
)

// RequireRootUser reports whether the current user is root.
func RequireRootUser() error {
	current, err := user.Current()
	if err != nil {
		return fmt.Errorf("could not determine the current user: %w", err)
	}

	if current.Username != "root" {
		return fmt.Errorf("%w, current user is %q", errNotRoot, current.Username)
	}

	return nil
}
