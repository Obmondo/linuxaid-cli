package helper

import (
	"bytes"
	"crypto/tls"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"gitea.obmondo.com/go-scripts/constants"

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

// FetchURL calls an Obmondo API URL
func FetchURL(url string) (*http.Response, error) {
	puppetCert := script.IfExists(os.Getenv(constants.PuppetCertEnv))
	puppetPrivKey := script.IfExists(os.Getenv(constants.PuppetPrivKeyEnv))

	if puppetCert.ExitStatus() != 0 || puppetPrivKey.ExitStatus() != 0 {
		slog.Error("puppet host cert or puppet private key is not present on the node")
		os.Exit(1)
	}

	cert, err := tls.LoadX509KeyPair(os.Getenv(constants.PuppetCertEnv), os.Getenv(constants.PuppetPrivKeyEnv))
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
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
		slog.Error("unexpected error received", slog.String("error", err.Error()))
		os.Exit(1)
	}
	return response, nil
}
