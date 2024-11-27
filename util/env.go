package util

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

func CheckPuppetEnv() {
	_, certOk := os.LookupEnv("PUPPETCERT")
	if !certOk {
		slog.Error("PUPPETCERT env variable not set")
		os.Exit(1)
	}

	_, keyOk := os.LookupEnv("PUPPETPRIVKEY")
	if !keyOk {
		slog.Error("PUPPETPRIVKEY env variable not set")
		os.Exit(1)
	}
}

func CheckCertNameEnv() {
	_, certnameOk := os.LookupEnv("CERTNAME")
	if !certnameOk {
		slog.Error("CERTNAME env variable not set")
		os.Exit(1)
	}
}

func CheckOSNameEnv() {
	_, codeName := os.LookupEnv("NAME")
	if !codeName {
		slog.Error("NAME env variable not set")
		os.Exit(1)
	}
}

func CheckOSVersionEnv() {
	_, codeName := os.LookupEnv("VERSION")
	if !codeName {
		slog.Error("VERSION env variable not set")
		os.Exit(1)
	}
}

func CheckUbuntuCodeNameEnv() {
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
