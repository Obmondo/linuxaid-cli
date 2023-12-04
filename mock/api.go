package mock

import (
	"bytes"
	"encoding/json"
	"go-scripts/pkg/obmondo_api"
	"io"
	"net/http"
)

type MockObmondoClient struct {
}

func (*MockObmondoClient) FetchServiceWindowStatus() (*http.Response, error) {
	data := map[string]interface{}{
		"status":     200,
		"success":    true,
		"data":       "yes",
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

func (*MockObmondoClient) CloseServiceWindow() (*http.Response, error) {
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
