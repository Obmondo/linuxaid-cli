package api

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/certs"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/constant"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/httpx"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/prettyfmt"
	"gopkg.in/yaml.v3"
)

const (
	obmondoProdAPIURL = "https://api.obmondo.com/api"
	obmondoBetaAPIURL = "https://api-beta.obmondo.com/api"
	apiTimeOut        = 15
)

type ObmondoClient interface {
	GetServiceWindowStatus() (*ServiceWindow, error)
	FetchServiceWindowStatus() (*http.Response, error)
	CloseServiceWindow(windowType, certname, timezone, comment string) error
	GetCustomerSettings(customerID string) (*CustomerSettings, error)
	VerifyInstallToken(input *InstallScriptInput) error
	NotifyInstallScriptFailure(input *InstallScriptInput) error
	ServerPing() error
	UpdatePuppetLastRunReport() error
	GetServerEnvironment(certname string) (string, error)
}

type obmondoClient struct {
	apiURL                     string
	notifyInstallScriptFailure bool
	certPath                   string
	keyPath                    string
}

func (c *obmondoClient) VerifyInstallToken(input *InstallScriptInput) error {
	url := fmt.Sprintf("%s/servers/install-script/verify/certname/%s?token=%s", c.apiURL, input.Certname, url.QueryEscape(input.Token))
	client := &http.Client{}

	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		slog.Error("failed to create request for validating install token", slog.Any("error", err), slog.String("url", url))
		return err
	}
	resp, err := client.Do(request)
	if err != nil {
		slog.Error("error occurred while requesting client to validate install token", slog.Any("error", err), slog.String("url", url))
		return err
	}
	defer func() {
		if resp.Body != nil {
			if err := resp.Body.Close(); err != nil {
				slog.Error("failed to close body", slog.Any("error", err))
			}
		}
	}()

	const scriptFailureLogErrorMessage = "failed to validate install token"
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return decodeAPIError(resp.Body, resp.StatusCode, "invalid token")
	case http.StatusNotAcceptable:
		return decodeAPIError(resp.Body, resp.StatusCode, "invalid token or certname")
	case http.StatusOK:
		return nil
	case http.StatusBadRequest:
		return decodeAPIError(resp.Body, resp.StatusCode, scriptFailureLogErrorMessage)
	default:
		err := errors.New(scriptFailureLogErrorMessage)
		slog.Error(err.Error(), slog.Int("http_status", resp.StatusCode))
		return err
	}
}

// decodeAPIError parses the Obmondo API's response envelope out of body and
// renders its error_text/resolution to the operator, returning an error
// built from error_text. Falls back to fallbackMsg (logged with statusCode
// and the raw body) when body is empty or does not decode into a usable
// error_text.
func decodeAPIError(body io.Reader, statusCode int, fallbackMsg string) error {
	raw, err := io.ReadAll(body)
	if err != nil {
		slog.Error(fallbackMsg, slog.Int("http_status", statusCode), slog.Any("error", err))
		return errors.New(fallbackMsg)
	}

	apiResponse := &ObmondoAPIResponse[string]{}
	if err := json.Unmarshal(raw, apiResponse); err != nil || apiResponse.ErrorText == "" {
		slog.Error(fallbackMsg, slog.Int("http_status", statusCode), slog.String("response", string(raw)))
		return errors.New(fallbackMsg)
	}

	prettyfmt.PrettyPrintln(prettyfmt.FontRed(fmt.Sprintf("error: %s, resolution: %s", apiResponse.ErrorText, apiResponse.Resolution)))
	return errors.New(apiResponse.ErrorText)
}

func (c *obmondoClient) UpdatePuppetLastRunReport() error {
	url := fmt.Sprintf("%s/servers/puppet_last_run_report", c.apiURL)
	data, err := c.readPuppetLastRunReport()
	if err != nil {
		return err
	}

	resp, err := c.apiCallWithTransport(url, data, http.MethodPut)
	defer func() {
		if resp != nil && resp.Body != nil {
			if cerr := resp.Body.Close(); cerr != nil {
				slog.Error("failed to close body", slog.Any("error", cerr))
			}
		}
	}()
	if err != nil {
		slog.Error("error occurred while trying to inform obmondo about puppet run",
			slog.Any("error", err), slog.String("url", url))
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return decodeAPIError(resp.Body, resp.StatusCode, "failed to inform about latest puppet run status")
	}

	return nil
}

