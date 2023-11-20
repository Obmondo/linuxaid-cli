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

	constants "go-scripts/constants"
	"go-scripts/util"

	"github.com/bitfield/script"
)

const (
	apiTimeOut = 15
)

type Client struct {
}

type ObmondoClient interface {
	FetchServiceWindowStatus() (*http.Response, error)
	CloseServiceWindow() (*http.Response, error)
}

const (
	obmondoAPIURL = constants.ObmondoAPIURL
)

func fetchURL(url string, data []byte, requestType string) (*http.Response, error) {
	puppetCert := script.IfExists(os.Getenv("PUPPETCERT"))
	puppetPrivKey := script.IfExists(os.Getenv("PUPPETPRIVKEY"))

	if puppetCert.ExitStatus() != 0 || puppetPrivKey.ExitStatus() != 0 {
		log.Fatal("puppet host cert or puppet private key is not present on the node")
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

	httpClient := http.Client{Transport: t, Timeout: apiTimeOut * time.Second}

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

func (*Client) FetchServiceWindowStatus() (*http.Response, error) {
	serviceWindowURL := fmt.Sprintf("%s/window/now", obmondoAPIURL)
	return fetchURL(serviceWindowURL, nil, http.MethodGet)
}

func (*Client) CloseServiceWindow() (*http.Response, error) {
	hostCert := script.IfExists(os.Getenv("PUPPETCERT"))
	if hostCert.ExitStatus() != 0 {
		log.Println("puppet host cert is not present on the node")
	}
	certname := util.GetCommonNameFromCertFile(os.Getenv("PUPPETCERT"))
	customerID := util.GetCustomerID(certname)
	yearMonthDay := time.Now().Format("2006-01-02")
	closeWindowURL := fmt.Sprintf("%s/window/close/customer/%s/certname/%s/date/%s", obmondoAPIURL, customerID, certname, yearMonthDay)
	data := []byte(`{"comments": "server has been updated"}`)

	return fetchURL(closeWindowURL, data, http.MethodPut)
}

func NewObmondoClient() ObmondoClient {
	return &Client{}
}
