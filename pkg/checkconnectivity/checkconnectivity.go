package checkconnectivity

import (
	"context"
	"errors"
	"fmt"
	"go-scripts/util"
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/tevino/tcp-shaker"
)

const (
	puppetSuffix = ".puppet.obmondo.com"
	port         = "443"
	timeout      = time.Second * 5
	metricsFile  = "/var/lib/node_exporter/obmondo_domains_reachable.prom"
)

var enableitHosts = []string{
	"api.obmondo.com",
	"prometheus.obmondo.com",
}

var runPuppetMetric *prometheus.GaugeVec
var registry *prometheus.Registry

func getHostList() ([]string, error) {
	customerID := util.GetCustomerIDFromEnv()
	if len(customerID) == 0 {
		return nil, errors.New("customerID not found")
	}

	return append(enableitHosts, customerID+puppetSuffix), nil
}

func init() {
	registry = prometheus.NewRegistry()

	runPuppetMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "runPuppet_connectivity",
			Help: "Api connectivity status",
		},
		[]string{"host", "port"},
	)

	registry.MustRegister(runPuppetMetric)
}

func CheckTCPConnection() bool {
	// Initializing the checker
	// It is expected to be shared among goroutines, only one instance is necessary.
	c := tcp.NewChecker()

	ctx, stopChecker := context.WithCancel(context.Background())
	defer stopChecker()
	go func() {
		if err := c.CheckingLoop(ctx); err != nil {
			slog.Info("checking loop stopped due to fatal error: ", slog.String("error", err.Error()))
		}
	}()

	<-c.WaitReady()

	hosts, err := getHostList()

	if err != nil {
		slog.Info("Error resolving ip ", slog.String("error", err.Error()))
		return false
	}

	allAPIReachable := true

	for _, host := range hosts {
		err := c.CheckAddr(fmt.Sprintf("%s:%s", host, port), timeout)
		if err != nil {
			allAPIReachable = false
			runPuppetMetric.WithLabelValues(host, port).Set(1)
			continue
		}
		runPuppetMetric.WithLabelValues(host, port).Set(0)
	}

	if err := prometheus.WriteToTextfile(metricsFile, registry); err != nil {
		slog.Info("Error writing metrics to file:", slog.String("error", err.Error()))
		return false
	}

	return allAPIReachable
}