func (*obmondoClient) readPuppetLastRunReport() ([]byte, error) {
	const lastRunStatus = "/opt/puppetlabs/puppet/cache/state/last_run_report.yaml"

	var puppetLastRunReport PuppetLastRunReport
	puppetLastRunReport.IsLastRunYamlFileNotPresent = true

	if _, err := os.Stat(lastRunStatus); err == nil {
		data, err := os.ReadFile(lastRunStatus)
		if err != nil {
			slog.Error("failed to read last run report", slog.Any("error", err))
			return nil, err
		}
		if err := yaml.Unmarshal(data, &puppetLastRunReport); err != nil {
			slog.Error("failed to unmarshal last run report yaml", slog.Any("error", err))
			return nil, err
		}
		puppetLastRunReport.IsLastRunYamlFileNotPresent = false
	} else if !os.IsNotExist(err) {
		slog.Error("failed to stat last run report", slog.Any("error", err))
		return nil, err
	}

	data, err := json.Marshal(&puppetLastRunReport)
	if err != nil {
		slog.Error("failed to marshal last run report into json", slog.Any("error", err))
		return nil, err
	}

	return data, nil
}

func (c *obmondoClient) ServerPing() error {

	url := fmt.Sprintf("%s/servers/ping", c.apiURL)

	resp, err := c.apiCallWithTransport(url, nil, http.MethodPut)
	defer func() {
		if resp != nil && resp.Body != nil {
			if cerr := resp.Body.Close(); cerr != nil {
				slog.Error("failed to close body", slog.Any("error", cerr))
			}
		}
	}()
	if err != nil {
		slog.Error("error occurred while trying to inform obmondo about puppet run",
			slog.Any("error", err), slog.String("url", url))
		return err
	}

	return nil
}

func (c *obmondoClient) NotifyInstallScriptFailure(input *InstallScriptInput) error {
	if !c.notifyInstallScriptFailure {
		return nil
	}
	url := fmt.Sprintf("%s/servers/install-script-failure/certname/%s?token=%s", c.apiURL, input.Certname, url.QueryEscape(input.Token))
	client := &http.Client{}

	request, err := http.NewRequest(http.MethodPut, url, nil)
	if err != nil {
		slog.Error("failed to create request for notifying script failure", slog.Any("error", err), slog.String("url", url))
		return err
	}
	resp, err := client.Do(request)
	if err != nil {
		slog.Error("error occurred after notifying script failure", slog.Any("error", err), slog.String("url", url))
		return err
	}
	defer func() {
		if resp.Body != nil {
			if err := resp.Body.Close(); err != nil {
				slog.Error("failed to close body", slog.Any("error", err))
			}
		}
	}()

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return decodeAPIError(resp.Body, resp.StatusCode, "invalid token")

	case http.StatusNotAcceptable:
		return decodeAPIError(resp.Body, resp.StatusCode, "invalid token or certname")

	case http.StatusNoContent:
		fmt.Printf("\nInstallation setup failed, please contact ops@obmondo.com\nDon't worry, obmondo has the failed logs to analyze it.\n") //nolint:revive,forbidigo
		return nil
	case http.StatusOK:
		return nil
	default:
		return decodeAPIError(resp.Body, resp.StatusCode, "failed to notify about script failure to obmondo")
	}
}

func (c *obmondoClient) getCustomHTTPTransportWithPuppetCerts() (*http.Transport, error) {
	cert, err := tls.LoadX509KeyPair(c.certPath, c.keyPath)
	if err != nil {
		slog.Error("failed to load certifcate", slog.Any("error", err), slog.String("cert", c.certPath), slog.String("cert", c.keyPath))
		return nil, err
	}
	t := &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
		},
	}
	return t, nil
}

