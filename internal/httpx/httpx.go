package httpx

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/constant"

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
	puppetCert := script.IfExists(os.Getenv(constant.PuppetCertEnv))
	puppetPrivKey := script.IfExists(os.Getenv(constant.PuppetPrivKeyEnv))

	if puppetCert.ExitStatus() != 0 || puppetPrivKey.ExitStatus() != 0 {
		return nil, errors.New("puppet host cert or puppet private key is not present on the node")
	}

	cert, err := tls.LoadX509KeyPair(os.Getenv(constant.PuppetCertEnv), os.Getenv(constant.PuppetPrivKeyEnv))
	if err != nil {
		return nil, fmt.Errorf("could not load the puppet key pair: %w", err)
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
		return nil, fmt.Errorf("request to %s failed: %w", url, err)
	}

	return response, nil
}
