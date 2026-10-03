// FILE: d:/hacks/BitNBuilds/sim/main.go
// PURPOSE: Main entry point for the simulation microservice
// INPUTS / OUTPUTS: Starts HTTP control server on port 8090
// DEPENDS ON: sim/internal/control
// USED BY: Docker compose (sim container), CLI
// RULES: Provide clear logging and environment variable fallbacks
// DO NOT: N/A

package main

import (
	"fmt"
	"log"
	"os"

	"fairdrop/sim/internal/control"
)

func main() {
	apiURL := os.Getenv("API_URL")
	if apiURL == "" {
		apiURL = "http://api:8080"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}

	addr := ":" + port
	fmt.Printf("Simulator service listening on %s (target API: %s)\n", addr, apiURL)

	server := control.NewServer(apiURL)
	if err := server.Start(addr); err != nil {
		log.Fatalf("Simulator server failed: %v", err)
	}
}
