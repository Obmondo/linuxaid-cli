package util

import (
	"log/slog"
	"os"
	"os/user"
)

// Check if the current user is root or not
// fail if user is not root
func CheckUser() {
	user, err := user.Current()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	if user.Username == "root" {
		return
	}
	slog.Error("exiting, script needs to be run as root", slog.String("current_user", user.Username))
	os.Exit(1)
}
