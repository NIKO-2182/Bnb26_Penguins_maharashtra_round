// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/handlers/join.go
// PURPOSE: Handler for /join
// INPUTS / OUTPUTS: N/A
// DEPENDS ON: session.go
// USED BY: httpx/router.go
// RULES: N/A
// DO NOT: N/A
package handlers

import (
	"encoding/json"
	"fairdrop/api/internal/session"
	"fairdrop/api/internal/store"
	"net/http"
)

type JoinHandler struct {
	mgr *session.Manager
	rdb *store.RedisClient
}

func NewJoinHandler(mgr *session.Manager, rdb *store.RedisClient) *JoinHandler {
	return &JoinHandler{mgr: mgr, rdb: rdb}
}

func (h *JoinHandler) HandleJoin(w http.ResponseWriter, req *http.Request) {
	var body struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	token, err := h.mgr.Join(req.Context(), body.Username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"session_token": token, "pow_challenge": "solve_me"})
}
