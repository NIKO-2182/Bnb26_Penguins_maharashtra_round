// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/metrics/fairness.go
// PURPOSE: Compute real fairness metrics (Bot Advantage Ratio, ablation rows) from the ledger instead of a fixture
// INPUTS / OUTPUTS: RunSummary structs consumed by /results and /metrics
// DEPENDS ON: ledger/store.go, metrics/service.go
// USED BY: httpx/router.go
// RULES: MUST be derived from observed ledger events only; never hardcode a headline number
// DO NOT: Read ground-truth labels for any decision (evaluation use only)
package metrics

import (
	"context"
	"fairdrop/api/internal/ledger"
)

// RunSummary is one measured run's outcome, computed from the ledger.
type RunSummary struct {
	Mode           string  `json:"mode"`
	Seats          int     `json:"seats"`
	SeatsHumans    int     `json:"seats_humans"`
	SeatsBots      int     `json:"seats_bots"`
	TotalSeats     int     `json:"total_seats"`
	BotShare       float64 `json:"bot_share"`
	HumanWinRate   float64 `json:"human_win_rate"`
	HumansInPool   int     `json:"humans_in_pool"`
	BotsInPool     int     `json:"bots_in_pool"`
	Precision      float64 `json:"precision"`
	Recall         float64 `json:"recall"`
	HumanFalseRej  float64 `json:"human_false_rejection_rate"`
	OversellCount  int     `json:"oversell_count"`
	DuplicateCount int     `json:"duplicate_count"`
	EventsScanned  int     `json:"events_scanned"`

	// BotAdvantageRatio is the headline fairness number:
	//   (bots' win rate) / (humans' win rate)
	// 1.0 means a bot is no more likely to win than a human. Above 1 means
	// bots have an advantage. Below 1 means humans are being over-rejected,
	// which is also a failure -- just a different one.
	BotAdvantageRatio float64 `json:"bot_advantage_ratio"`

	// HumansDenied is the count of humans the system actively blocked, and
	// HumansLostToSoldOut is the count that simply arrived after the pool
	// emptied. Conflating these two is what previously made every late
	// arrival look like a false rejection.
	HumansDenied        int `json:"humans_denied"`
	HumansLostToSoldOut int `json:"humans_lost_to_sold_out"`
	BotsDenied          int `json:"bots_denied"`
	AttemptsLimitHit    int `json:"attempts_limit_hit"`
}

// AdvantageRatio is a pure helper so the arithmetic is testable in isolation.
func AdvantageRatio(botsInPool, botSeats, humansInPool, humanSeats int) float64 {
	if humansInPool == 0 || botsInPool == 0 {
		return 0
	}
	botRate := float64(botSeats) / float64(botsInPool)
	humanRate := float64(humanSeats) / float64(humansInPool)
	if humanRate == 0 {
		// Humans were scored and none won: unbounded disadvantage.
		if botRate == 0 {
			return 1
		}
		return 999
	}
	return botRate / humanRate
}

// ComputeRunSummary derives one run's fairness summary from ledger events.
// Only "scored" outcomes count toward the ratio: a request that hit sold_out
// after the pool emptied is not evidence about the classifier.
func ComputeRunSummary(ctx context.Context, ls *ledger.Service, totalSeats int) (RunSummary, error) {
	events, err := ls.GetAll(ctx)
	if err != nil {
		return RunSummary{}, err
	}

	sum := RunSummary{TotalSeats: totalSeats, Seats: totalSeats, EventsScanned: len(events)}

	grantCounts := make(map[string]int)
	var tp, fp, fn, tn int64
	seenBots := make(map[string]struct{})
	seenHumans := make(map[string]struct{})

	for _, ev := range events {
		rejected := isRejectionCode(ev.ReasonCode)
		isGranted := ev.ReasonCode == ledger.ReasonSeatGranted || ev.ReasonCode == ledger.ReasonAcceptedPool

		switch {
		case isGranted:
			sum.Seats--
			grantCounts[ev.UserID]++
			if ev.UserType == "bot" {
				sum.SeatsBots++
				fn++
			} else if ev.UserType == "human" {
				sum.SeatsHumans++
				tn++
			}
		case ev.ReasonCode == ledger.ReasonSoldOut || ev.ReasonCode == ledger.ReasonNotSelectedDraw:
			if ev.UserType == "human" {
				sum.HumansLostToSoldOut++
			}
		case ev.ReasonCode == ledger.ReasonRejectedAttemptLimit:
			sum.AttemptsLimitHit++
			if ev.UserType == "bot" {
				sum.BotsDenied++
			}
		case rejected:
			if ev.UserType == "bot" {
				sum.BotsDenied++
				tp++
			} else if ev.UserType == "human" {
				sum.HumansDenied++
				fp++
			}
		}

		// Pool membership counts DISTINCT users, not requests, otherwise a
		// bot that retries 20 times counts as 20 "people".
		if ev.UserType == "bot" {
			seenBots[ev.UserID] = struct{}{}
		} else if ev.UserType == "human" {
			seenHumans[ev.UserID] = struct{}{}
		}
	}

	// A duplicate GRANT is one user winning more than once. Repeated
	// duplicate_claim events are just a client retrying, which is expected and
	// is NOT a double-allocation -- counting those would report a violation
	// every time a legitimate client retried safely.
	for _, n := range grantCounts {
		if n > 1 {
			sum.DuplicateCount += n - 1
		}
	}
	if sum.Seats < 0 {
		sum.Seats = 0
	}

	sum.BotsInPool = len(seenBots)
	sum.HumansInPool = len(seenHumans)

	if sum.SeatsHumans+sum.SeatsBots > 0 {
		sum.BotShare = float64(sum.SeatsBots) / float64(sum.SeatsHumans+sum.SeatsBots)
	}
	scoredHumans := tn + fp
	if scoredHumans > 0 {
		sum.HumanWinRate = float64(tn) / float64(scoredHumans)
		sum.HumanFalseRej = float64(fp) / float64(scoredHumans)
	}
	if tp+fp > 0 {
		sum.Precision = float64(tp) / float64(tp+fp)
	}
	if tp+fn > 0 {
		sum.Recall = float64(tp) / float64(tp+fn)
	}
	sum.BotAdvantageRatio = AdvantageRatio(sum.BotsInPool, sum.SeatsBots, sum.HumansInPool, sum.SeatsHumans)

	return sum, nil
}

// isRejectionCode mirrors the classifier's definition of a deliberate block.
func isRejectionCode(code string) bool {
	switch code {
	case ledger.ReasonRejectedLowTrust,
		ledger.ReasonRejectedRateLimitIp,
		ledger.ReasonRejectedRateLimitTk,
		ledger.ReasonRejectedRateLimitCooldown,
		ledger.ReasonRejectedSubnetLimit,
		ledger.ReasonRejectedPoWInvalid,
		ledger.ReasonRejectedPoWTooFast,
		ledger.ReasonRejectedAttemptLimit,
		ledger.ReasonRejectedTicketInvalid:
		return true
	}
	return false
}
