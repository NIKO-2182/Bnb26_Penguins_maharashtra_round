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
	mux.HandleFunc("/verify", handlers.NewVerifyHandler(mgr, r.rdb).HandleVerify)
	mux.HandleFunc("/session", handlers.NewSessionHandler(mgr, r.rdb).HandleSession)
	mux.HandleFunc("/claim", handlers.NewClaimHandler(r.rdb, r.cfg).HandleClaim)
	mux.HandleFunc("/metrics", r.HandleMetrics)
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

func (r *Router) HandleLedger(w http.ResponseWriter, req *http.Request) {
	limitStr := req.URL.Query().Get("limit")
	limit := int64(50)
	if limitStr != "" {
		if parsed, err := strconv.ParseInt(limitStr, 10, 64); err == nil && parsed > 0 {
			limit = parsed
		}
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
	r.rdb.Client.Del(ctx, store.KeySeatsLeft, store.KeySeatsClaimed, "ledger:events", "metrics:total_reqs", "metrics:429s", "metrics:pool_size", "metrics:granted_seats")
	r.rdb.Client.Set(ctx, store.KeySeatsLeft, r.cfg.TotalSeats, 0)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (r *Router) HandleAdminConfig(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
