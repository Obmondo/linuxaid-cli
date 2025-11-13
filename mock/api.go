package mock

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/helper"
	api "gitea.obmondo.com/EnableIT/linuxaid-cli/pkg/obmondo"
)

// nolint: revive
type MockObmondoClient struct{}

func (*MockObmondoClient) VerifyInstallToken(_ *api.InstallScriptInput) error {
	return nil
}

// NotifyInstallScriptFailure implements api.ObmondoClient.
func (*MockObmondoClient) NotifyInstallScriptFailure(_ *api.InstallScriptInput) error {
	return nil
}

// ServerPing implements api.ObmondoClient.
func (*MockObmondoClient) ServerPing() error {
	return nil
}

// UpdatePuppetLastRunReport implements api.ObmondoClient.
func (*MockObmondoClient) UpdatePuppetLastRunReport() error {
	return nil
}

func (*MockObmondoClient) FetchServiceWindowStatus() (*http.Response, error) {
	data := map[string]interface{}{
		"status":  http.StatusOK,
		"success": true,
		"data": map[string]interface{}{
			"is_window_open": true,
			"window_type":    "automatic",
			"timezone":       "UTC",
		},
		"message":    "successfully got current service window status",
		"resolution": "",
		"error_text": "",
	}

	dataBytes, _ := json.Marshal(data)
	response := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBuffer(dataBytes)),
		Header:     make(http.Header),
	}
	return response, nil
}

func (m *MockObmondoClient) GetServiceWindowStatus() (*api.ServiceWindow, error) {
	resp, err := m.FetchServiceWindowStatus()
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	_, responseBody, err := helper.ParseResponse(resp)
	if err != nil {
		return nil, err
	}

	return api.GetServiceWindowDetails(responseBody)
}

func (*MockObmondoClient) CloseServiceWindow(string, string, string) error {
	return nil
}

func (*MockObmondoClient) CloseServiceWindowNow(string, string) (*http.Response, error) {
	response := &http.Response{
		StatusCode: http.StatusAccepted,
		Body:       io.NopCloser(bytes.NewBufferString("")),
		Header:     make(http.Header),
	}
	return response, nil
}

func NewMockObmondoClient() api.ObmondoClient {
	return &MockObmondoClient{}
}
