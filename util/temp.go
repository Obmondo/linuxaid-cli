package util

import (
	"log/slog"
	"os"
)

func TempDir() string {
	dir, err := os.MkdirTemp("", "")
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	return dir
}
