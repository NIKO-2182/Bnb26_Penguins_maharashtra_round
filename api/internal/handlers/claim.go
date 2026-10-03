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
	"fairdrop/api/internal/session"
	"fairdrop/api/internal/store"
	"math/rand"
	"net/http"
)

type ClaimHandler struct {
	rdb   *store.RedisClient
	cfg   *config.Config
	trust *abuse.TrustManager
}

func NewClaimHandler(rdb *store.RedisClient, cfg *config.Config) *ClaimHandler {
	return &ClaimHandler{
		rdb:   rdb,
		cfg:   cfg,
		trust: abuse.NewTrustManager(rdb, cfg),
	}
}

func (h *ClaimHandler) HandleClaim(w http.ResponseWriter, req *http.Request) {
	var body struct {
		ClaimToken string `json:"claim_token"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userID, err := session.VerifyToken(body.ClaimToken, h.cfg.HMACSecret)
	if err != nil {
		userID = body.ClaimToken
	}

	trustScore := h.trust.GetTrustScore(req.Context(), userID)
	randomVal := rand.Float64()

	res, err := allocator.ClaimSeat(req.Context(), h.rdb, body.ClaimToken, trustScore, randomVal)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": res.Status})
}
