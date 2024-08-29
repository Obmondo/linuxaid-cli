package util

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func CheckPuppetEnv() {
	_, certOk := os.LookupEnv("PUPPETCERT")
	if !certOk {
		log.Fatal("PUPPETCERT env variable not set")
	}

	_, keyOk := os.LookupEnv("PUPPETPRIVKEY")
	if !keyOk {
		log.Fatal("PUPPETPRIVKEY env variable not set")
	}
}

func CheckCertNameEnv() {
	_, certnameOk := os.LookupEnv("CERTNAME")
	if !certnameOk {
		log.Fatal("CERTNAME env variable not set")
	}
}

func CheckOSNameEnv() {
	_, codeName := os.LookupEnv("NAME")
	if !codeName {
		log.Fatal("NAME env variable not set")
	}
}

func CheckOSVersionEnv() {
	_, codeName := os.LookupEnv("VERSION")
	if !codeName {
		log.Fatal("VERSION env variable not set")
	}
}

func CheckUbuntuCodeNameEnv() {
	_, codeName := os.LookupEnv("UBUNTU_CODENAME")
	if !codeName {
		log.Fatal("UBUNTU_CODENAME env variable not set")
	}
}

func LoadOSReleaseEnv() {
	err := godotenv.Load("/etc/os-release")
	if err != nil {
		log.Fatal("Error loading .env file:", err)
	}
}
