package util

import (
	"os"
	"log"
	"strings"
	"crypto/x509"
	"encoding/pem"
)

func GetCommonNameFromCertFile(certPath string) string {
	hostCert, err := os.ReadFile(certPath)
	if err != nil {
		log.Printf("Failed to fetch hostcert: %s", err)
		return ""
	}

	block, _ := pem.Decode(hostCert)
	if block == nil {
		log.Printf("Failed to decode hostcert: %s", err)
		return ""
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		log.Printf("Failed to parse hostcert: %s", err)
		return ""
	}

	return cert.Subject.CommonName
}

func GetCustomerId(certname string) string {
	parts := strings.Split(certname, ".")
	if len(parts) < 2 {
		log.Println("In correct formatt for certname")
		return ""
	}
	return parts[1]
}