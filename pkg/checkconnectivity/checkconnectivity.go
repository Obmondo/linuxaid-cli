package checkconnectivity

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/tevino/tcp-shaker"
)

const (
	port        = "443"
	timeout     = time.Second * 5
	metricsFile = "/var/lib/node_exporter/obmondo_domains_reachable.prom"

	apiHost = "api.obmondo.com"
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

	// Metrics are best-effort: the node_exporter textfile directory may not
	// exist on this system, and that must not fail the connectivity check.
	if err := prometheus.WriteToTextfile(metricsFile, registry); err != nil {
		slog.Warn("failed to write connectivity metrics", slog.String("error", err.Error()))
	}

	return allAPIReachable
}
