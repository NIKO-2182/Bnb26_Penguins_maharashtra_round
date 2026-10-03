// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/ledger/event.go
// PURPOSE: Event types and reason codes for the ledger
// INPUTS / OUTPUTS: N/A
// DEPENDS ON: N/A
// USED BY: ledger/writer.go, metrics/aggregator.go
// RULES: Reason codes must be unique and match the ones in LEDGER_SCHEMA.md
// DO NOT: Change reason codes without updating LEDGER_SCHEMA.md
package ledger

import "time"

type Event struct {
	ID         int64           `json:"id"`
	Timestamp  time.Time       `json:"timestamp"`
	UserID     string          `json:"user_id"`
	EventType  string          `json:"event_type"`
	ReasonCode string          `json:"reason_code"`
	Metadata   map[string]any  `json:"metadata"`
}

const (
	EventTypeJoin          = "join"
	EventTypeVerify        = "verify"
	EventTypeSessionUpdate = "session_update"
	EventTypeClaim         = "claim"
	EventTypeDraw          = "draw"

	ReasonAcceptedPool        = "accepted_pool"
	ReasonSelected            = "selected"
	ReasonSeatGranted         = "seat_granted"
	ReasonRejectedRateLimitIp = "rejected_ratelimit_ip"
	ReasonRejectedRateLimitTk = "rejected_ratelimit_token"
	ReasonRejectedPoWInvalid  = "rejected_pow_invalid"
	ReasonRejectedPoWTooFast  = "rejected_pow_too_fast"
	ReasonRejectedLowTrust    = "rejected_low_trust"
	ReasonNotSelectedDraw     = "not_selected_draw"
	ReasonSoldOut             = "sold_out"
	ReasonDuplicateClaim      = "duplicate_claim"
)
