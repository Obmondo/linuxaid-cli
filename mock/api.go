package mock

import (
	"bytes"
	"encoding/json"
	api "go-scripts/pkg/obmondo"
	"io"
	"net/http"
)

type MockObmondoClient struct{}

func (*MockObmondoClient) FetchServiceWindowStatus() (*http.Response, error) {
	data := map[string]interface{}{
		"status":  200,
		"success": true,
		"data": map[string]interface{}{
			"is_window_open": true,
			"window_type":    "automatic",
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

func (*MockObmondoClient) CloseServiceWindow(_ string) (*http.Response, error) {
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
