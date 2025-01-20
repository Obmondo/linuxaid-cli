package utils

import (
	"crypto/x509"
	"encoding/pem"
	"log/slog"
	"os"
	"strings"

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

func GetCustomerID(certname string) string {
	parts := strings.Split(certname, ".")
	if len(parts) < two {
		slog.Error("incorrect format for certname")
		return ""
	}
	return parts[1]
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
