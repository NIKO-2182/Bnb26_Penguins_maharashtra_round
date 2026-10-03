// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/handlers/verify.go
// PURPOSE: Handler for /verify
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

type VerifyHandler struct {
	mgr *session.Manager
	rdb *store.RedisClient
}

func NewVerifyHandler(mgr *session.Manager, rdb *store.RedisClient) *VerifyHandler {
	return &VerifyHandler{mgr: mgr, rdb: rdb}
}

func (h *VerifyHandler) HandleVerify(w http.ResponseWriter, req *http.Request) {
	var body struct {
		SessionToken string `json:"session_token"`
		Solution     string `json:"solution"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := h.mgr.Verify(req.Context(), body.SessionToken, body.Solution)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "verified"})
}
