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
	"fairdrop/api/internal/abuse"
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
		Token        string `json:"token"`
		T            string `json:"t"`
		Solution     string `json:"solution"`
		Nonce        string `json:"nonce"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tok := body.SessionToken
	if tok == "" {
		tok = body.Token
	}
	if tok == "" {
		tok = body.T
	}

	sol := body.Solution
	if sol == "" {
		sol = body.Nonce
	}

	ip := extractIP(req)
	subnet := abuse.GetSubnet(ip)

	err := h.mgr.Verify(req.Context(), tok, sol, ip, subnet)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "verified",
		"state":  "verified",
	})
}
