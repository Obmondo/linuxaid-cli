package helper

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

// requireEnv exits when the given environment variable is not set.
func requireEnv(name string) {
	if _, ok := os.LookupEnv(name); !ok {
		slog.Error(fmt.Sprintf("%s env variable not set", name))
		os.Exit(1)
	}
}

func RequireOSNameEnv() {
	requireEnv("NAME")
}

func RequireOSVersionEnv() {
	requireEnv("VERSION")
}

func RequireUbuntuCodeNameEnv() {
	requireEnv("UBUNTU_CODENAME")
}

func LoadOSReleaseEnv() {
	err := godotenv.Load("/etc/os-release")
	if err != nil {
		slog.Error("error loading .env file", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

// LoadPuppetEnv doesnt throw error if the file doesnt exist
func LoadPuppetEnv() {
	err := godotenv.Load("/etc/default/run_puppet")
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		slog.Error("error loading .env file", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
