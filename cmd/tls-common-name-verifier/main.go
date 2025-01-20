package main

import (
	"flag"
	"go-scripts/cmd/tls-common-name-verifier/checkendpointsreachable"
	"go-scripts/utils/logger"
	"log/slog"
	"os"
)

func main() {
	var configFilename string
	debug := true
	logger.InitLogger(debug)

	// Define a flag for config file
	flag.StringVar(&configFilename, "config", "/opt/obmondo/etc/tls-common-name-verifier/config.yaml", "Path to the config.yaml file")
	flag.Parse()

	domains, err := checkendpointsreachable.LoadConfig(configFilename)
	if err != nil {
		slog.Error("error loading config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	results, err := checkendpointsreachable.ObmondoEndpointStatus(*domains)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	yamlOutput, err := checkendpointsreachable.PrintYAML(results)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	slog.Info(yamlOutput)
}
