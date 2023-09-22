package api

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"go-scripts/contants"
	"go-scripts/util"

	"github.com/bitfield/script"
)

type Client struct {
}

type ObmondoClient interface {
	FetchServiceWindowStatus() (*http.Response, error)
	CloseServiceWindow() (*http.Response, error)
}

const (
	obmondoAPIURL = constants.OBMONDO_API_URL
)

func fetchURL(url string, data []byte, requestType string) (*http.Response, error) {
	puppetCert := script.IfExists(os.Getenv("PUPPETCERT"))
	puppetPrivKey := script.IfExists(os.Getenv("PUPPETPRIVKEY"))

	if puppetCert.ExitStatus() == 0 || puppetPrivKey.ExitStatus() == 0 {
		log.Println("puppet host cert or puppet private key is not present on the node")
	}

	cert, err := tls.LoadX509KeyPair(os.Getenv("PUPPETCERT"), os.Getenv("PUPPETPRIVKEY"))
	if err != nil {
		log.Println("Failed to load host cert & key pair")
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

	httpClient := http.Client{Transport: t, Timeout: 15 * time.Second}

	request, err := http.NewRequest(requestType, url, body)
	if err != nil {
		log.Println("Failed to create api request to obmonod")
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := httpClient.Do(request)
	if err != nil {
		log.Println("Failed to make api request to obmonod")
		return nil, err
	}
	return response, nil
}

func (c *Client) FetchServiceWindowStatus() (*http.Response, error) {
	serviceWindowURl := fmt.Sprintf("%s/window/now", obmondoAPIURL)
	return fetchURL(serviceWindowURl, nil, http.MethodGet)
}

func (c *Client) CloseServiceWindow() (*http.Response, error) {
	hostCert := script.IfExists(os.Getenv("PUPPETCERT"))
	if hostCert.ExitStatus() != 0 {
		log.Println("puppet host cert is not present on the node")
	}
	certname := util.GetCommonNameFromCertFile(os.Getenv("PUPPETCERT"))
	customerID := util.GetCustomerId(certname)
	yearMonthDay := time.Now().Format("2006-01-02")
	closeWindowURL := fmt.Sprintf("%s/window/close/customer/%s/certname/%s/date/%s", obmondoAPIURL, customerID, certname, yearMonthDay)
	data := []byte(`{"comments": "server has been updated"}`)

	return fetchURL(closeWindowURL, data, http.MethodPut)
}

func NewObmondoClient() ObmondoClient {
	return &Client{}
}