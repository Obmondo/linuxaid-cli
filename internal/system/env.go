package system

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

func RequireOSNameEnv() {
	_, codeName := os.LookupEnv("NAME")
	if !codeName {
		slog.Error("NAME env variable not set")
		os.Exit(1)
	}
}

func RequireOSVersionEnv() {
	_, codeName := os.LookupEnv("VERSION")
	if !codeName {
		slog.Error("VERSION env variable not set")
		os.Exit(1)
	}
}

func RequireUbuntuCodeNameEnv() {
	_, codeName := os.LookupEnv("UBUNTU_CODENAME")
	if !codeName {
		slog.Error("UBUNTU_CODENAME env variable not set")
		os.Exit(1)
	}
}

func LoadOSReleaseEnv() {
	err := godotenv.Load("/etc/os-release")
	if err != nil {
		slog.Error("error loading .env file", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
