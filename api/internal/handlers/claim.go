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
	"fairdrop/api/internal/allocator"
	"fairdrop/api/internal/store"
	"net/http"
)

type ClaimHandler struct {
	rdb *store.RedisClient
}

func NewClaimHandler(rdb *store.RedisClient) *ClaimHandler {
	return &ClaimHandler{rdb: rdb}
}

func (h *ClaimHandler) HandleClaim(w http.ResponseWriter, req *http.Request) {
	var body struct {
		ClaimToken string `json:"claim_token"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	res, err := allocator.ClaimSeat(req.Context(), h.rdb, body.ClaimToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": res.Status})
}
