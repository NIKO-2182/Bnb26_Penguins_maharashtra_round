// FILE: d:/hacks/BitNBuilds/sim/internal/report/ledger_pull.go
// PURPOSE: Pulls ledger data from API backend to perform sim-vs-ledger cross-checking
// INPUTS / OUTPUTS: Returns ledger event counts and cross-check status
// DEPENDS ON: N/A
// USED BY: sim/internal/report/report.go
// RULES: Verify that sim-side total requests match ledger event totals
// DO NOT: N/A

package report

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type LedgerEvent struct {
	UserID       string  `json:"user_id"`
	UserType     string  `json:"user_type"`
	HumanProfile string  `json:"human_profile"`
	IP           string  `json:"ip"`
	EventType    string  `json:"event_type"`
	ReasonCode   string  `json:"reason_code"`
	TrustScore   float64 `json:"trust_score"`
}

func FetchLedgerCrossCheck(apiBaseURL string, simTotal int) map[string]interface{} {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(fmt.Sprintf("%s/ledger?limit=5000", apiBaseURL))
	if err != nil || resp == nil || resp.StatusCode != http.StatusOK {
		return map[string]interface{}{
			"status": "unavailable",
			"error":  "Failed to fetch ledger endpoint",
		}
	}
	defer resp.Body.Close()

	var events []LedgerEvent
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		return map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		}
	}

	ledgerTotal := len(events)
	mismatch := ledgerTotal != simTotal

	return map[string]interface{}{
		"status":        "verified",
		"sim_totals":    simTotal,
		"ledger_totals": ledgerTotal,
		"mismatch":      mismatch,
	}
}
