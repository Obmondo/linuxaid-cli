package api

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	constants "go-scripts/constants"
	"go-scripts/utils"
)

const (
	apiTimeOut    = 15
	obmondoAPIURL = constants.ObmondoAPIURL
)

type Client struct {
}

type ObmondoClient interface {
	FetchServiceWindowStatus() (*http.Response, error)
	CloseServiceWindow(windowType string, timezone string) (*http.Response, error)
}

func fetchURL(url string, data []byte, requestType string) (*http.Response, error) {
	cert, err := tls.LoadX509KeyPair(os.Getenv("PUPPETCERT"), os.Getenv("PUPPETPRIVKEY"))
	if err != nil {
		slog.Error("failed to load host cert & key pair")
		return nil, err
	}

	t := &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
		},
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

func (*Client) FetchServiceWindowStatus() (*http.Response, error) {
	serviceWindowURL := fmt.Sprintf("%s/window/now", obmondoAPIURL)
	return fetchURL(serviceWindowURL, nil, http.MethodGet)
}

func (*Client) CloseServiceWindow(windowType string, timezone string) (*http.Response, error) {
	certname := utils.GetCommonNameFromCertFile(os.Getenv("PUPPETCERT"))
	customerID := utils.GetCustomerID(certname)
	location, err := time.LoadLocation(timezone)
	if err != nil {
		slog.Error("failed to get timezone of provided location", slog.Any("error", err), slog.String("location", timezone))
		return nil, err
	}
	yearMonthDay := time.Now().In(location).Format("2006-01-02")
	closeWindowURL := fmt.Sprintf("%s/window/close/customer/%s/certname/%s/date/%s/type/%s", obmondoAPIURL, customerID, certname, yearMonthDay, windowType)
	data := []byte(`{"comments": "server has been updated"}`)

	return fetchURL(closeWindowURL, data, http.MethodPut)
}

func NewObmondoClient() ObmondoClient {
	return &Client{}
}
