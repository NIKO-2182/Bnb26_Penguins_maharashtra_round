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

	// Seed a deterministic RNG so an identical seed reproduces identical user
	// ids and therefore an identical ledger. Previously bare rand.Intn was used,
	// so runs varied despite a fixed seed.
	rng := rand.New(rand.NewSource(r.cfg.Seed))

	// 1. Spawn HUMANS FIRST.
	//
	// Ordering was a fairness bug, not a cosmetic one: bots were spawned first
	// and there is no arrival window, so bots drained the entire pool before a
	// single human reached /claim. Human profiles then reported
	// acc=0 rej=0 nodec=<all> -- they were never even judged. Humans enter first
	// so the classifier actually has to discriminate rather than the pool simply
	// being gone.
	humanProfiles := []string{"human_normal", "human_slow", "human_frustrated", "human_shared_ip"}
	for i := 0; i < r.cfg.HumansCount; i++ {
		wg.Add(1)
		pName := humanProfiles[i%len(humanProfiles)]
		prof := profiles.DefaultProfiles[pName]
		userID := fmt.Sprintf("human_%s_%d_%d", pName, i, rng.Intn(10000))
		ip := r.ipGen.GetIP("human", pName, i)

		go func(p profiles.ProfileConfig, uID, clientIP string) {
			defer wg.Done()
			ffClient.Execute(ctx, p, uID, clientIP, r.collector)
		}(prof, userID, ip)
	}

	// 2. Spawn bots after the humans have cleared /join, so human intent is
	// registered in the pool before the flood arrives.
	time.Sleep(r.humanHeadStart())

	botProfiles := []string{"bot_naive", "bot_solver", "bot_distributed", "bot_retry"}
	for i := 0; i < r.cfg.BotsCount; i++ {
		wg.Add(1)
		pName := botProfiles[i%len(botProfiles)]
		prof := profiles.DefaultProfiles[pName]
		userID := fmt.Sprintf("bot_%s_%d_%d", pName, i, rng.Intn(10000))
		ip := r.ipGen.GetIP("bot", pName, i)

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

// humanHeadStart is how long to let humans get ahead of the bot flood. It is
// sized to clear the slowest human profile's PoW solve (human_slow, 1200ms) plus
// its click delay headroom, so a slow human is not penalised for being slow.
func (r *Runner) humanHeadStart() time.Duration {
	if r.cfg.HumansCount == 0 {
		return 0
	}
	return 2 * time.Second
}
