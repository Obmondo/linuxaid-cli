package checkstatus

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

type DomainConfig struct {
	IP         string `yaml:"ip"`
	CommonName string `yaml:"common_name"`
}

type Result struct {
	ObmondoEndpointsReachable map[string]bool `yaml:"obmondo_endpoints_reachable"`
}

type Config struct {
	Domains []DomainConfig `yaml:"domains"`
}

// CheckStatus checks the status of the domains in the config file
func CheckStatus() {
	config, err := loadConfig("config.yaml")
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	results := make(map[string]bool)
	for _, domain := range config.Domains {
		result, err := connectAndVerify(&domain)
		if err != nil {
			results[domain.IP] = false
		} else {
			results[domain.IP] = result
		}
	}

	output := Result{ObmondoEndpointsReachable: results}

	yamlOutput, err := yaml.Marshal(output)
	if err != nil {
		fmt.Println("Error marshalling YAML:", err)
		return
	}

	fmt.Println(string(yamlOutput))
}

// loadConfig loads the config from the given filename
func loadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// connectAndVerify connects to the IP address and verifies the certificate
func connectAndVerify(domain *DomainConfig) (bool, error) {
	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
	}

	// Connect to the IP address and verify the certificate
	conn, err := tls.DialWithDialer(dialer, "tcp", domain.IP+":443", &tls.Config{
		InsecureSkipVerify: true,
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			cert, err := x509.ParseCertificate(rawCerts[0])
			if err != nil {
				return fmt.Errorf("failed to parse certificate: %v", err)
			}

			// Verify the Common Name (CN) of the certificate
			if cert.Subject.CommonName != domain.CommonName {
				return fmt.Errorf("common name mismatch: expected %s, got %s", domain.CommonName, cert.Subject.CommonName)
			}

			return nil
		},
	})

	if err != nil {
		return false, err
	}
	defer conn.Close()

	return true, nil
}
