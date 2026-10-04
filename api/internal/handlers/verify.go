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
	"fairdrop/api/internal/config"
	"fairdrop/api/internal/session"
	"fairdrop/api/internal/store"
	"net/http"
)

type VerifyHandler struct {
	mgr *session.Manager
	rdb *store.RedisClient
	cfg *config.Config
}

func NewVerifyHandler(mgr *session.Manager, rdb *store.RedisClient, cfg *config.Config) *VerifyHandler {
	return &VerifyHandler{mgr: mgr, rdb: rdb, cfg: cfg}
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

	// Record the arrival BEFORE evaluating trust. Previously signals were only
	// recorded on /claim, which happens after the score is already fixed -- so
	// the variance and burst checks always saw GapCount==0 and never fired.
	// Recording here gives the trust pass a real timeline to inspect.
	trustMgr := abuse.NewTrustManager(h.rdb, h.cfg)
	trustMgr.RecordRequestSignal(req.Context(), tok, ip, subnet)

	sess, err := session.GetSession(req.Context(), h.rdb, tok)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	trustScore, sigs, err := h.mgr.VerifyDetailed(req.Context(), tok, sol, ip, subnet)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Persist the behavioural feature vector. Previously these signals were
	// computed inside EvaluateSessionTrust, reduced to a scalar, and discarded,
	// so no training data existed for offline analysis.
	trustMgr.RecordFeatures(req.Context(), tok, sess.UserID, ip, subnet, "verify", trustScore, sigs)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status": "verified",
		"state":  "verified",
		"trust":  trustScore,
	})
}
