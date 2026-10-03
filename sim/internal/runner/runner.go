// FILE: d:/hacks/BitNBuilds/sim/internal/runner/runner.go
// PURPOSE: Scenario runner executing concurrent virtual users using FullFlowClient & IPGenerator
// INPUTS / OUTPUTS: Returns final SimulationReport matching SIMULATOR_SPEC.md
// DEPENDS ON: sim/internal/profiles, sim/internal/report, sim/internal/runner/client.go, sim/internal/runner/ipgen.go
// USED BY: sim/internal/control
// RULES: Manage worker goroutines and enforce client flow execution
// DO NOT: Bypass HTTP protocol layers

package runner

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"fairdrop/sim/internal/profiles"
	"fairdrop/sim/internal/report"
)

type Config struct {
	BaseURL     string
	Scenario    string
	Mode        string
	Seed        int64
	Seats       int
	BotsCount   int
	HumansCount int
	Timeout     time.Duration
}

type Runner struct {
	cfg       Config
	ipGen     *IPGenerator
	collector *report.Collector
}

func NewRunner(cfg Config) *Runner {
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}
	if cfg.Seed == 0 {
		cfg.Seed = time.Now().UnixNano()
	}
	if cfg.Seats == 0 {
		cfg.Seats = 500
	}
	return &Runner{
		cfg:       cfg,
		ipGen:     NewIPGenerator(),
		collector: report.NewCollector(),
	}
}

func (r *Runner) RunScenario(ctx context.Context) report.SimulationReport {
	var wg sync.WaitGroup
	httpClient := &http.Client{Timeout: 10 * time.Second}
	ffClient := NewFullFlowClient(r.cfg.BaseURL, httpClient)

	// 1. Spawn Bots
	botProfiles := []string{"bot_naive", "bot_solver", "bot_distributed", "bot_retry"}
	for i := 0; i < r.cfg.BotsCount; i++ {
		wg.Add(1)
		pName := botProfiles[i%len(botProfiles)]
		prof := profiles.DefaultProfiles[pName]
		userID := fmt.Sprintf("bot_%s_%d_%d", pName, i, rand.Intn(10000))
		ip := r.ipGen.GetIP("bot", pName, i)

		go func(p profiles.ProfileConfig, uID, clientIP string) {
			defer wg.Done()
			ffClient.Execute(ctx, p, uID, clientIP, r.collector)
		}(prof, userID, ip)
	}

	// 2. Spawn Humans
	humanProfiles := []string{"human_normal", "human_slow", "human_frustrated", "human_shared_ip"}
	for i := 0; i < r.cfg.HumansCount; i++ {
		wg.Add(1)
		pName := humanProfiles[i%len(humanProfiles)]
		prof := profiles.DefaultProfiles[pName]
		userID := fmt.Sprintf("human_%s_%d_%d", pName, i, rand.Intn(10000))
		ip := r.ipGen.GetIP("human", pName, i)

		go func(p profiles.ProfileConfig, uID, clientIP string) {
			defer wg.Done()
			ffClient.Execute(ctx, p, uID, clientIP, r.collector)
		}(prof, userID, ip)
	}

	wg.Wait()

	rep := r.collector.GenerateReport(r.cfg.Scenario, r.cfg.Mode, r.cfg.Seed, r.cfg.Seats)
	rep.LedgerCrossCheck = report.FetchLedgerCrossCheck(r.cfg.BaseURL, rep.TotalRequests)
	return rep
}
