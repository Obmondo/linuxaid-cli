package helper

import (
	"crypto/x509"
	"encoding/pem"
	"log/slog"
	"os"
	"strings"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/config"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/constant"
	"github.com/bitfield/script"
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

func getCertnameFromPrivateKey() string {
	items, err := os.ReadDir(constant.PuppetPrivKeyPath)
	if err != nil {
		slog.Debug("failed to list directory", slog.Any("error", err), slog.String("path", constant.PuppetPrivKeyPath))
		return ""
	}

	for _, item := range items {
		if item.IsDir() {
			continue
		}
		certname, ok := strings.CutSuffix(item.Name(), ".pem")
		if !ok {
			continue
		}
		return certname
	}

	slog.Debug("no file found in the directory", slog.Any("error", err), slog.String("path", constant.PuppetPrivKeyPath))
	return ""
}

func GetCertname() string {
	puppetCert, puppetCertExists := os.LookupEnv(constant.PuppetCertEnv)
	if puppetCertExists {
		return GetCommonNameFromCertFile(puppetCert)
	}

	certname := getCertnameFromPrivateKey()
	if certname != "" {
		return certname
	}

	return config.GetCertname()
}

func GetCustomerID(certname string) string {
	parts := strings.Split(certname, ".")
	// nolint: mnd
	if len(parts) >= 2 {
		return parts[1]
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
