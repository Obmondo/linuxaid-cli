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

	"gitea.obmondo.com/EnableIT/linuxaid-cli/constant"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/helper"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/pkg/prettyfmt"
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
	CloseServiceWindow(windowType, certname string, timezone string) error
	VerifyInstallToken(input *InstallScriptInput) error
	NotifyInstallScriptFailure(input *InstallScriptInput) error
	ServerPing() error
	UpdatePuppetLastRunReport() error
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
		err := errors.New("invalid token")
		slog.Error(scriptFailureLogErrorMessage, slog.Any("error", err))
		return err
	case http.StatusNotAcceptable:
		err := errors.New("invalid token or certname")
		slog.Error(scriptFailureLogErrorMessage, slog.Any("error", err))
		return err
	case http.StatusOK:
		return nil
	case http.StatusBadRequest:
		apiResponse := &ObmondoAPIResponse[string]{}
		if err := json.NewDecoder(resp.Body).Decode(apiResponse); err != nil {
			slog.Error("failed to decode api response", slog.Any("error", err))
			return err
		}
		prettyfmt.PrettyPrintln(prettyfmt.FontRed(fmt.Sprintf("error: %s, resolution: %s", apiResponse.ErrorText, apiResponse.Resolution)))
		return errors.New(apiResponse.ErrorText)
	default:
		err := errors.New(scriptFailureLogErrorMessage)
		slog.Error(err.Error(), slog.Int("http_status", resp.StatusCode))
		return err
	}
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
		const failureLogMsg = "api returned non-204 response"
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			slog.Error(fmt.Sprintf("%s, failed to parse JSON body", failureLogMsg),
				slog.Int("status_code", resp.StatusCode),
				slog.Any("error", err),
			)

			return err
		}
		err = errors.New("failed to inform about latest puppet run status")
		slog.Error(fmt.Sprintf("%s while informing about last puppet run status", failureLogMsg), slog.Int("status_code", resp.StatusCode), slog.String("api_response", string(body)))
		return err
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

	const scriptFailureLogErrorMessage = "failed to notify about script failure to obmondo"
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		err := errors.New("invalid token")
		slog.Error(scriptFailureLogErrorMessage, slog.Any("error", err))
		return err

	case http.StatusNotAcceptable:
		err := errors.New("invalid token or certname")
		slog.Error(scriptFailureLogErrorMessage, slog.Any("error", err))
		return err

	case http.StatusNoContent:
		fmt.Printf("\nInstallation setup failed, please contact ops@obmondo.com\nDon't worry, obmondo has the failed logs to analyze it.\n") //nolint:revive,forbidigo
		return nil
	case http.StatusOK:
		return nil
	default:
		err := errors.New(scriptFailureLogErrorMessage)
		slog.Error(err.Error(), slog.Int("http_status", resp.StatusCode))
		return err
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
	statusCode, responseBody, err := helper.ParseResponse(resp)
	if err != nil {
		slog.Error("unexpected error reading response body", slog.String("error", err.Error()))
		return nil, err
	}

	if statusCode != http.StatusOK {
		slog.Error("unexpected", slog.Int("status_code", statusCode), slog.String("response", string(responseBody)))
		return nil, fmt.Errorf("unexpected non-200 HTTP status code received: %d", statusCode)
	}

	serviceWindow, err := GetServiceWindowDetails(responseBody)
	if err != nil {
		slog.Error("unable to determine the service window", slog.String("error", err.Error()))
		return nil, err
	}

	return serviceWindow, nil
}

func (c *obmondoClient) CloseServiceWindow(windowType, certname string, timezone string) error {
	customerID := helper.GetCustomerID(certname)
	location, err := time.LoadLocation(timezone)
	if err != nil {
		slog.Error("failed to get timezone of provided location", slog.Any("error", err), slog.String("location", timezone))
		return err
	}
	yearMonthDay := time.Now().In(location).Format(time.DateOnly)
	closeWindowURL := fmt.Sprintf("%s/window/close/customer/%s/certname/%s/date/%s/type/%s", c.apiURL, customerID, certname, yearMonthDay, windowType)
	data := []byte(`{"comments": "server has been updated"}`)

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
		bodyBytes, err := io.ReadAll(closeWindow.Body)
		if err != nil {
			slog.Error("failed to read response body", slog.String("error", err.Error()))
			return err
		}

		// Log the response status code and body
		slog.Error("closing service window failed", slog.Int("status_code", closeWindow.StatusCode), slog.String("response", string(bodyBytes)))
		return fmt.Errorf("incorrect response code received from API: %d", closeWindow.StatusCode)
	}
}

// ------------------------------------------------
// ------------------------------------------------

func NewObmondoClient(obmondoAPIURL string, notifyInstallScriptFailure bool) ObmondoClient {
	certname := helper.GetCertname()

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
