package app

import (
	"errors"
	"testing"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/constant"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/mock"
	api "gitea.obmondo.com/EnableIT/linuxaid-cli/internal/obmondo"
)

// failingEnvironmentClient stands in for an unreachable Obmondo API.
type failingEnvironmentClient struct {
	api.ObmondoClient
}

func (*failingEnvironmentClient) GetServerEnvironment(_ string) (string, error) {
	return "", errors.New("api is unreachable")
}

// emptyEnvironmentClient answers without resolving an environment.
type emptyEnvironmentClient struct {
	api.ObmondoClient
}

func (*emptyEnvironmentClient) GetServerEnvironment(_ string) (string, error) {
	return "", nil
}

func TestResolveOpenvoxEnvironment(t *testing.T) {
	const certname = "hostname.example"

	tests := []struct {
		name       string
		flagValue  string
		client     api.ObmondoClient
		opensource bool
		expected   string
	}{
		{
			name:      "the flag wins over the environment set in Obmondo",
			flagValue: "testing",
			client:    mock.NewMockObmondoClient(),
			expected:  "testing",
		},
		{
			name:     "without the flag the environment comes from Obmondo",
			client:   mock.NewMockObmondoClient(),
			expected: mock.MockServerEnvironment,
		},
		{
			name:     "an unreachable api falls back to the default environment",
			client:   &failingEnvironmentClient{},
			expected: constant.DefaultOpenvoxEnv,
		},
		{
			name:     "an unresolved environment falls back to the default environment",
			client:   &emptyEnvironmentClient{},
			expected: constant.DefaultOpenvoxEnv,
		},
		{
			// opensource nodes are not registered with Obmondo, so they never call the API
			name:       "opensource nodes use the default environment",
			client:     &failingEnvironmentClient{},
			opensource: true,
			expected:   constant.DefaultOpenvoxEnv,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			environment := resolveOpenvoxEnvironment(test.client, certname, test.flagValue, test.opensource)
			if environment != test.expected {
				t.Errorf("expected environment %q, got %q", test.expected, environment)
			}
		})
	}
}

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

func TestResolveSystemUpdateEnvironment(t *testing.T) {
	const certname = "hostname.example"

	automaticWindow := func(tag string) *api.ServiceWindow {
		return &api.ServiceWindow{
			IsWindowOpen:    true,
			DoesWindowExist: true,
			WindowType:      constant.ServiceWindowTypeAutomatic,
			Timezone:        "UTC",
			Linux:           &api.LinuxWindowDetails{LinuxAidTag: tag},
		}
	}

	tests := []struct {
		name          string
		serviceWindow *api.ServiceWindow
		client        api.ObmondoClient
		opensource    bool
		expected      string
	}{
		{
			name:          "an automatic window pins the environment for the run",
			serviceWindow: automaticWindow("v11.0.0"),
			client:        mock.NewMockObmondoClient(),
			expected:      "v11_0_0",
		},
		{
			name:          "an automatic window without a tag falls back to the API",
			serviceWindow: automaticWindow(""),
			client:        mock.NewMockObmondoClient(),
			expected:      mock.MockServerEnvironment,
		},
		{
			name: "an adhoc window falls back to the API",
			serviceWindow: &api.ServiceWindow{
				IsWindowOpen:    true,
				DoesWindowExist: true,
				WindowType:      "adhoc",
				Timezone:        "UTC",
			},
			client:   mock.NewMockObmondoClient(),
			expected: mock.MockServerEnvironment,
		},
		{
			// opensource nodes are not registered with Obmondo, so nothing pins a tag for them
			name:          "opensource nodes use the default environment",
			serviceWindow: automaticWindow("v11.0.0"),
			client:        mock.NewMockObmondoClient(),
			opensource:    true,
			expected:      constant.DefaultOpenvoxEnv,
		},
		{
			name:          "an unreachable API falls back to the default environment",
			serviceWindow: automaticWindow(""),
			client:        &failingEnvironmentClient{},
			expected:      constant.DefaultOpenvoxEnv,
		},
		{
			name:          "no window at all falls back to the API",
			serviceWindow: nil,
			client:        mock.NewMockObmondoClient(),
			expected:      mock.MockServerEnvironment,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			environment := resolveSystemUpdateEnvironment(test.client, certname, test.serviceWindow, test.opensource)
			if environment != test.expected {
				t.Errorf("expected environment %q, got %q", test.expected, environment)
			}
		})
	}
}
