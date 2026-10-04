// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/httpx/router.go
// PURPOSE: Defines the application's routes
// INPUTS / OUTPUTS: N/A
// DEPENDS ON: config, handlers, session, store, ledger, metrics
// USED BY: main.go
// RULES: Every endpoint from API_CONTRACT.md must be here
// DO NOT: N/A
package httpx

import (
	"encoding/json"
	"fairdrop/api/internal/abuse"
	"fairdrop/api/internal/config"
	"fairdrop/api/internal/handlers"
	"fairdrop/api/internal/ledger"
	"fairdrop/api/internal/metrics"
	"fairdrop/api/internal/session"
	"fairdrop/api/internal/store"
	"net/http"
	"strconv"
)

type Router struct {
	cfg     *config.Config
	rdb     *store.RedisClient
	ledger  *ledger.Service
	metrics *metrics.Service
}

func NewRouter(cfg *config.Config, rdb *store.RedisClient) *Router {
	return &Router{
		cfg:     cfg,
		rdb:     rdb,
		ledger:  ledger.NewService(rdb),
		metrics: metrics.NewService(rdb, cfg),
	}
}

func (r *Router) Routes() *http.ServeMux {
	mux := http.NewServeMux()

	mgr := session.NewManager(r.rdb, r.cfg)

	mux.HandleFunc("/health", r.HandleHealth)
	mux.HandleFunc("/join", handlers.NewJoinHandler(mgr, r.rdb).HandleJoin)
	mux.HandleFunc("/verify", handlers.NewVerifyHandler(mgr, r.rdb, r.cfg).HandleVerify)
	mux.HandleFunc("/session", handlers.NewSessionHandler(mgr, r.rdb).HandleSession)
	mux.HandleFunc("/claim", handlers.NewClaimHandler(r.rdb, r.cfg, r.ledger).HandleClaim)
	mux.HandleFunc("/metrics", r.HandleMetrics)
	mux.HandleFunc("/features", r.HandleFeatures)
	mux.HandleFunc("/results", r.HandleResults)
	mux.HandleFunc("/ledger", r.HandleLedger)
	mux.HandleFunc("/admin/mode", r.HandleAdminMode)
	mux.HandleFunc("/admin/reset", r.HandleAdminReset)
	mux.HandleFunc("/admin/config", r.HandleAdminConfig)

	return mux
}

