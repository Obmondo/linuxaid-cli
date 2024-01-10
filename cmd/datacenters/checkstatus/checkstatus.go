package checkstatus

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

// Config represents the structure of the YAML file
type Config struct {
	Domains map[string]string `yaml:"domains"`
}

func CheckStatus() {
	// Read YAML file
	fileContent, err := os.ReadFile("domains.yaml")
	if err != nil {
		fmt.Println("Error reading YAML file:", err)
		return
	}

	// Parse YAML content
	var config Config
	err = yaml.Unmarshal(fileContent, &config)
	if err != nil {
		fmt.Println("Error parsing YAML:", err)
		return
	}

	// Verify TLS connection for each domain
	for domain, ip := range config.Domains {
		fmt.Printf("Verifying TLS connection for %s (IP: %s)\n", domain, ip)
		if verifyTLSConnection(domain, ip) {
			fmt.Println("TLS connection successful")
		} else {
			fmt.Println("TLS connection failed")
		}
	}
}

func verifyTLSConnection(domain, ip string) bool {
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", fmt.Sprintf("%s:443", ip), &tls.Config{
		InsecureSkipVerify: true, // traefik certificates are self-signed
	})
	if err != nil {
		fmt.Println("Error connecting:", err)
		return false
	}
	defer conn.Close()

	return true
}
