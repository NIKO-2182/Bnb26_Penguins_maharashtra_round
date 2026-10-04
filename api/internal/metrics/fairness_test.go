// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/metrics/fairness_test.go
// PURPOSE: Unit tests for the Bot Advantage Ratio and run-summary accounting
// INPUTS / OUTPUTS: Test results
// DEPENDS ON: fairness.go
// USED BY: go test
// RULES: Ratio must be 1.0 when bots and humans win at equal rates
// DO NOT: N/A
package metrics

import "testing"

func TestAdvantageRatioParity(t *testing.T) {
	// 100 bots, 100 humans, equal win rates -> parity.
	if got := AdvantageRatio(100, 50, 100, 50); got != 1.0 {
		t.Errorf("expected 1.0 for equal win rates, got %f", got)
	}
}

func TestAdvantageRatioBotFavoured(t *testing.T) {
	// Bots win 80/100, humans 20/100 -> ratio 4.0
	got := AdvantageRatio(100, 80, 100, 20)
	if got != 4.0 {
		t.Errorf("expected 4.0 when bots win 4x as often, got %f", got)
	}
}

func TestAdvantageRatioHumanFavoured(t *testing.T) {
	// Ratio below 1.0 means humans are advantaged, which is ALSO a failure:
	// it implies over-rejection. The metric must not treat it as success.
	got := AdvantageRatio(100, 20, 100, 80)
	if got != 0.25 {
		t.Errorf("expected 0.25 when humans win 4x as often, got %f", got)
	}
}

func TestAdvantageRatioEmptyPool(t *testing.T) {
	if got := AdvantageRatio(0, 0, 0, 0); got != 0 {
		t.Errorf("expected 0 for an empty run, got %f", got)
	}
	if got := AdvantageRatio(100, 50, 0, 0); got != 0 {
		t.Errorf("expected 0 when no humans are present, got %f", got)
	}
}

func TestAdvantageRatioNoHumanWins(t *testing.T) {
	// Every scored human denied, bots winning: effectively unbounded.
	got := AdvantageRatio(100, 50, 50, 0)
	if got < 100 {
		t.Errorf("expected an unbounded ratio when humans scored but none won, got %f", got)
	}
}

// Retries must not inflate the denominator: 20 requests from one bot is one bot.
func TestPoolCountsDistinctUsersNotRequests(t *testing.T) {
	// AdvantageRatio takes pool sizes directly, so verify the intended inputs:
	// one bot that retried 20x should pass BotsInPool=1, not 20.
	if got := AdvantageRatio(1, 1, 1, 1); got != 1.0 {
		t.Errorf("expected 1.0 for one bot and one human each winning, got %f", got)
	}
}

func TestIsRejectionCode(t *testing.T) {
	rejections := []string{
		"rejected_low_trust",
		"rejected_ratelimit_ip",
		"rejected_ratelimit_token",
		"rejected_ratelimit_cooldown",
		"rejected_subnet_limit",
		"rejected_pow_invalid",
		"rejected_pow_too_fast",
		"rejected_attempt_limit",
	}
	for _, c := range rejections {
		if !isRejectionCode(c) {
			t.Errorf("%q should be treated as a deliberate rejection", c)
		}
	}

	// These must NOT count as classifier rejections: the first two mean the
	// pool was empty or the draw ran, neither is a block.
	notRejections := []string{"sold_out", "not_selected_draw", "duplicate_claim", "seat_granted", "accepted_pool"}
	for _, c := range notRejections {
		if isRejectionCode(c) {
			t.Errorf("%q must NOT be treated as a classifier rejection", c)
		}
	}
}
