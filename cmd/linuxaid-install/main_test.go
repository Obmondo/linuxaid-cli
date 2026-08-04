package main

import (
	"testing"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/config"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/constant"
)

// testEnvironment is the environment the flag tests below install with.
const testEnvironment = "v1_8_3"

// The install environment flag was renamed from --openvox-environment to --environment. The old
// spelling appears in the documented install commands, so it has to keep working.
func TestEnvironmentFlagAcceptsBothSpellings(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "current spelling", args: []string{"--environment", testEnvironment}},
		{name: "shorthand", args: []string{"-E", testEnvironment}},
		{name: "deprecated spelling", args: []string{"--openvox-environment", testEnvironment}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Cleanup(func() {
				openvoxEnvFlag = constant.DefaultOpenvoxEnv
			})

			if err := rootCmd.ParseFlags(test.args); err != nil {
				t.Fatalf("could not parse %v: %v", test.args, err)
			}

			if openvoxEnvFlag != testEnvironment {
				t.Errorf("expected the flag to be %q, got %q", testEnvironment, openvoxEnvFlag)
			}

			if environment := config.GetOpenvoxEnv(); environment != testEnvironment {
				t.Errorf("expected the resolved environment to be %q, got %q", testEnvironment, environment)
			}
		})
	}
}

// viper runs with AutomaticEnv, so the flag is deliberately kept under the openvox-environment key:
// a key named "environment" would be satisfied by any stray ENVIRONMENT variable on the host.
func TestEnvironmentFlagIgnoresStrayEnvironmentVariable(t *testing.T) {
	t.Setenv("ENVIRONMENT", "leaked-from-env")
	t.Cleanup(func() {
		openvoxEnvFlag = constant.DefaultOpenvoxEnv
	})

	if err := rootCmd.ParseFlags([]string{"--environment", testEnvironment}); err != nil {
		t.Fatalf("could not parse flags: %v", err)
	}

	if environment := config.GetOpenvoxEnv(); environment != testEnvironment {
		t.Errorf("expected the resolved environment to be %q, got %q", testEnvironment, environment)
	}
}
