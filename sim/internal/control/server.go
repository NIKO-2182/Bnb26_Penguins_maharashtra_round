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
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"fairdrop/sim/internal/report"
	"fairdrop/sim/internal/runner"
	"fairdrop/sim/internal/scenarios"
)

type Server struct {
	baseURL    string
	adminURL   string
	mu         sync.RWMutex
	isRunning  bool
	cancelFn   context.CancelFunc
	lastReport report.SimulationReport
}

func NewServer(baseURL string) *Server {
	return &Server{
		baseURL:  baseURL,
		adminURL: baseURL,
	}
}

// setAPIMode asks the API to switch allocation mode before the run starts.
//
// The run body's "mode" used to be a LABEL ONLY: it was copied into the report
// and never reached the allocator, so the API stayed in whatever mode it was
// last put into. The result was the dashboard showing two different modes at
// once -- /metrics reading admin:mode while /results showed the sim's label --
// which made an ablation look like it had measured something it had not.
//
// The API is now the single source of truth: the sim sets it, and the report
// reflects what the API actually did.
func (s *Server) setAPIMode(ctx context.Context, mode string) error {
	payload := strings.NewReader(fmt.Sprintf(`{"mode":%q}`, mode))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.adminURL+"/admin/mode", payload)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("admin/mode returned %d", resp.StatusCode)
	}
	return nil
}

// fetchAPIMode reads back the mode the API actually has active, so the report
// can record observed truth rather than what we asked for.
func (s *Server) fetchAPIMode(ctx context.Context) string {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.adminURL+"/metrics", nil)
	if err != nil {
		return ""
	}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	var m struct {
		Mode string `json:"mode"`
	}
	if json.NewDecoder(resp.Body).Decode(&m) != nil {
		return ""
	}
	return m.Mode
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

	// Switch the API into the requested mode BEFORE any traffic starts. Doing it
	// here rather than relying on the caller means a run's mode is always the
	// mode the allocator actually used.
	if err := s.setAPIMode(r.Context(), mode); err != nil {
		s.mu.Unlock()
		http.Error(w, "could not set API mode: "+err.Error(), http.StatusBadGateway)
		return
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

		// Record what the API actually reports, so the report cannot claim a mode
		// the allocator did not use.
		if observed := s.fetchAPIMode(context.Background()); observed != "" {
			rep.Mode = observed
		}

		s.mu.Lock()
		s.lastReport = rep
		s.mu.Unlock()
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       "started",
		"scenario":     scenarioName,
		"mode":         mode,
		"mode_applied": true,
		"seats":        seats,
		"bots":         bots,
		"humans":       humans,
	})
}
