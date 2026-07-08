package checkconnectivity

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/tevino/tcp-shaker"
)

const (
	port        = "443"
	timeout     = time.Second * 5
	metricsFile = "/var/lib/node_exporter/obmondo_domains_reachable.prom"

	apiHost = "api.obmondo.com"

	// skipMetricsEnv, when set, skips writing the node_exporter textfile metric
	// (used to disable it in environments like Kubernetes that lack it).
	skipMetricsEnv = "OBMONDO_SKIP_CONNECTIVITY_METRICS"
)

var runPuppetMetric *prometheus.GaugeVec
var registry *prometheus.Registry

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

func CheckTCPConnection(prometheusHost, puppetServerHost string) bool {
	hosts := []string{apiHost, prometheusHost, puppetServerHost}

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

	// The .prom metric feeds the fleet's node_exporter textfile collector; skip it
	// where that isn't present (e.g. Kubernetes) so a failed write can't abort the run.
	if os.Getenv(skipMetricsEnv) == "" {
		if err := prometheus.WriteToTextfile(metricsFile, registry); err != nil {
			slog.Info("Error writing metrics to file:", slog.String("error", err.Error()))
			return false
		}
	}

	return allAPIReachable
}
