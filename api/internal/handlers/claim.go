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
		// Not a legacy userID:signature token. It may be an admission ticket.
		// Distinguish the two by SHAPE first: tickets are pipe-delimited with a
		// queue position, legacy tokens are colon-delimited. Without this, a
		// perfectly valid legacy token would be parsed as a malformed ticket
		// and rejected as forged.
		if strings.Contains(body.ClaimToken, "|") {
			tk, perr := session.ParseTicket(body.ClaimToken)
			if perr == nil {
				if verr := tk.Verify(h.cfg.HMACSecret); verr != nil {
					// Edited queue position or swapped identity: forged.
					h.ledger.RecordEvent(req.Context(), ledger.Event{
						UserID:     tk.UserID,
						EventType:  ledger.EventTypeClaim,
						ReasonCode: ledger.ReasonRejectedTicketInvalid,
						Metadata:   map[string]any{"claimed_seq": tk.Seq},
					})
					http.Error(w, "invalid admission ticket", http.StatusUnauthorized)
					return
				}
				userID = tk.UserID
			} else {
				http.Error(w, "malformed admission ticket", http.StatusBadRequest)
				return
			}
		} else {
			userID = body.ClaimToken
		}
	}

	mode, _ := h.rdb.Client.Get(req.Context(), "admin:mode").Result()
	if mode == "" {
		mode = "fair"
	}

	h.trust.RecordRequestSignal(req.Context(), body.ClaimToken, ip, subnet)

	trustScore := h.trust.GetTrustScore(req.Context(), body.ClaimToken)
	if trustScore < 0.2 && userID != body.ClaimToken {
		trustScore = h.trust.GetTrustScore(req.Context(), userID)
	}

	// In FCFS mode, bypass trust scoring
	if mode == "fcfs" {
		trustScore = 1.0
	}

	// FCFS is the control arm and must stay a pure "no fairness layer"
	// baseline, so all anti-abuse ceilings are disabled in that mode. Otherwise
	// the fcfs control would inherit the very protections it exists to measure
	// against, and the A/B would be meaningless.
	fairMode := mode != "fcfs"

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
	res, attempt, err := allocator.ClaimSeat(req.Context(), h.rdb, body.ClaimToken, trustScore, randomVal, ip, fairMode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	reasonCode := res.Status

	// Record the claim-time feature row. By now the behavioural picture has
	// changed since verify (retry gaps, IP/subnet pressure), so this is a
	// genuinely different observation of the same session rather than a copy.
	claimSigs := h.trust.ExtractNow(req.Context(), body.ClaimToken, ip, subnet, 0)
	h.trust.RecordFeatures(req.Context(), body.ClaimToken, userID, ip, subnet, "claim", trustScore, claimSigs)

	h.ledger.RecordEvent(req.Context(), ledger.Event{
		UserID:       userID,
		UserType:     userType,
		HumanProfile: humanProfile,
		IP:           ip,
		Subnet:       subnet,
		TrustScore:   trustScore,
		EventType:    ledger.EventTypeClaim,
		ReasonCode:   reasonCode,
		Metadata: map[string]any{
			"trust_score":  trustScore,
			"ip_count":     h.trust.GetIPCount(req.Context(), ip),
			"attempt":      attempt,
			"mode":         mode,
			"max_attempts": allocator.MaxDrawAttempts,
			// Signal vector inlined alongside the decision, so the ledger alone
			// is enough to explain or re-train the scorer offline.
			"gap_count":            claimSigs.GapCount,
			"timing_variance":      claimSigs.TimingVariance,
			"burst_count":          claimSigs.BurstCount,
			"request_count":        claimSigs.RequestCount,
			"mean_gap_ms":          claimSigs.MeanGapMs,
			"min_gap_ms":           claimSigs.MinGapMs,
			"max_gap_ms":           claimSigs.MaxGapMs,
			"gap_stddev_ms":        claimSigs.GapStdDevMs,
			"cv":                   claimSigs.CV,
			"burst_ratio":          claimSigs.BurstRatio,
			"gaps_per_second":      claimSigs.GapsPerSecond,
			"subnet_session_count": claimSigs.SubnetSessionCount,
		},
	})

	// Idempotency: if this session already holds a seat, return that same
	// allocation instead of a bare "duplicate_claim". A client whose connection
	// dropped must be able to safely retry and learn its outcome, otherwise it
	// cannot tell "I already won" from "I lost".
	if res.Status == ledger.ReasonDuplicateClaim {
		resp := map[string]any{
			"status":      ledger.ReasonDuplicateClaim,
			"already_won": true,
			"idempotent":  true,
		}
		// Only echo identity back from a ticket whose signature verifies;
		// otherwise a forged token could make the server confirm someone
		// else's allocation.
		if ticket, perr := session.ParseTicket(body.ClaimToken); perr == nil {
			if verr := ticket.Verify(h.cfg.HMACSecret); verr == nil {
				resp["user_id"] = ticket.UserID
				resp["queue_position"] = ticket.Seq
				resp["ticket_valid"] = true
			} else {
				resp["ticket_valid"] = false
			}
		}
		if sess, serr := session.GetSession(req.Context(), h.rdb, body.ClaimToken); serr == nil {
			resp["trust"] = sess.Trust
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": res.Status})
}
