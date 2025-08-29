package main

import (
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"gitea.obmondo.com/EnableIT/go-scripts/helper"
	"gitea.obmondo.com/EnableIT/go-scripts/mock"
)

func TestGetCustomerID(t *testing.T) {
	t.Setenv("CERTNAME", "hostname.example")
	expected := "example"
	op := helper.GetCustomerID()
	if op != expected {
		t.Errorf("Failed to parse customer id, expeceted: %s, output: %s", expected, op)
	}
}

func TestGetServiceWindowStatus(t *testing.T) {
	mockObmondoClient := mock.NewMockObmondoClient()
	serviceWindowNow, err := GetServiceWindowStatus(mockObmondoClient)
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
	var closeWindowSuccessStatuses = map[int]bool{http.StatusAccepted: true, http.StatusNoContent: true, http.StatusAlreadyReported: true}

	op, err := closeWindow(mockObmondoClient, "automatic", time.UTC.String())
	if err != nil {
		t.Errorf("o/p: %+v", op)
	}

	defer op.Body.Close()
	if !closeWindowSuccessStatuses[op.StatusCode] {
		t.Errorf("o/p: %+v, err: %s", op, err.Error())
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
