package main

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/constant"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/helper"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/mock"
	api "gitea.obmondo.com/EnableIT/linuxaid-cli/pkg/obmondo"
)

func TestGetCustomerID(t *testing.T) {
	t.Setenv("CERTNAME", "hostname.example")
	expected := "example"
	op := helper.GetCustomerID("hostname.example")
	if op != expected {
		t.Errorf("Failed to parse customer id, expeceted: %s, output: %s", expected, op)
	}
}

func TestGetServiceWindowStatus(t *testing.T) {
	mockObmondoClient := mock.NewMockObmondoClient()
	serviceWindowNow, err := mockObmondoClient.GetServiceWindowStatus()
	if err != nil {
		t.Errorf("o/p: %+v", err)
	}

	if !serviceWindowNow.IsWindowOpen {
		t.Errorf("Expected service window to be open, but got: %t", serviceWindowNow.IsWindowOpen)
	}

	if serviceWindowNow.WindowType != "automatic" {
		t.Errorf("Expected window type to be 'automatic', but got: %s", serviceWindowNow.WindowType)
	}
	if serviceWindowNow.Timezone != "UTC" {
		t.Errorf("Expected window timezone to be 'UTC', but got: %s", serviceWindowNow.Timezone)
	}

	if !serviceWindowNow.DoesWindowExist {
		t.Errorf("Expected service window to exist, but got: %t", serviceWindowNow.DoesWindowExist)
	}
	if serviceWindowNow.Linux == nil {
		t.Fatal("Expected linux window details to be present, but got nil")
	}
	if !serviceWindowNow.Linux.NeedsReboot {
		t.Errorf("Expected needs reboot to be true, but got: %t", serviceWindowNow.Linux.NeedsReboot)
	}
	if serviceWindowNow.Linux.LinuxAidTag != "11.0.0" {
		t.Errorf("Expected linuxaid tag to be '11.0.0', but got: %s", serviceWindowNow.Linux.LinuxAidTag)
	}

}

func TestCloseWindow(t *testing.T) {
	mockObmondoClient := mock.NewMockObmondoClient()

	if err := mockObmondoClient.CloseServiceWindow("automatic", "hostname.example", time.UTC.String(), "server has been updated"); err != nil {
		t.Errorf("o/p: %+v", err)
	}
}

func TestGetInstalledKernel(t *testing.T) {
	testBootDirectory, err := filepath.Abs("../../test/boot/")
	if err != nil {
		t.Fatal(err)
	}
	expectedKernelOutput := "6.11.0-3-generic"

	latestKernel, err := getInstalledKernel(testBootDirectory)
	if err != nil {
		t.Fatal(err)
	}
	if latestKernel != expectedKernelOutput {
		t.Errorf("\n expected: %s\n actual: %s", expectedKernelOutput, latestKernel)
		t.FailNow()
	}

}

// Need tests for 204 and 208 and a failed scenario as well

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
			got := extractHostname(tt.rawURL)
			if got != tt.expected {
				t.Errorf("extractHostname(%q) = %q, want %q", tt.rawURL, got, tt.expected)
			}
		})
	}
}

// mockSettingsClient is a mock that returns configurable customer settings.
type mockSettingsClient struct {
	mock.MockObmondoClient
	settings *api.CustomerSettings
	err      error
}

func (m *mockSettingsClient) GetCustomerSettings(_ string) (*api.CustomerSettings, error) {
	return m.settings, m.err
}

