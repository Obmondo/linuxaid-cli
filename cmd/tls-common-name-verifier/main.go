package main

import (
	"fmt"
	"log"

	"checkendpointsreachable"
)

func main() {
	configFilename := "config.yaml"
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

	fmt.Println(yamlOutput)
}
