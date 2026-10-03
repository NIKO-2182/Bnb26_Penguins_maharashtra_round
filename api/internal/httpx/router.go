// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/httpx/router.go
// PURPOSE: Defines the application's routes
// INPUTS / OUTPUTS: N/A
// DEPENDS ON: config, handlers, session, store, ledger, metrics
// USED BY: main.go
// RULES: Every endpoint from API_CONTRACT.md must be here
// DO NOT: N/A
package httpx

import (
	"fairdrop/api/internal/config"
	"fairdrop/api/internal/handlers"
	"fairdrop/api/internal/ledger"
	"fairdrop/api/internal/metrics"
	"fairdrop/api/internal/session"
	"fairdrop/api/internal/store"
	"net/http"
)

type Router struct {
	cfg    *config.Config
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
	mux.HandleFunc("/claim", handlers.NewClaimHandler(r.rdb).HandleClaim)
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
	// Logic will be here or in metrics service
}

func (r *Router) HandleLedger(w http.ResponseWriter, req *http.Request) {
	// Logic will be here or in ledger service
}

func (r *Router) HandleAdminMode(w http.ResponseWriter, req *http.Request) {
	// Logic will be here
}

func (r *Router) HandleAdminReset(w http.ResponseWriter, req *http.Request) {
	// Logic will be here
}

func (r *Router) HandleAdminConfig(w http.ResponseWriter, req *http.Request) {
	// Logic will be here
}
