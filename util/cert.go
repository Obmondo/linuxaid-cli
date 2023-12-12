package util

import (
	"crypto/x509"
	"encoding/pem"
	"log"
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
		log.Println("In correct format for certname")
		return ""
	}
	return parts[1]
}

// Need this, otherwise remotelog func wont work
func IsCaCertificateInstalled(cmd string) bool {
	pipe := script.Exec(cmd)
	pipe.Wait()
	exitStatus := pipe.ExitStatus()
	return exitStatus == 0
}
