// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/ledger/event.go
// PURPOSE: Event types and reason codes for the ledger
// INPUTS / OUTPUTS: N/A
// DEPENDS ON: N/A
// USED BY: ledger/writer.go, metrics/aggregator.go
// RULES: Reason codes must be unique and match the ones in LEDGER_SCHEMA.md
// DO NOT: Change reason codes without updating LEDGER_SCHEMA.md
package ledger

import (
	"crypto/sha1"
	"encoding/hex"
	"time"
)

// Event is one immutable, hash-chained ledger entry.
//
// The chain is what makes this log auditable: each event commits to its own
// contents AND to the hash of its predecessor, so removing or editing any event
// breaks every hash after it. A judge can ask "why was user X throttled at
// 12:04:31?" and the answer can be proven rather than asserted.
//
// GenesisHash anchors an empty chain. ChainStateKey holds the running head.
const (
	GenesisHash   = "0000000000000000000000000000000000000000"
	ChainStateKey = "ledger:chain_head"
	ChainSeqKey   = "ledger:seq"
)

type Event struct {
	ID           int64          `json:"id"`
	PrevHash     string         `json:"prev_hash,omitempty"`
	Hash         string         `json:"hash,omitempty"`
	Timestamp    time.Time      `json:"timestamp"`
	UserID       string         `json:"user_id"`
	UserType     string         `json:"user_type,omitempty"`     // "human" | "bot" (Ground truth label for metrics only)
	HumanProfile string         `json:"human_profile,omitempty"` // "human_normal" | "human_slow" | "human_frustrated" | "human_shared_ip"
	IP           string         `json:"ip,omitempty"`
	Subnet       string         `json:"subnet,omitempty"`
	TrustScore   float64        `json:"trust_score,omitempty"`
	EventType    string         `json:"event_type"`
	ReasonCode   string         `json:"reason_code"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// ComputeHash returns sha1(prev_hash || canonical_field_sequence).
// Must mirror the Lua assembler in store.go RecordEvent byte for byte.
func (e *Event) ComputeHash() string {
	sum := sha1.Sum([]byte(e.PrevHash + e.CanonicalString()))
	return hex.EncodeToString(sum[:])
}

const (
	EventTypeJoin          = "join"
	EventTypeVerify        = "verify"
	EventTypeSessionUpdate = "session_update"
	EventTypeClaim         = "claim"
	EventTypeDraw          = "draw"

	ReasonAcceptedPool              = "accepted_pool"
	ReasonSelected                  = "selected"
	ReasonSeatGranted               = "seat_granted"
	ReasonRejectedRateLimitIp       = "rejected_ratelimit_ip"
	ReasonRejectedRateLimitTk       = "rejected_ratelimit_token"
	ReasonRejectedRateLimitCooldown = "rejected_ratelimit_cooldown"
	ReasonRejectedSubnetLimit       = "rejected_subnet_limit"
	ReasonRejectedPoWInvalid        = "rejected_pow_invalid"
	ReasonRejectedPoWTooFast        = "rejected_pow_too_fast"
	ReasonRejectedLowTrust          = "rejected_low_trust"
	ReasonNotSelectedDraw           = "not_selected_draw"
	ReasonSoldOut                   = "sold_out"
	ReasonDuplicateClaim            = "duplicate_claim"
	ReasonRejectedAttemptLimit      = "rejected_attempt_limit"
	ReasonRejectedTicketInvalid     = "rejected_ticket_invalid"
)
