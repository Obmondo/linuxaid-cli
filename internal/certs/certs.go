package certs

import (
	"crypto/x509"
	"encoding/pem"
	"log/slog"
	"net"
	"os"
	"strings"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/constant"
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

// GetCertname resolves this node's certname: the puppet host certificate, then the puppet private
// key, then the configured value the caller passes in, and finally the machine's FQDN.
func GetCertname(configured string) string {
	puppetCert, puppetCertExists := os.LookupEnv(constant.PuppetCertEnv)
	if puppetCertExists {
		return GetCommonNameFromCertFile(puppetCert)
	}

	certname := getCertnameFromPrivateKey()
	if certname != "" {
		return certname
	}

	if configured != "" {
		return configured
	}

	return getHostnameFQDN()
}

// getHostnameFQDN returns the machine's fully qualified hostname, falling back
// to the short hostname when the FQDN cannot be resolved. Certnames are always
// lowercase.
func getHostnameFQDN() string {
	hostname, err := os.Hostname()
	if err != nil {
		slog.Debug("failed to get hostname", slog.Any("error", err))
		return ""
	}
	hostname = strings.ToLower(hostname)

	// Resolve the canonical name, like `hostname -f` does via getaddrinfo.
	if cname, err := net.LookupCNAME(hostname); err == nil {
		if fqdn := strings.ToLower(strings.TrimSuffix(cname, ".")); fqdn != "" {
			return fqdn
		}
	}

	return hostname
}

func GetCustomerID(certname string) string {
	parts := strings.Split(certname, ".")
	// nolint: mnd
	if len(parts) >= 2 {
		return parts[1]
	}

	return ""
}
