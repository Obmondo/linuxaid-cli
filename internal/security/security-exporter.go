package security

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type securityExporter struct {
	httpClient *http.Client
	hostURL    string
}

const (
	totalNumberOfPackageUpdatesPath = "/total_number_of_packages_with_update"
	scanPath                        = "/scan"
	scanTimeout                     = 6 * time.Minute
)

func (s *securityExporter) GetNumberOfPackageUpdates() (*TotalNumberOfPackagesWithUpdateResponse, error) {
	getNumberOfPackageUpdates := &TotalNumberOfPackagesWithUpdateResponse{}

	resp, err := s.httpClient.Get(fmt.Sprintf("%s%s", s.hostURL, totalNumberOfPackageUpdatesPath))
	if err != nil {
		slog.Debug("failed to get response from exporter's endpoint", slog.Any("error", err))
		return nil, err
	}
	defer func() {
		if resp != nil {
			if err := resp.Body.Close(); err != nil {
				slog.Error("failed to close response body", slog.Any("error", err))
			}
		}
	}()

	if resp.StatusCode >= http.StatusBadRequest {
		cmdErrRsp := &CommandErrResponse{}
		if err := json.NewDecoder(resp.Body).Decode(cmdErrRsp); err != nil {
			slog.Debug("failed to unmarshall response from exporter's endpoint", slog.Any("error", err))
			return nil, err
		}
		slog.Debug("error occurred in security exporter", slog.Any("error", cmdErrRsp.Error), slog.Any("output", cmdErrRsp.CmdOutput))
		return nil, cmdErrRsp.Error
	}

	if err := json.NewDecoder(resp.Body).Decode(getNumberOfPackageUpdates); err != nil {
		slog.Debug("failed to unmarshall response from exporter's endpoint", slog.Any("error", err))
		return nil, err
	}

	return getNumberOfPackageUpdates, nil
}

// nolint: revive
type SecurityExporter interface {
	GetNumberOfPackageUpdates() (*TotalNumberOfPackagesWithUpdateResponse, error)
	TriggerScan() (*ScanResponse, error)
}

func (s *securityExporter) TriggerScan() (*ScanResponse, error) {
	client := &http.Client{Timeout: scanTimeout}

	resp, err := client.Post(fmt.Sprintf("%s%s", s.hostURL, scanPath), "application/json", bytes.NewReader(nil))
	if err != nil {
		slog.Error("failed to trigger scan on security exporter", slog.Any("error", err))
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("failed to close response body", slog.Any("error", err))
		}
	}()

	scanResp := &ScanResponse{}
	if err := json.NewDecoder(resp.Body).Decode(scanResp); err != nil {
		slog.Error("failed to decode scan response", slog.Any("error", err))
		return nil, err
	}

	return scanResp, nil
}

func NewSecurityExporter(hostURL string) SecurityExporter {
	return &securityExporter{
		httpClient: &http.Client{},
		hostURL:    hostURL,
	}
}
