package utils

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"gitea.obmondo.com/go-scripts/config"
	"gitea.obmondo.com/go-scripts/constants"
	"github.com/bitfield/script"
)

const (
	two = 2
)

func GetCommonNameFromCertFile(certPath string) string {
	hostCert, err := os.ReadFile(certPath)
	if err != nil {
		slog.Error("failed to fetch hostcert", slog.String("error", err.Error()))
		return ""
	}

	block, _ := pem.Decode(hostCert)
	if block == nil {
		slog.Error("failed to decode hostcert")
		return ""
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		slog.Error("failed to parse hostcert", slog.String("error", err.Error()))
		return ""
	}

	return cert.Subject.CommonName
}

func getCustomerIDFromPuppetCertString(puppetCertString string) string {
	if strings.Contains(puppetCertString, ".") && len(strings.Split(puppetCertString, ".")) == 3 && len(strings.Split(puppetCertString, ".")[1]) > 0 {
		return strings.Split(puppetCertString, ".")[1]
	}
	return ""
}

func GetCustomerID() string {
	certName := config.GetCertName()
	parts := strings.Split(certName, ".")
	if len(parts) >= two {
		return parts[1]
	}

	fmt.Println(certName)
	puppetCert, puppetCertExists := os.LookupEnv(constants.PuppetCertEnv)
	if puppetCertExists && len(puppetCert) > 0 {
		return getCustomerIDFromPuppetCertString(puppetCert)
	}

	return ""
}

// Need this, otherwise remotelog func wont work
func IsCaCertificateInstalled(cmd string) bool {
	pipe := script.Exec(cmd)
	if err := pipe.Wait(); err != nil {
		slog.Error("failed to determine if ca-certificates is installed", slog.String("error", err.Error()))
	}
	exitStatus := pipe.ExitStatus()
	return exitStatus == 0
}
