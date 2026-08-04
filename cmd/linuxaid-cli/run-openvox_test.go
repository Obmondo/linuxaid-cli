package main

import (
	"errors"
	"testing"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/constant"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/mock"
	api "gitea.obmondo.com/EnableIT/linuxaid-cli/pkg/obmondo"
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
