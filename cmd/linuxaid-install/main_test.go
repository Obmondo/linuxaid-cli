package main

import (
	"testing"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/config"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/constant"
)

// The install environment flag was renamed from --openvox-environment to --environment. The old
// spelling appears in the documented install commands, so it has to keep working.
func TestEnvironmentFlagAcceptsBothSpellings(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "current spelling", args: []string{"--environment", "v1_8_3"}},
		{name: "shorthand", args: []string{"-E", "v1_8_3"}},
		{name: "deprecated spelling", args: []string{"--openvox-environment", "v1_8_3"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Cleanup(func() {
				openvoxEnvFlag = constant.DefaultOpenvoxEnv
			})

			if err := rootCmd.ParseFlags(test.args); err != nil {
				t.Fatalf("could not parse %v: %v", test.args, err)
			}

			if openvoxEnvFlag != "v1_8_3" {
				t.Errorf("expected the flag to be v1_8_3, got %q", openvoxEnvFlag)
			}

			if environment := config.GetOpenvoxEnv(); environment != "v1_8_3" {
				t.Errorf("expected the resolved environment to be v1_8_3, got %q", environment)
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

	if err := rootCmd.ParseFlags([]string{"--environment", "v1_8_3"}); err != nil {
		t.Fatalf("could not parse flags: %v", err)
	}

	if environment := config.GetOpenvoxEnv(); environment != "v1_8_3" {
		t.Errorf("expected the resolved environment to be v1_8_3, got %q", environment)
	}
}
