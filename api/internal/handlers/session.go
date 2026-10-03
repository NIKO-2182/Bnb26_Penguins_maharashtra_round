// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/handlers/session.go
// PURPOSE: Handler for /session
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

type SessionHandler struct {
	mgr *session.Manager
	rdb *store.RedisClient
}

func NewSessionHandler(mgr *session.Manager, rdb *store.RedisClient) *SessionHandler {
	return &SessionHandler{mgr: mgr, rdb: rdb}
}

func (h *SessionHandler) HandleSession(w http.ResponseWriter, req *http.Request) {
	token := req.URL.Query().Get("token")
	sess, err := session.GetSession(req.Context(), h.rdb, token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(sess)
}
