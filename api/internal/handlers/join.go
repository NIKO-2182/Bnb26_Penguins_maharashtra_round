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
		body.Username = "user_anon"
	}
	if body.Username == "" {
		body.Username = "user_anon"
	}

	token, err := h.mgr.Join(req.Context(), body.Username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token":         token,
		"session_token": token,
		"challenge":     "solve_me",
		"pow_challenge": "solve_me",
		"difficulty":    4,
	})
}
