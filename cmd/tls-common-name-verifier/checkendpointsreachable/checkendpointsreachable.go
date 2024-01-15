// Package tlscommonnameverifier provides a function to check the reachability
// and TLS certificate common name verification for a list of domains.
package checkendpointsreachable

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

// DomainConfig represents the configuration for a domain.
type DomainConfig struct {
	IP         string `yaml:"ip"`
	CommonName string `yaml:"common_name"`
}

type Config struct {
	Domains []DomainConfig `yaml:"domains"`
}

// ObmondoEndpointStatus checks the reachability and TLS certificate common name
// verification for a list of domains.
func ObmondoEndpointStatus(config Config) (map[string]bool, error) {
	results := make(map[string]bool)

	for _, domain := range config.Domains {
		result, err := connectAndVerify(&domain)
		if err != nil {
			results[domain.IP] = false
		} else {
			results[domain.IP] = result
		}
	}

	return results, nil
}

// connectAndVerify connects to the domain and verifies the TLS certificate
func connectAndVerify(domain *DomainConfig) (bool, error) {
	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
	}

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

// LoadConfig reads the configuration from a YAML file.
func LoadConfig(filename string) (*Config, error) {
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

// PrintYAML prints the YAML representation of the map containing the reachability
// status of the domains.
func PrintYAML(results map[string]bool) (string, error) {
	output := make(map[string]interface{})
	output["obmondo_endpoints_reachable"] = results

	yamlOutput, err := yaml.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("error marshalling YAML: %v", err)
	}

	return string(yamlOutput), nil
}
