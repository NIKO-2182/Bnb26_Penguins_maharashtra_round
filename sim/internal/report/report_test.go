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

	// 10 Humans (8 successful)
	for i := 0; i < 10; i++ {
		c.Record(ResultMetric{
			UserType: "human",
			Status:   200,
			Latency:  100 * time.Millisecond,
			Claimed:  i < 8,
		})
	}

	// 100 Bots (5 successful, 50 rate limited)
	for i := 0; i < 100; i++ {
		status := 200
		if i >= 50 {
			status = 429
		}
		c.Record(ResultMetric{
			UserType: "bot",
			Status:   status,
			Latency:  50 * time.Millisecond,
			Claimed:  i < 5,
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
	if rep.BotSuccessRate != 5.0 {
		t.Errorf("Expected 5.0 bot success rate, got %f", rep.BotSuccessRate)
	}
}
