package main

import (
	"path/filepath"
	"testing"
	"time"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/helper"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/mock"
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

}

func TestCloseWindow(t *testing.T) {
	mockObmondoClient := mock.NewMockObmondoClient()

	if err := mockObmondoClient.CloseServiceWindow("automatic", "hostname.example", time.UTC.String()); err != nil {
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
