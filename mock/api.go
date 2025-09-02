package mock

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	api "gitea.obmondo.com/EnableIT/go-scripts/pkg/obmondo"
)

type MockObmondoClient struct{}

// NotifyInstallScriptFailure implements api.ObmondoClient.
func (m *MockObmondoClient) NotifyInstallScriptFailure(input *api.InstallScriptFailureInput) error {
	return nil
}

// ServerPing implements api.ObmondoClient.
func (m *MockObmondoClient) ServerPing() error {
	return nil
}

// UpdatePuppetLastRunReport implements api.ObmondoClient.
func (m *MockObmondoClient) UpdatePuppetLastRunReport() error {
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

func (*MockObmondoClient) CloseServiceWindow(_ string, _ string) (*http.Response, error) {
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
