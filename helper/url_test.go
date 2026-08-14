package helper

import "testing"

func TestExtractHostname(t *testing.T) {
	tests := []struct {
		name     string
		rawURL   string
		expected string
	}{
		{"https URL", "https://prom-test.obmondo.com", "prom-test.obmondo.com"},
		{"https URL with port", "https://prom-test.obmondo.com:9090", "prom-test.obmondo.com"},
		{"https URL with path", "https://prom-test.obmondo.com/api/v1", "prom-test.obmondo.com"},
		{"http URL", "http://prometheus.local:9090", "prometheus.local"},
		{"empty string", "", ""},
		{"bare hostname (no scheme)", "prom-test.obmondo.com", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractHostname(tt.rawURL)
			if got != tt.expected {
				t.Errorf("ExtractHostname(%q) = %q, want %q", tt.rawURL, got, tt.expected)
			}
		})
	}
}

func TestNormalizeToHostname(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		expected string
	}{
		{"URL is reduced to its hostname", "https://enableit.puppet.obmondo.com:8140", "enableit.puppet.obmondo.com"},
		{"bare hostname is kept unchanged", "enableit.puppet.obmondo.com", "enableit.puppet.obmondo.com"},
		{"empty string stays empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeToHostname(tt.raw)
			if got != tt.expected {
				t.Errorf("NormalizeToHostname(%q) = %q, want %q", tt.raw, got, tt.expected)
			}
		})
	}
}
