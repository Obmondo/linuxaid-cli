package certs

import (
	"testing"
)

func TestGetCustomerID(t *testing.T) {
	t.Setenv("CERTNAME", "hostname.example")
	expected := "example"
	op := GetCustomerID("hostname.example")
	if op != expected {
		t.Errorf("Failed to parse customer id, expeceted: %s, output: %s", expected, op)
	}
}
