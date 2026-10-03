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
	"strings"
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
	if token == "" {
		auth := req.Header.Get("Authorization")
		if len(auth) > 7 && auth[:7] == "Bearer " {
			token = auth[7:]
		}
	}
	if token == "" {
		http.Error(w, "missing token parameter", http.StatusBadRequest)
		return
	}

	sess, err := session.GetSession(req.Context(), h.rdb, token)
	if err != nil || sess == nil {
		if strings.Contains(token, " ") {
			altToken := strings.ReplaceAll(token, " ", "+")
			if altSess, altErr := session.GetSession(req.Context(), h.rdb, altToken); altErr == nil && altSess != nil {
				sess = altSess
				err = nil
			}
		}
	}

	if err != nil || sess == nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sess)
}
