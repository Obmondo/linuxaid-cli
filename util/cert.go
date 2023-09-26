package util

import (
	"crypto/x509"
	"encoding/pem"
	"log"
	"os"
	"strings"
)

const (
	two = 2
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

func GetCustomerID(certname string) string {
	parts := strings.Split(certname, ".")
	if len(parts) < two {
		log.Println("In correct formatt for certname")
		return ""
	}
	return parts[1]
}
