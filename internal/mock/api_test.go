package mock

import (
	"testing"
	"time"
)

func TestGetServiceWindowStatus(t *testing.T) {
	mockObmondoClient := NewMockObmondoClient()
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

	if !serviceWindowNow.DoesWindowExist {
		t.Errorf("Expected service window to exist, but got: %t", serviceWindowNow.DoesWindowExist)
	}
	if serviceWindowNow.Linux == nil {
		t.Fatal("Expected linux window details to be present, but got nil")
	}
	if !serviceWindowNow.Linux.NeedsReboot {
		t.Errorf("Expected needs reboot to be true, but got: %t", serviceWindowNow.Linux.NeedsReboot)
	}
	if serviceWindowNow.Linux.LinuxAidTag != "11.0.0" {
		t.Errorf("Expected linuxaid tag to be '11.0.0', but got: %s", serviceWindowNow.Linux.LinuxAidTag)
	}

}

func TestCloseWindow(t *testing.T) {
	mockObmondoClient := NewMockObmondoClient()

	if err := mockObmondoClient.CloseServiceWindow("automatic", "hostname.example", time.UTC.String(), "server has been updated"); err != nil {
		t.Errorf("o/p: %+v", err)
	}
}
