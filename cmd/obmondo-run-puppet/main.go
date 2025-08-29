package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"gitea.obmondo.com/go-scripts/pkg/checkconnectivity"
	"gitea.obmondo.com/go-scripts/helper"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/bitfield/script"
)

var Version string

const (
	obmondoBasepathAPIURL      = "https://api.obmondo.com/api"
	serverPingAndGetNoopAPIURL = "/servers/ping-and-get-noop"

	puppetCertEnvKey       = "PUPPETCERT"
	puppetPrivateKeyEnvKey = "PUPPETPRIVKEY"
	httpTimeout            = 15
)

// ParseResponse reads a response, returning the status code, body and error that occurred.
func ParseResponse(response *http.Response) (int, []byte, error) {
	defer response.Body.Close()

	bts, err := io.ReadAll(response.Body)
	if err != nil {
		slog.Error("unable to read or parse the response", slog.String("error", err.Error()))
		return response.StatusCode, nil, err
	}

	return response.StatusCode, bts, nil
}

// FetchURL calls an Obmondo API URL
func FetchURL(url string) (*http.Response, error) {
	puppetCertFile, puppetCertExists := os.LookupEnv(puppetCertEnvKey)
	if !puppetCertExists {
		slog.Error("unable to find puppet host cert, or is not present on the node", slog.String("env_key", puppetCertEnvKey))
		return nil, errors.New("unable to find puppet host cert, or is not present on the node")
	}

	slog.Info("successfully found the puppet cert file", slog.String("file_path", puppetCertFile))
	puppetPrivKeyFile, puppetPrivKeyExists := os.LookupEnv(puppetPrivateKeyEnvKey)
	if !puppetPrivKeyExists {
		slog.Error("unable to find puppet private key, or is not present on the node", slog.String("env_key", puppetPrivateKeyEnvKey))
		return nil, errors.New("unable to find puppet private key, or is not present on the node")
	}

	slog.Info("successfully found the puppet private key file", slog.String("file_path", puppetPrivKeyFile))
	cert, err := tls.LoadX509KeyPair(puppetCertFile, puppetPrivKeyFile)
	if err != nil {
		slog.Error("unable to read or parse puppet cert/key", slog.String("error", err.Error()))
		return nil, err
	}

	t := &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
		},
	}
	httpClient := http.Client{Transport: t, Timeout: httpTimeout * time.Second}

	request, err := http.NewRequest(http.MethodPut, url, http.NoBody)
	if err != nil {
		slog.Error("unable to create a new HTTP request", slog.String("error", err.Error()))
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := httpClient.Do(request)
	if err != nil {
		slog.Error("unable to send the HTTP request", slog.String("error", err.Error()))
		return nil, err
	}

	return response, nil
}

// Get puppet state from obmondo api
func getPuppetState() (state bool, err error) {
	slog.Info("fetching puppet state", slog.String("url", obmondoBasepathAPIURL+serverPingAndGetNoopAPIURL))
	// Get a response from api, currently it only returns true (200), 4xx, or 5xx
	response, err := FetchURL(obmondoBasepathAPIURL + serverPingAndGetNoopAPIURL)
	if err != nil {
		slog.Error("unable to fetch response from Obmondo URL", slog.String("error", err.Error()))
		return
	}

	statusCode, responseBody, err := ParseResponse(response)
	if err != nil {
		slog.Error("unable to parse the response from Obmondo URL", slog.String("error", err.Error()))
		return
	}

	slog.Info("successfully got the response", slog.Int("status_code", statusCode), slog.String("body", string(responseBody)))
	if statusCode != http.StatusOK {
		slog.Error("non-200 status code from response received")
		err = errors.New("non-200 status code from response received")
		return
	}

	state, err = strconv.ParseBool(string(responseBody))
	if err != nil {
		slog.Error("unable to parse the response into boolean value", slog.String("error", err.Error()))
		return
	}

	return
}

// Run the puppet agent in noop mode for now
func runPuppet() error {
	// Puppet run execution returns total 5 status codes
	//
	// 0: The run succeeded with no changes or failures; the system was already in the desired state.
	// 1: The run failed, or wasn't attempted due to another run already in progress.
	// 2: The run succeeded, and some resources were changed.
	// 4: The run succeeded, and some resources failed.
	// 6: The run succeeded, and included both changes and failures.
	// [Source: https://www.puppet.com/docs/puppet/7/man/agent.html#usage-notes]
	//
	// We throw error at status code 1, and return.
	// Status codes other than 2 are considered as warning.
	// Status code 0 doesn't count as error, so no need to handle it.

	statusCodeFailed := 1
	statusCodeSucceededWithChanges := 2

	slog.Info("executing the puppet agent command")
	cmdPipe := script.Exec("/opt/puppetlabs/bin/puppet agent -t --noop")
	_, err := cmdPipe.Stdout()
	if err != nil {
		// When encountering status code 1, consider it as an error, and return.
		if cmdPipe.ExitStatus() == statusCodeFailed {
			slog.Error("puppet agent command execution failed", slog.String("status", err.Error()))
			return err
		}

		// When encountering status codes other than 2, just log it as a warning.
		if cmdPipe.ExitStatus() != statusCodeSucceededWithChanges {
			slog.Warn("puppet agent run succeeded, but with failures", slog.String("status", err.Error()))
		}
	}

	slog.Info("completed the puppet agent command execution")
	return nil
}

// Entry point
func main() {
	versionFlag := flag.Bool("version", false, "Print version and exit")

	flag.Parse()

	if *versionFlag {
		slog.Info("run_puppet", "version", Version)
		os.Exit(0)
	}

	helper.LoadPuppetEnv()

	slog.Info("run_puppet", "version", Version)

	allAPIReachable := checkconnectivity.CheckTCPConnection()
	if !allAPIReachable {
		slog.Error("unable to connect to obmondo api, aborting", slog.String("error", "api not accessible"))
		return
	}

	noopStatus, err := getPuppetState()
	if err != nil {
		slog.Error("unable to get the puppet state", slog.String("error", err.Error()))
		return
	}

	// Since we want the state of the puppet agent run on client.
	// So it can be either noop or no-noop
	if noopStatus {
		if err := runPuppet(); err != nil {
			slog.Error("unable to run the puppet agent", slog.String("error", err.Error()))
			return
		}

		return
	}

	// Need to have case here later in future, when we migrate the endpoints in go-api
	if err := runPuppet(); err != nil {
		slog.Error("unable to run the puppet agent", slog.String("error", err.Error()))
		return
	}
}
