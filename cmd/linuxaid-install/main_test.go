package main

import (
	"testing"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/constant"
)

// testEnvironment is the environment the flag tests below install with.
const testEnvironment = "v1_8_3"

func TestInstallEnvironment(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		envVar   string
		expected string
	}{
		{name: "flag", args: []string{"--environment", testEnvironment}, expected: testEnvironment},
		{name: "shorthand", args: []string{"-E", testEnvironment}, expected: testEnvironment},
		{name: "environment variable", envVar: testEnvironment, expected: testEnvironment},
		{name: "the flag wins over the environment variable", args: []string{"--environment", testEnvironment}, envVar: "from-env", expected: testEnvironment},
		{name: "neither given", expected: constant.DefaultOpenvoxEnv},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(constant.EnvVarEnvironment, test.envVar)
			t.Cleanup(func() {
				openvoxEnvFlag = ""
			})

			if err := rootCmd.ParseFlags(test.args); err != nil {
				t.Fatalf("could not parse %v: %v", test.args, err)
			}

			if environment := installEnvironment(); environment != test.expected {
				t.Errorf("expected the environment to be %q, got %q", test.expected, environment)
			}
		})
	}
}
