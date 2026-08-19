package system

import (
	"errors"
	"fmt"

	"os"

	"github.com/joho/godotenv"
)

// requireEnv reports a missing environment variable as an error rather than terminating: the
// caller decides whether a missing variable is fatal for what it is doing.
func requireEnv(name string) error {
	if _, ok := os.LookupEnv(name); !ok {
		return fmt.Errorf("%s env variable not set", name)
	}

	return nil
}

func RequireOSNameEnv() error {
	return requireEnv("NAME")
}

func RequireOSVersionEnv() error {
	return requireEnv("VERSION")
}

func RequireUbuntuCodeNameEnv() error {
	return requireEnv("UBUNTU_CODENAME")
}

func LoadOSReleaseEnv() error {
	if err := godotenv.Load("/etc/os-release"); err != nil {
		return fmt.Errorf("could not load /etc/os-release: %w", err)
	}

	return nil
}

var errNotRoot = errors.New("this command needs to be run as the root user")