func TestResolveCustomerURLs(t *testing.T) {
	defaultPuppetServer := constant.DefaultPuppetServerCustomerID + constant.DefaultPuppetServerDomainSuffix

	t.Run("returns custom URLs when settings exist", func(t *testing.T) {
		client := &mockSettingsClient{
			settings: &api.CustomerSettings{
				CustomerID: "enableit",
				LinuxAid: &api.LinuxAidSettings{
					PrometheusURL:   "https://prom-test.obmondo.com",
					PuppetserverURL: "https://enableit.puppet.obmondo.com",
				},
			},
		}
		t.Setenv("CERTNAME", "testserver.enableit")

		promHost, puppetHost := resolveCustomerURLs(client, "testserver.enableit")
		if promHost != "prom-test.obmondo.com" {
			t.Errorf("prometheus host = %q, want %q", promHost, "prom-test.obmondo.com")
		}
		if puppetHost != "enableit.puppet.obmondo.com" {
			t.Errorf("puppet server host = %q, want %q", puppetHost, "enableit.puppet.obmondo.com")
		}
	})

	t.Run("returns defaults when linuxaid settings are nil", func(t *testing.T) {
		client := &mockSettingsClient{
			settings: &api.CustomerSettings{
				CustomerID: "enableit",
				LinuxAid:   nil,
			},
		}
		t.Setenv("CERTNAME", "testserver.enableit")

		promHost, puppetHost := resolveCustomerURLs(client, "testserver.enableit")
		if promHost != defaultPrometheusHost {
			t.Errorf("prometheus host = %q, want default %q", promHost, defaultPrometheusHost)
		}
		if puppetHost != defaultPuppetServer {
			t.Errorf("puppet server host = %q, want default %q", puppetHost, defaultPuppetServer)
		}
	})

	t.Run("returns defaults when API call fails", func(t *testing.T) {
		client := &mockSettingsClient{
			settings: nil,
			err:      errors.New("connection refused"),
		}
		t.Setenv("CERTNAME", "testserver.enableit")

		promHost, puppetHost := resolveCustomerURLs(client, "testserver.enableit")
		if promHost != defaultPrometheusHost {
			t.Errorf("prometheus host = %q, want default %q", promHost, defaultPrometheusHost)
		}
		if puppetHost != defaultPuppetServer {
			t.Errorf("puppet server host = %q, want default %q", puppetHost, defaultPuppetServer)
		}
	})

	t.Run("returns defaults when certname has no customer ID", func(t *testing.T) {
		client := &mockSettingsClient{
			settings: &api.CustomerSettings{},
		}

		promHost, puppetHost := resolveCustomerURLs(client, "")
		if promHost != defaultPrometheusHost {
			t.Errorf("prometheus host = %q, want default %q", promHost, defaultPrometheusHost)
		}
		if puppetHost != defaultPuppetServer {
			t.Errorf("puppet server host = %q, want default %q", puppetHost, defaultPuppetServer)
		}
	})

	t.Run("uses default for prometheus when only puppetserver is set", func(t *testing.T) {
		client := &mockSettingsClient{
			settings: &api.CustomerSettings{
				CustomerID: "enableit",
				LinuxAid: &api.LinuxAidSettings{
					PuppetserverURL: "https://custom.puppet.server.com",
				},
			},
		}
		t.Setenv("CERTNAME", "testserver.enableit")

		promHost, puppetHost := resolveCustomerURLs(client, "testserver.enableit")
		if promHost != defaultPrometheusHost {
			t.Errorf("prometheus host = %q, want default %q", promHost, defaultPrometheusHost)
		}
		if puppetHost != "custom.puppet.server.com" {
			t.Errorf("puppet server host = %q, want %q", puppetHost, "custom.puppet.server.com")
		}
	})

	t.Run("uses default for puppetserver when only prometheus is set", func(t *testing.T) {
		client := &mockSettingsClient{
			settings: &api.CustomerSettings{
				CustomerID: "enableit",
				LinuxAid: &api.LinuxAidSettings{
					PrometheusURL: "https://custom-prom.example.com:9090",
				},
			},
		}
		t.Setenv("CERTNAME", "testserver.enableit")

		promHost, puppetHost := resolveCustomerURLs(client, "testserver.enableit")
		if promHost != "custom-prom.example.com" {
			t.Errorf("prometheus host = %q, want %q", promHost, "custom-prom.example.com")
		}
		if puppetHost != defaultPuppetServer {
			t.Errorf("puppet server host = %q, want default %q", puppetHost, defaultPuppetServer)
		}
	})
}
