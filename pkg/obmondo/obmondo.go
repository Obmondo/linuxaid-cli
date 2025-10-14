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

	"gitea.obmondo.com/EnableIT/go-scripts/constant"
	"gitea.obmondo.com/EnableIT/go-scripts/helper"
	"gopkg.in/yaml.v3"
)

const (
	apiTimeOut = 15
)

type Client struct {
	notifyInstallScriptFailure bool
	certPath                   string
	keyPath                    string
}

func (c *Client) UpdatePuppetLastRunReport() error {
	url := fmt.Sprintf("%s/servers/puppet_last_run_report", constant.ObmondoAPIURL)
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

func (*Client) readPuppetLastRunReport() ([]byte, error) {
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

func (c *Client) ServerPing() error {

	url := fmt.Sprintf("%s/servers/ping", constant.ObmondoAPIURL)

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

func (c *Client) NotifyInstallScriptFailure(input *InstallScriptFailureInput) error {
	if !c.notifyInstallScriptFailure {
		return nil
	}
	url := fmt.Sprintf("%s/servers/install-script-failure/certname/%s?token=%s", constant.ObmondoAPIURL, input.Certname, url.QueryEscape(os.Getenv("INSTALL_TOKEN")))
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

	default:
		err := errors.New(scriptFailureLogErrorMessage)
		slog.Error(err.Error())
		return err
	}

}

type ObmondoClient interface {
	FetchServiceWindowStatus() (*http.Response, error)
	CloseServiceWindow(windowType string, timezone string) (*http.Response, error)
	NotifyInstallScriptFailure(input *InstallScriptFailureInput) error
	ServerPing() error
	UpdatePuppetLastRunReport() error
}

func (c *Client) getCustomHTTPTransportWithPuppetCerts() (*http.Transport, error) {
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

func (c *Client) apiCallWithTransport(url string, data []byte, requestType string) (*http.Response, error) {
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

func (c *Client) FetchServiceWindowStatus() (*http.Response, error) {
	serviceWindowURL := fmt.Sprintf("%s/window/now", constant.ObmondoAPIURL)
	return c.apiCallWithTransport(serviceWindowURL, nil, http.MethodGet)
}

func (c *Client) CloseServiceWindow(windowType string, timezone string) (*http.Response, error) {
	certname := helper.GetCertname()
	customerID := helper.GetCustomerID()
	location, err := time.LoadLocation(timezone)
	if err != nil {
		slog.Error("failed to get timezone of provided location", slog.Any("error", err), slog.String("location", timezone))
		return nil, err
	}
	yearMonthDay := time.Now().In(location).Format("2006-01-02")
	closeWindowURL := fmt.Sprintf("%s/window/close/customer/%s/certname/%s/date/%s/type/%s", constant.ObmondoAPIURL, customerID, certname, yearMonthDay, windowType)
	data := []byte(`{"comments": "server has been updated"}`)

	return c.apiCallWithTransport(closeWindowURL, data, http.MethodPut)
}

func NewObmondoClient(notifyInstallScriptFailure bool) ObmondoClient {
	return &Client{
		notifyInstallScriptFailure: notifyInstallScriptFailure,
		certPath:                   fmt.Sprintf("/etc/puppetlabs/puppet/ssl/certs/%s.pem", helper.GetCertname()),
		keyPath:                    fmt.Sprintf("%s/%s.pem", constant.PuppetPrivKeyPath, helper.GetCertname()),
	}
}
