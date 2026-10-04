// FILE: d:/hacks/BitNBuilds/sim/internal/report/report_test.go
// PURPOSE: Unit test for simulator report generation and fairness score logic
// INPUTS / OUTPUTS: Tests collector metric processing
// DEPENDS ON: sim/internal/report
// USED BY: go test
// RULES: Verify fairness score and latency calculation
// DO NOT: N/A

package report

import (
	"testing"
	"time"
)

func TestReportGeneration(t *testing.T) {
	c := NewCollector()

	// 10 Humans (8 successful, 2 blocked by the classifier)
	for i := 0; i < 10; i++ {
		reason := "rejected_low_trust"
		if i < 8 {
			reason = "seat_granted"
		}
		c.Record(ResultMetric{
			UserType: "human",
			Status:   200,
			Latency:  100 * time.Millisecond,
			Claimed:  i < 8,
			Reason:   reason,
		})
	}

	// 100 Bots (5 successful, 45 blocked, 50 rate limited)
	for i := 0; i < 100; i++ {
		status := 200
		reason := "rejected_low_trust"
		if i >= 50 {
			status = 429
		}
		if i < 5 {
			reason = "seat_granted"
		}
		c.Record(ResultMetric{
			UserType: "bot",
			Status:   status,
			Latency:  50 * time.Millisecond,
			Claimed:  i < 5,
			Reason:   reason,
		})
	}

	rep := c.GenerateReport("test_scenario", "fair", 12345, 100)

	if rep.HumanRequests != 10 {
		t.Errorf("Expected 10 human requests, got %d", rep.HumanRequests)
	}
	if rep.BotRequests != 100 {
		t.Errorf("Expected 100 bot requests, got %d", rep.BotRequests)
	}
	if rep.HumanClaims != 8 {
		t.Errorf("Expected 8 human claims, got %d", rep.HumanClaims)
	}
	if rep.BotClaims != 5 {
		t.Errorf("Expected 5 bot claims, got %d", rep.BotClaims)
	}
	if rep.RateLimiterHits != 50 {
		t.Errorf("Expected 50 rate limiter hits, got %d", rep.RateLimiterHits)
	}
	if rep.HumanSuccessRate != 80.0 {
		t.Errorf("Expected 80.0 human success rate, got %f", rep.HumanSuccessRate)
	}
	// All 100 bot requests scored: the 50 rate-limited ones carry a rejection
	// reason too, so the denominator stays 100.
	if rep.BotSuccessRate != 5.0 {
		t.Errorf("Expected 5.0 bot success rate (5/100 scored), got %f", rep.BotSuccessRate)
	}
	if rep.BotScored != 100 {
		t.Errorf("Expected 100 bots scored, got %d", rep.BotScored)
	}
	if rep.HumanScored != 10 {
		t.Errorf("Expected 10 humans scored, got %d", rep.HumanScored)
	}
	if rep.SoldOutRequests != 0 {
		t.Errorf("Expected 0 sold_out, got %d", rep.SoldOutRequests)
	}
	if rep.SeatsClaimed != 13 {
		t.Errorf("Expected 13 seats granted (8 human + 5 bot), got %d", rep.SeatsClaimed)
	}
}

// A pool that empties mid-run must NOT be scored as a classifier failure.
// This is the regression guard for the 100%-false-rejection bug: sold_out
// requests arrive after the pool is empty and say nothing about the classifier.
func TestSoldOutIsNotARejection(t *testing.T) {
	c := NewCollector()

	// 400 humans arrive after the pool emptied.
	for i := 0; i < 400; i++ {
		c.Record(ResultMetric{
			UserType: "human",
			Status:   200,
			Latency:  10 * time.Millisecond,
			Claimed:  false,
			Reason:   "sold_out",
		})
	}
	// 2 bots correctly blocked, 3 bots win seats.
	for i := 0; i < 2; i++ {
		c.Record(ResultMetric{UserType: "bot", Status: 200, Reason: "rejected_low_trust"})
	}
	for i := 0; i < 3; i++ {
		c.Record(ResultMetric{UserType: "bot", Status: 200, Claimed: true, Reason: "seat_granted"})
	}

	rep := c.GenerateReport("s", "fair", 1, 5)

	if rep.SoldOutRequests != 400 {
		t.Errorf("Expected 400 sold_out counted, got %d", rep.SoldOutRequests)
	}
	if rep.FP != 0 {
		t.Errorf("sold_out must not count as a human false positive, got FP=%d", rep.FP)
	}
	if rep.HumanScored != 0 {
		t.Errorf("Expected 0 humans scored (all sold_out), got %d", rep.HumanScored)
	}
	if rep.TP != 2 {
		t.Errorf("Expected 2 bots blocked (TP), got %d", rep.TP)
	}
	if rep.FN != 3 {
		t.Errorf("Expected 3 bots through (FN), got %d", rep.FN)
	}
	// Only 3 claims were granted and all 3 went to bots: 100% bot share of
	// the seats actually won.
	if rep.BotSeatShare != 100.0 {
		t.Errorf("Expected 100.0 bot seat share (3/3 grants), got %f", rep.BotSeatShare)
	}
	if rep.SeatsClaimed != 3 {
		t.Errorf("Expected 3 seats claimed, got %d", rep.SeatsClaimed)
	}
	// No human was ever scored, so no false-rejection rate can be claimed.
	if rep.HumanFalseRejectionRate != 0.0 {
		t.Errorf("Expected 0.0 false rejection rate, got %f", rep.HumanFalseRejectionRate)
	}
}

// The Lua allocator must never hand out more seats than the pool held.
func TestOversellDetected(t *testing.T) {
	c := NewCollector()
	for i := 0; i < 60; i++ {
		c.Record(ResultMetric{UserType: "bot", Status: 200, Claimed: true, Reason: "seat_granted"})
	}
	rep := c.GenerateReport("s", "fair", 1, 50) // pool of 50
	if rep.OversellCount != 10 {
		t.Errorf("Expected oversell of 10, got %d", rep.OversellCount)
	}
	if rep.SeatsClaimed != 60 {
		t.Errorf("Expected 60 seats claimed, got %d", rep.SeatsClaimed)
	}
}
