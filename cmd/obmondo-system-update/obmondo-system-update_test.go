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

func TestGetServiceWindowStatus(t *testing.T) {
	expected := true
	mockObmondoClient := mock.NewMockObmondoClient()
	op, err := GetServiceWindowStatus(mockObmondoClient)
	if err != nil {
		t.Errorf("o/p: %+v", err)
	}

	if op != expected {
		t.Errorf("o/p: %t %t", expected, op)
	}
}

func TestCloseWindow(t *testing.T) {
	mockObmondoClient := mock.NewMockObmondoClient()
	var closeWindowSuccessStatuses = map[int]bool{http.StatusAccepted: true, http.StatusNoContent: true, http.StatusAlreadyReported: true}

	op, err := closeWindow(mockObmondoClient)
	if err != nil {
		t.Errorf("o/p: %+v", op)
	}

	defer op.Body.Close()
	if !closeWindowSuccessStatuses[op.StatusCode] {
		t.Errorf("o/p: %+v, err: %s", op, err.Error())
	}
}

// Need tests for 204 and 208 and a failed scenario as well
