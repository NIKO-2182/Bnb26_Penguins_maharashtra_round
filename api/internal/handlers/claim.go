// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/handlers/claim.go
// PURPOSE: Handler for /claim
// INPUTS / OUTPUTS: N/A
// DEPENDS ON: allocator.go
// USED BY: httpx/router.go
// RULES: N/A
// DO NOT: N/A
package handlers

import (
	"encoding/json"
	"fairdrop/api/internal/abuse"
	"fairdrop/api/internal/allocator"
	"fairdrop/api/internal/config"
	"fairdrop/api/internal/ledger"
	"fairdrop/api/internal/session"
	"fairdrop/api/internal/store"
	"math/rand"
	"net/http"
	"strings"
)

type ClaimHandler struct {
	rdb         *store.RedisClient
	cfg         *config.Config
	trust       *abuse.TrustManager
	rateLimiter *abuse.RateLimiter
	ledger      *ledger.Service
}

func NewClaimHandler(rdb *store.RedisClient, cfg *config.Config, ledgerSvc *ledger.Service) *ClaimHandler {
	return &ClaimHandler{
		rdb:         rdb,
		cfg:         cfg,
		trust:       abuse.NewTrustManager(rdb, cfg),
		rateLimiter: abuse.NewRateLimiter(rdb, cfg),
		ledger:      ledgerSvc,
	}
}

func extractIP(req *http.Request) string {
	if simIP := req.Header.Get("X-Sim-IP"); simIP != "" {
		return simIP
	}
	if fwd := req.Header.Get("X-Forwarded-For"); fwd != "" {
		parts := strings.Split(fwd, ",")
		return strings.TrimSpace(parts[0])
	}
	return req.RemoteAddr
}

func (h *ClaimHandler) HandleClaim(w http.ResponseWriter, req *http.Request) {
	var body struct {
		ClaimToken string `json:"claim_token"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ip := extractIP(req)
	subnet := abuse.GetSubnet(ip)

	userType := ""
	humanProfile := ""
	if h.cfg.SIMMode {
		userType = req.Header.Get("X-Sim-User-Type")
		humanProfile = req.Header.Get("X-Sim-Profile")
	}

	userID, err := session.VerifyToken(body.ClaimToken, h.cfg.HMACSecret)
	if err != nil {
		userID = body.ClaimToken
	}

	trustScore := h.trust.GetTrustScore(req.Context(), userID)

	// Check Cooldown soft penalty
	if h.rateLimiter.CheckCooldown(req.Context(), userID) {
		h.ledger.RecordEvent(req.Context(), ledger.Event{
			UserID:       userID,
			UserType:     userType,
			HumanProfile: humanProfile,
			IP:           ip,
			Subnet:       subnet,
			TrustScore:   trustScore,
			EventType:    ledger.EventTypeClaim,
			ReasonCode:   ledger.ReasonRejectedRateLimitCooldown,
		})
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{"status": ledger.ReasonRejectedRateLimitCooldown, "message": "Slow down, try again in 5s"})
		return
	}

	randomVal := rand.Float64()
	res, err := allocator.ClaimSeat(req.Context(), h.rdb, body.ClaimToken, trustScore, randomVal)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	reasonCode := res.Status
	h.ledger.RecordEvent(req.Context(), ledger.Event{
		UserID:       userID,
		UserType:     userType,
		HumanProfile: humanProfile,
		IP:           ip,
		Subnet:       subnet,
		TrustScore:   trustScore,
		EventType:    ledger.EventTypeClaim,
		ReasonCode:   reasonCode,
	})

	json.NewEncoder(w).Encode(map[string]string{"status": res.Status})
}
