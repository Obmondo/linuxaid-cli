package helper

import (
	"fmt"
	"log/slog"
	"os"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/constant"
	"github.com/joho/godotenv"
)

func RequirePuppetEnv() {
	_, certOk := os.LookupEnv(constant.PuppetCertEnv)
	if !certOk {
		slog.Error(fmt.Sprintf("%s env variable not set", constant.PuppetCertEnv))
		os.Exit(1)
	}

	_, keyOk := os.LookupEnv(constant.PuppetPrivKeyEnv)
	if !keyOk {
		slog.Error(fmt.Sprintf("%s env variable not set", constant.PuppetPrivKeyEnv))
		os.Exit(1)
	}
}

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
