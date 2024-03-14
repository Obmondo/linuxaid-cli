package main

import (
	"flag"
	"log"

	"checkendpointsreachable"
)

func main() {
	var configFilename string

	// Define a flag for config file
	flag.StringVar(&configFilename, "config", "/opt/obmondo/etc/tls-common-name-verifier/config.yaml", "Path to the config.yaml file")
	flag.Parse()

	domains, err := checkendpointsreachable.LoadConfig(configFilename)
	if err != nil {
		log.Fatal("Error loading config:", err)
	}

	results, err := checkendpointsreachable.ObmondoEndpointStatus(*domains)
	if err != nil {
		log.Fatal(err)
	}

	yamlOutput, err := checkendpointsreachable.PrintYAML(results)
	if err != nil {
		log.Fatal(err)
	}

	log.Println(yamlOutput)
}
