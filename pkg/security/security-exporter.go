package security

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

type securityExporter struct {
	httpClient *http.Client
	hostURL    string
}

const (
	totalNumberOfPackageUpdatesPath = "/total_number_of_packages_with_update"
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
}

func NewSecurityExporter(hostURL string) SecurityExporter {
	return &securityExporter{
		httpClient: &http.Client{},
		hostURL:    hostURL,
	}
}