func (c *obmondoClient) apiCallWithTransport(url string, data []byte, requestType string) (*http.Response, error) {
	t, err := c.getCustomHTTPTransportWithPuppetCerts()
	if err != nil {
		slog.Error("failed to load host cert & key pair", slog.String("error", err.Error()))
		return nil, err
	}
	var body io.Reader = http.NoBody
	if data != nil {
		body = bytes.NewBuffer(data)
	}

	httpClient := http.Client{Transport: t, Timeout: apiTimeOut * time.Second}

	request, err := http.NewRequest(requestType, url, body)
	if err != nil {
		slog.Error("failed to create API request to obmondo")
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := httpClient.Do(request)
	if err != nil {
		slog.Error("failed to make API request to obmondo")
		return nil, err
	}
	return response, nil
}

func (c *obmondoClient) FetchServiceWindowStatus() (*http.Response, error) {
	serviceWindowURL := fmt.Sprintf("%s/window/now", c.apiURL)
	return c.apiCallWithTransport(serviceWindowURL, nil, http.MethodGet)
}

// ------------------------------------------------
// ------------------------------------------------

func GetServiceWindowDetails(response []byte) (*ServiceWindow, error) {
	type ServiceWindowResponse struct {
		Data ServiceWindow `json:"data"`
	}

	var serviceWindowResponse ServiceWindowResponse

	if err := json.Unmarshal(response, &serviceWindowResponse); err != nil {
		slog.Error("failed to parse service window JSON", slog.String("error", err.Error()))
		return nil, err
	}

	return &serviceWindowResponse.Data, nil
}

func (c *obmondoClient) GetServiceWindowStatus() (*ServiceWindow, error) {
	resp, err := c.FetchServiceWindowStatus()
	if err != nil {
		slog.Error("unexpected error fetching service window url", slog.String("error", err.Error()))
		return nil, err
	}

	defer resp.Body.Close()
	statusCode, responseBody, err := httpx.ParseResponse(resp)
	if err != nil {
		slog.Error("unexpected error reading response body", slog.String("error", err.Error()))
		return nil, err
	}

	if statusCode != http.StatusOK {
		return nil, decodeAPIError(bytes.NewReader(responseBody), statusCode, fmt.Sprintf("unexpected non-200 HTTP status code received: %d", statusCode))
	}

	serviceWindow, err := GetServiceWindowDetails(responseBody)
	if err != nil {
		slog.Error("unable to determine the service window", slog.String("error", err.Error()))
		return nil, err
	}

	return serviceWindow, nil
}

func (c *obmondoClient) CloseServiceWindow(windowType, certname, timezone, comment string) error {
	customerID := certs.GetCustomerID(certname)
	location, err := time.LoadLocation(timezone)
	if err != nil {
		slog.Error("failed to get timezone of provided location", slog.Any("error", err), slog.String("location", timezone))
		return err
	}
	yearMonthDay := time.Now().In(location).Format(time.DateOnly)
	closeWindowURL := fmt.Sprintf("%s/window/close/customer/%s/certname/%s/date/%s/type/%s", c.apiURL, customerID, certname, yearMonthDay, windowType)
	data := []byte(fmt.Sprintf(`{"comments": %q}`, comment))

	closeWindow, err := c.apiCallWithTransport(closeWindowURL, data, http.MethodPut)
	if err != nil {
		slog.Error("closing service window failed", slog.String("error", err.Error()))
		return err
	}
	defer closeWindow.Body.Close()

	switch closeWindow.StatusCode {
	// 202 -> When a certname says it's done but the overall window is not auto-closed
	// 204 -> When a certname says it's done AND the overall window is auto-closed
	// 208 -> When any of the above requests happen again and again
	case http.StatusAccepted, http.StatusNoContent, http.StatusAlreadyReported:
		return nil
	default:
		return decodeAPIError(closeWindow.Body, closeWindow.StatusCode, fmt.Sprintf("incorrect response code received from API: %d", closeWindow.StatusCode))
	}
}

// ------------------------------------------------
// ------------------------------------------------

func (c *obmondoClient) GetCustomerSettings(customerID string) (*CustomerSettings, error) {
	settingsURL := fmt.Sprintf("%s/customer/settings/%s", c.apiURL, customerID)

	resp, err := c.apiCallWithTransport(settingsURL, nil, http.MethodGet)
	defer func() {
		if resp != nil && resp.Body != nil {
			if cerr := resp.Body.Close(); cerr != nil {
				slog.Error("failed to close body", slog.Any("error", cerr))
			}
		}
	}()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch customer settings: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, decodeAPIError(resp.Body, resp.StatusCode, fmt.Sprintf("unexpected status code fetching customer settings: %d", resp.StatusCode))
	}

	var apiResp ObmondoAPIResponse[CustomerSettings]
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode customer settings: %w", err)
	}

	return &apiResp.Data, nil
}

// GetServerEnvironment asks the API which puppet environment this node should run with. The
// endpoint is authenticated by the node's client certificate and always resolves an answer, so an
// empty environment means the API could not decide and the caller has to fall back.
func (c *obmondoClient) GetServerEnvironment(certname string) (string, error) {
	environmentURL := fmt.Sprintf("%s/servers/environment/certname/%s", c.apiURL, certname)

	resp, err := c.apiCallWithTransport(environmentURL, nil, http.MethodGet)
	defer func() {
		if resp != nil && resp.Body != nil {
			if cerr := resp.Body.Close(); cerr != nil {
				slog.Error("failed to close body", slog.Any("error", cerr))
			}
		}
	}()
	if err != nil {
		return "", fmt.Errorf("failed to fetch the puppet environment: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code fetching the puppet environment: %d", resp.StatusCode)
	}

	var apiResp ObmondoAPIResponse[ServerEnvironment]
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", fmt.Errorf("failed to decode the puppet environment: %w", err)
	}

	return apiResp.Data.Environment, nil
}

func NewObmondoClient(obmondoAPIURL string, notifyInstallScriptFailure bool) ObmondoClient {
	certname := certs.GetCertname()

	return &obmondoClient{
		apiURL:                     obmondoAPIURL,
		notifyInstallScriptFailure: notifyInstallScriptFailure,
		certPath:                   fmt.Sprintf("/etc/puppetlabs/puppet/ssl/certs/%s.pem", certname),
		keyPath:                    fmt.Sprintf("%s/%s.pem", constant.PuppetPrivKeyPath, certname),
	}
}

func GetObmondoURL() string {
	obmondoAPIURL := obmondoProdAPIURL
	if os.Getenv(constant.ObmondoEnv) == "1" {
		obmondoAPIURL = obmondoBetaAPIURL
	}

	return obmondoAPIURL
}
