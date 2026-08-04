package main

import (
	"testing"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/config"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/constant"
)

// testEnvironment is the environment the flag tests below install with.
const testEnvironment = "v1_8_3"

func TestEnvironmentFlag(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "current spelling", args: []string{"--environment", testEnvironment}},
		{name: "shorthand", args: []string{"-E", testEnvironment}},
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

			if environment := config.GetInstallEnvironment(); environment != testEnvironment {
				t.Errorf("expected the resolved environment to be %q, got %q", testEnvironment, environment)
			}
		})
	}
}

// viper runs with AutomaticEnv, so the flag is deliberately bound to a key that is not named
// "environment": such a key would be satisfied by any stray ENVIRONMENT variable on the host.
func TestEnvironmentFlagIgnoresStrayEnvironmentVariable(t *testing.T) {
	t.Setenv("ENVIRONMENT", "leaked-from-env")
	t.Cleanup(func() {
		openvoxEnvFlag = constant.DefaultOpenvoxEnv
	})

	if err := rootCmd.ParseFlags([]string{"--environment", testEnvironment}); err != nil {
		t.Fatalf("could not parse flags: %v", err)
	}

	if environment := config.GetInstallEnvironment(); environment != testEnvironment {
		t.Errorf("expected the resolved environment to be %q, got %q", testEnvironment, environment)
	}
}
