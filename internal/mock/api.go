package mock

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/httpx"
	api "gitea.obmondo.com/EnableIT/linuxaid-cli/internal/obmondo"
)

// MockServerEnvironment is the puppet environment the mock API resolves for every certname.
const MockServerEnvironment = "v1_0_0"

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

// GetServerEnvironment implements api.ObmondoClient.
func (*MockObmondoClient) GetServerEnvironment(_ string) (string, error) {
	return MockServerEnvironment, nil
}

func (*MockObmondoClient) GetCustomerSettings(_ string) (*api.CustomerSettings, error) {
	return &api.CustomerSettings{}, nil
}

func (*MockObmondoClient) FetchServiceWindowStatus() (*http.Response, error) {
	data := map[string]any{
		"status":  http.StatusOK,
		"success": true,
		"data": map[string]any{
			"is_window_open":    true,
			"does_window_exist": true,
			"window_type":       "automatic",
			"timezone":          "UTC",
			"linux": map[string]any{
				"linuxaid_tag": "11.0.0",
				"needs_reboot": true,
			},
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
	_, responseBody, err := httpx.ParseResponse(resp)
	if err != nil {
		return nil, err
	}

	return api.GetServiceWindowDetails(responseBody)
}

func (*MockObmondoClient) CloseServiceWindow(string, string, string, string) error {
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
