// FILE: d:/hacks/BitNBuilds/sim/internal/control/server.go
// PURPOSE: HTTP control server listening on port 8090 for scenario execution and reports
// INPUTS / OUTPUTS: JSON endpoints matching SIMULATOR_SPEC.md §3
// DEPENDS ON: sim/internal/runner, sim/internal/report, sim/internal/scenarios
// USED BY: sim/main.go
// RULES: Provide /run, /stop, /status, /report endpoints
// DO NOT: Block handlers indefinitely

package control

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"fairdrop/sim/internal/report"
	"fairdrop/sim/internal/runner"
	"fairdrop/sim/internal/scenarios"
)

type Server struct {
	baseURL    string
	mu         sync.RWMutex
	isRunning  bool
	cancelFn   context.CancelFunc
	lastReport report.SimulationReport
}

func NewServer(baseURL string) *Server {
	return &Server{
		baseURL: baseURL,
	}
}

func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/sim/run", s.handleRun)
	mux.HandleFunc("/run", s.handleRun)
	mux.HandleFunc("/sim/stop", s.handleStop)
	mux.HandleFunc("/stop", s.handleStop)
	mux.HandleFunc("/sim/report", s.handleReport)
	mux.HandleFunc("/report", s.handleReport)
	mux.HandleFunc("/sim/status", s.handleStatus)
	mux.HandleFunc("/status", s.handleStatus)

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 120 * time.Second,
	}

	return server.ListenAndServe()
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "simulator"})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"running":     s.isRunning,
		"last_report": s.lastReport,
	})
}

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.lastReport)
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cancelFn != nil {
		s.cancelFn()
		s.isRunning = false
		s.cancelFn = nil
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "stopped"})
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		http.Error(w, "Simulation already in progress", http.StatusConflict)
		return
	}

	var reqBody struct {
		Scenario string `json:"scenario"`
		Mode     string `json:"mode"`
		Seed     int64  `json:"seed"`
		Seats    int    `json:"seats"`
		Bots     int    `json:"bots"`
		Humans   int    `json:"humans"`
	}
	_ = json.NewDecoder(r.Body).Decode(&reqBody)

	scenarioName := reqBody.Scenario
	if scenarioName == "" {
		scenarioName = "flash_sale"
	}

	mode := reqBody.Mode
	if mode == "" {
		mode = "fair"
	}

	seats := reqBody.Seats
	bots := reqBody.Bots
	humans := reqBody.Humans

	if sc, ok := scenarios.DefaultScenarios[scenarioName]; ok {
		if seats == 0 {
			seats = sc.Seats
		}
		if bots == 0 {
			bots = sc.BotsCount
		}
		if humans == 0 {
			humans = sc.HumansCount
		}
	}

	if seats == 0 {
		seats = 200
	}
	if bots == 0 && humans == 0 {
		bots = 200
		humans = 50
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	s.cancelFn = cancel
	s.isRunning = true
	s.mu.Unlock()

	go func() {
		defer func() {
			s.mu.Lock()
			s.isRunning = false
			s.cancelFn = nil
			s.mu.Unlock()
		}()

		rnr := runner.NewRunner(runner.Config{
			BaseURL:     s.baseURL,
			Scenario:    scenarioName,
			Mode:        mode,
			Seed:        reqBody.Seed,
			Seats:       seats,
			BotsCount:   bots,
			HumansCount: humans,
			Timeout:     120 * time.Second,
		})

		rep := rnr.RunScenario(ctx)

		s.mu.Lock()
		s.lastReport = rep
		s.mu.Unlock()
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "started",
		"scenario": scenarioName,
		"mode":     mode,
		"seats":    seats,
		"bots":     bots,
		"humans":   humans,
	})
}
