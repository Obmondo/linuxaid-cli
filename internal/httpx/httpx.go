package httpx

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/constant"
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
	// a plain existence check; it needs no shell
	for _, path := range []string{os.Getenv(constant.PuppetCertEnv), os.Getenv(constant.PuppetPrivKeyEnv)} {
		if _, err := os.Stat(path); err != nil {
			return nil, fmt.Errorf("puppet host cert or puppet private key is not present on the node: %w", err)
		}
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
