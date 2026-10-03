// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/config/config.go
// PURPOSE: Configuration loading and validation
// INPUTS / OUTPUTS: N/A
// DEPENDS ON: N/A
// USED BY: main.go
// RULES: N/A
// DO NOT: N/A
package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port           string
	RedisURL       string
	DBURL          string
	HMACSecret     string
	SIMMode        bool
	TrustThreshold float64
	FPBudget       int
	WindowSeconds  int
	TotalSeats     int
}

func Load() (*Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	simMode := os.Getenv("SIM_MODE") == "true"
	trustThreshold, _ := strconv.ParseFloat(os.Getenv("TRUST_THRESHOLD"), 64)
	fpBudget, _ := strconv.Atoi(os.Getenv("FP_BUDGET"))
	windowSeconds, _ := strconv.Atoi(os.Getenv("WINDOW_SECONDS"))
	totalSeats, _ := strconv.Atoi(os.Getenv("TOTAL_SEATS"))
	if totalSeats == 0 {
		totalSeats = 500
	}

	return &Config{
		Port:           port,
		RedisURL:       os.Getenv("REDIS_URL"),
		DBURL:          os.Getenv("DB_URL"),
		HMACSecret:     os.Getenv("HMAC_SECRET"),
		SIMMode:        simMode,
		TrustThreshold: trustThreshold,
		FPBudget:       fpBudget,
		WindowSeconds:  windowSeconds,
		TotalSeats:     totalSeats,
	}, nil
}