func (r *Router) HandleHealth(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (r *Router) HandleMetrics(w http.ResponseWriter, req *http.Request) {
	data, err := r.metrics.GetMetrics(req.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// HandleResults returns MEASURED run outcomes.
//
// This previously returned a hardcoded two-row fixture (bot_share 0.02 vs
// 0.88) that the dashboard rendered as if it were an experiment. Every number
// here is now derived from the hash-chained ledger, so the ablation table is
// evidence rather than illustration. When no run has happened yet the response
// is an empty, clearly-labelled result rather than invented data.
func (r *Router) HandleResults(w http.ResponseWriter, req *http.Request) {
	totalSeats := r.cfg.TotalSeats
	if totalSeats == 0 {
		totalSeats = 500
	}

	mode, _ := r.rdb.Client.Get(req.Context(), "admin:mode").Result()
	if mode == "" {
		mode = "fair"
	}

	sum, err := metrics.ComputeRunSummary(req.Context(), r.ledger, totalSeats)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// The sim used to label its own report with whatever mode it was asked for,
	// which could disagree with the mode the allocator actually ran in. The API
	// is authoritative: stamp the observed mode onto the summary so /metrics,
	// /results and the simulator can never show three different answers.
	sum.Mode = mode

	chain, _ := r.ledger.VerifyChain(req.Context(), 0)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"runs":           []metrics.RunSummary{sum},
		"measured":       true,
		"chain_ok":       chain.OK,
		"chain_checked":  chain.Checked,
		"chain_head":     chain.HeadHash,
		"chain_error":    chain.Error,
		"total_seats":    totalSeats,
		"current_mode":   mode,
		"ablation_hint":  "run mode=fcfs then mode=fair and compare bot_advantage_ratio",
		"interpretation": "bot_advantage_ratio ~= 1.0 means a bot is no likelier to win than a human",
	})
}

func (r *Router) HandleLedger(w http.ResponseWriter, req *http.Request) {
	limitStr := req.URL.Query().Get("limit")
	limit := int64(50)
	if limitStr != "" {
		if parsed, err := strconv.ParseInt(limitStr, 10, 64); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	// decided=1 returns only events that reached an actual system decision.
	//
	// During a flash sale the pool empties in seconds but thousands of
	// "sold_out" requests keep arriving for a minute afterwards — roughly 94%
	// of the whole log. Any newest-N window is therefore all tail, and a
	// visualisation of "arrivals" would show only identical late refusals and
	// never the opening rush that is actually worth watching. The server has to
	// filter, because the interesting events are far outside any feasible
	// client-side window.
	if req.URL.Query().Get("decided") == "1" {
		// Scan the WHOLE ledger, not a window. During a flash sale the tail is
		// ~94% of all traffic, so even a 10x window can land entirely inside it
		// and return nothing at all. Scanning everything is cheap here (a single
		// LRANGE) and guarantees the decided events are always reachable.
		total, _ := r.rdb.Client.LLen(req.Context(), "ledger:events").Result()
		scan := total
		if scan > 200000 {
			scan = 200000
		}
		events, err := r.ledger.GetRecent(req.Context(), scan)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		filtered := make([]ledger.Event, 0, len(events))
		for _, ev := range events {
			if ev.ReasonCode == ledger.ReasonSoldOut || ev.ReasonCode == ledger.ReasonNotSelectedDraw {
				continue
			}
			filtered = append(filtered, ev)
		}
		// Honour the client's requested cap AFTER filtering.
		if n := limit; n > 0 && int64(len(filtered)) > n {
			filtered = filtered[:n]
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(filtered)
		return
	}

	events, err := r.ledger.GetRecent(req.Context(), limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

func (r *Router) HandleAdminMode(w http.ResponseWriter, req *http.Request) {
	var body struct {
		Mode string `json:"mode"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	r.rdb.Client.Set(req.Context(), "admin:mode", body.Mode, 0)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (r *Router) HandleAdminReset(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	totalSeats := r.cfg.TotalSeats
	if totalSeats == 0 {
		totalSeats = 500
	}
	// A full reset must clear the chain head and sequence too, or the next run's
	// first event inherits the previous run's hash and the chain no longer starts
	// at genesis -- which is exactly what a reset needs to guarantee.
	r.rdb.Client.Del(ctx, store.KeySeatsLeft, store.KeySeatsClaimed, "ledger:events", "metrics:total_reqs", "metrics:429s", "metrics:pool_size", "metrics:granted_seats", "feat:all", ledger.ChainStateKey, ledger.ChainSeqKey)
	r.rdb.Client.Set(ctx, store.KeySeatsLeft, totalSeats, 0)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (r *Router) HandleAdminConfig(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// HandleFeatures streams the recorded behavioural feature vectors. This is the
// training/analysis surface: it exposes features only, never labels. The
// label (bot/human) lives in the ledger's user_type field and is joined offline
// by an analyst, so no decision-path code can ever read ground truth.
func (r *Router) HandleFeatures(w http.ResponseWriter, req *http.Request) {
	limit := int64(50000)
	if s := req.URL.Query().Get("limit"); s != "" {
		if parsed, err := strconv.ParseInt(s, 10, 64); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	tm := abuse.NewTrustManager(r.rdb, r.cfg)
	recs, err := tm.ExportFeatureRecords(req.Context(), limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	total := tm.CountFeatureRecords(req.Context())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":         "ok",
		"returned":       len(recs),
		"total_recorded": total,
		"note":           "features only; labels are joined offline from /ledger user_type",
		"records":        recs,
	})
}
