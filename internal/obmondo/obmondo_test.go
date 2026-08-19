package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeAPIError(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		statusCode  int
		fallbackMsg string
		wantErr     string
	}{
		{
			name:        "renders error_text and resolution from envelope",
			body:        `{"status":401,"success":false,"data":"","message":"","resolution":"request a new token","error_text":"invalid or expired token"}`,
			statusCode:  http.StatusUnauthorized,
			fallbackMsg: "invalid token",
			wantErr:     "invalid or expired token",
		},
		{
			name:        "falls back when body is empty",
			body:        "",
			statusCode:  http.StatusInternalServerError,
			fallbackMsg: "failed to notify about script failure to obmondo",
			wantErr:     "failed to notify about script failure to obmondo",
		},
		{
			name:        "falls back when body is malformed JSON",
			body:        "not json",
			statusCode:  http.StatusBadGateway,
			fallbackMsg: "failed to notify about script failure to obmondo",
			wantErr:     "failed to notify about script failure to obmondo",
		},
		{
			name:        "falls back when envelope decodes but error_text is empty",
			body:        `{"status":500,"success":false,"data":"","message":"","resolution":"","error_text":""}`,
			statusCode:  http.StatusInternalServerError,
			fallbackMsg: "failed to inform about latest puppet run status",
			wantErr:     "failed to inform about latest puppet run status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := decodeAPIError(strings.NewReader(tt.body), tt.statusCode, tt.fallbackMsg)
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if err.Error() != tt.wantErr {
				t.Errorf("decodeAPIError() error = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, errors.New("boom")
}

func TestDecodeAPIError_BodyReadFailure(t *testing.T) {
	const fallbackMsg = "failed to inform about latest puppet run status"

	err := decodeAPIError(errReader{}, http.StatusInternalServerError, fallbackMsg)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if err.Error() != fallbackMsg {
		t.Errorf("decodeAPIError() error = %q, want %q", err.Error(), fallbackMsg)
	}
}

// TestNotifyInstallScriptFailure_RendersAPIError checks that a status-handling
// method wired to decodeAPIError (here, the 401 branch) surfaces the API's
// own error_text instead of the hardcoded fallback.
func TestNotifyInstallScriptFailure_RendersAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"status":401,"success":false,"data":"","message":"","resolution":"request a new install link","error_text":"install token has expired"}`))
	}))
	defer server.Close()

	client := &obmondoClient{apiURL: server.URL, notifyInstallScriptFailure: true}

	err := client.NotifyInstallScriptFailure(&InstallScriptInput{Certname: "test.example", Token: "expired-token"})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if err.Error() != "install token has expired" {
		t.Errorf("NotifyInstallScriptFailure() error = %q, want %q", err.Error(), "install token has expired")
	}
}

// TestVerifyInstallToken_RendersAPIError checks that VerifyInstallToken's
// 406 branch surfaces the API's own error_text instead of the hardcoded
// "invalid token or certname" fallback. This is the path expiry takes,
// which matters once install-token TTLs shrink and expiry stops being rare.
func TestVerifyInstallToken_RendersAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotAcceptable)
		_, _ = w.Write([]byte(`{"status":406,"success":false,"data":"","message":"","resolution":"please ensure that certname is registered with us and token has not expired","error_text":"token is expired or certname is not registered"}`))
	}))
	defer server.Close()

	client := &obmondoClient{apiURL: server.URL}

	err := client.VerifyInstallToken(&InstallScriptInput{Certname: "test.example", Token: "expired-token"})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if err.Error() != "token is expired or certname is not registered" {
		t.Errorf("VerifyInstallToken() error = %q, want %q", err.Error(), "token is expired or certname is not registered")
	}
}

func TestServiceWindowPuppetEnvironment(t *testing.T) {
	tests := []struct {
		name          string
		serviceWindow *ServiceWindow
		expected      string
	}{
		{
			name: "an automatic window resolves the pinned tag to an environment",
			serviceWindow: &ServiceWindow{
				WindowType: "automatic",
				Linux:      &LinuxWindowDetails{LinuxAidTag: "v11.0.0"},
			},
			expected: "v11_0_0",
		},
		{
			name: "a tag that is already a valid environment is kept as is",
			serviceWindow: &ServiceWindow{
				WindowType: "automatic",
				Linux:      &LinuxWindowDetails{LinuxAidTag: "v11_0_0"},
			},
			expected: "v11_0_0",
		},
		{
			name: "an adhoc window pins no environment",
			serviceWindow: &ServiceWindow{
				WindowType: "adhoc",
				Linux:      &LinuxWindowDetails{LinuxAidTag: "v11.0.0"},
			},
			expected: "",
		},
		{
			name:          "a window without linux details pins no environment",
			serviceWindow: &ServiceWindow{WindowType: "automatic"},
			expected:      "",
		},
		{
			name: "a window without a tag pins no environment",
			serviceWindow: &ServiceWindow{
				WindowType: "automatic",
				Linux:      &LinuxWindowDetails{},
			},
			expected: "",
		},
		{
			name:          "no window pins no environment",
			serviceWindow: nil,
			expected:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if environment := tt.serviceWindow.PuppetEnvironment(); environment != tt.expected {
				t.Errorf("PuppetEnvironment() = %q, want %q", environment, tt.expected)
			}
		})
	}
}
