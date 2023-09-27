package util

import (
	"bytes"
	"crypto/tls"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/bitfield/script"
)

const (
	fifteen = 15
)

func ParseResponse(response *http.Response) (int, []byte, error) {
	code := response.StatusCode
	defer response.Body.Close()
	bts, err := io.ReadAll(response.Body)
	if err != nil {
		return code, nil, err
	}
	return code, bts, nil
}

// ParseResponse reads a response, returning the status code, body and error that occurred.
// func ParseResponse(response *http.Response) (int, []byte, error) {
// 	code := response.StatusCode
// 		defer response.Body.Close()
// 			bts, err := io.ReadAll(response.Body)
// 				if err != nil {
// 						return code, nil, err
// 							}
// 								return code, bts, nil
// 								}

// FetchURL calls an Obmondo API URL
func FetchURL(url string) (*http.Response, error) {
	puppetCert := script.IfExists(os.Getenv("PUPPETCERT"))
	puppetPrivKey := script.IfExists(os.Getenv("PUPPETPRIVKEY"))

	if puppetCert.ExitStatus() != 0 || puppetPrivKey.ExitStatus() != 0 {
		log.Fatal("puppet host cert or puppet private key is not present on the node")
	}

	cert, err := tls.LoadX509KeyPair(os.Getenv("PUPPETCERT"), os.Getenv("PUPPETPRIVKEY"))
	if err != nil {
		log.Fatal(err)
	}

	t := &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
		},
	}

	httpClient := http.Client{Transport: t, Timeout: fifteen * time.Second}

	request, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer([]byte("{}")))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := httpClient.Do(request)
	if err != nil {
		log.Fatalf("Unexpected error received: %s", err)
	}
	return response, nil
}
