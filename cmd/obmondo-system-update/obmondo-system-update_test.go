package main

import (
	mock "go-scripts/mock"
	"go-scripts/util"
	"net/http"
	"testing"
)

func TestGetCustomerID(t *testing.T) {
	certname := "hostname.example"
	expected := "example"
	op := util.GetCustomerID(certname)
	if op != expected {
		t.Errorf("Failed to parse customer id, expeceted: %s, output: %s", expected, op)
	}
}

// Failing on ci as kernel can't be installed
// on the ci instance, fix it

// func TestGetInstalledKernel(t *testing.T) {
// 	distribution := "Ubuntu"
// 	installedKernel := GetInstalledKernel(distribution)
// 	if installedKernel == "" {
// 		t.Errorf("o/p : %s", installedKernel)
// 	}
// }

func TestGetServiceWindowStatus(t *testing.T) {
	expected := true
	mockObmondoClient := mock.NewMockObmondoClient()
	op := GetServiceWindowStatus(mockObmondoClient)
	if op != expected {
		t.Errorf("o/p: %t %t", expected, op)
	}
}

func TestCloseWindow(t *testing.T) {
	mockObmondoClient := mock.NewMockObmondoClient()
	op, err := CloseWidow(mockObmondoClient)
	if err != nil {
		t.Errorf("o/p: %+v", op)
	}
	defer op.Body.Close()
	if op.StatusCode != http.StatusOK {
		t.Errorf("o/p: %+v, err: %s", op, err.Error())
	}
}
