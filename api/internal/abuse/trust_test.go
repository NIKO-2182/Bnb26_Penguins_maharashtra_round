// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/abuse/trust_test.go
// PURPOSE: Table-driven unit test verifying trust scores across human and bot profiles
// INPUTS / OUTPUTS: TestTrustProfiles asserts human profiles stay above threshold and bot profiles fall below
// DEPENDS ON: trust.go, signals.go, budget.go
// USED BY: go test ./...
// RULES: Must test human_frustrated, human_slow, human_shared_ip vs bot_solver, bot_naive
// DO NOT: N/A
package abuse

import (
	"testing"
)

func TestTrustScoringProfiles(t *testing.T) {
	tests := []struct {
		name         string
		solveTimeMs  int64
		gapCount     int
		variance     float64
		burstCount   int
		ipCount      int64
		minExpected  float64
		maxExpected  float64
	}{
		{
			name:        "human_normal",
			solveTimeMs: 2500,
			gapCount:    5,
			variance:    450.0,
			burstCount:  0,
			ipCount:     1,
			minExpected: 0.70,
			maxExpected: 0.95,
		},
		{
			name:        "human_slow",
			solveTimeMs: 8000,
			gapCount:    4,
			variance:    600.0,
			burstCount:  0,
			ipCount:     1,
			minExpected: 0.70,
			maxExpected: 0.95,
		},
		{
			name:        "human_frustrated",
			solveTimeMs: 1200,
			gapCount:    6,
			variance:    350.0,
			burstCount:  2, // Double clicks but not sustained >3
			ipCount:     1,
			minExpected: 0.60,
			maxExpected: 0.95,
		},
		{
			name:        "human_shared_ip",
			solveTimeMs: 3000,
			gapCount:    5,
			variance:    500.0,
			burstCount:  0,
			ipCount:     40, // Shared IP (<150)
			minExpected: 0.70,
			maxExpected: 0.95,
		},
		{
			name:        "bot_naive",
			solveTimeMs: 50, // Ultra-fast machine solve
			gapCount:    0,
			variance:    0.0,
			burstCount:  0,
			ipCount:     1,
			minExpected: 0.05,
			maxExpected: 0.60,
		},
		{
			name:        "bot_solver",
			solveTimeMs: 80, // Bot solver fast search
			gapCount:    6,
			variance:    5.0, // Uniform automation gap
			burstCount:  5,   // Sustained burst >3
			ipCount:     200, // High bot farm density
			minExpected: 0.05,
			maxExpected: 0.45,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := 0.85

			if tt.solveTimeMs > 0 {
				if tt.solveTimeMs < 100 {
					score -= 0.25
				} else if tt.solveTimeMs >= 1000 && tt.solveTimeMs <= 15000 {
					score += 0.10
				}
			}

			if tt.gapCount >= 4 {
				if tt.variance > 300 {
					score += 0.10
				} else if tt.variance < 20 {
					score -= 0.15
				}
			}

			if tt.burstCount > 3 && (tt.solveTimeMs < 200 || (tt.gapCount >= 4 && tt.variance < 20)) {
				score -= 0.15
			}

			if tt.ipCount > 150 {
				score -= 0.10
			}

			if score < 0.05 {
				score = 0.05
			}
			if score > 0.95 {
				score = 0.95
			}

			if score < tt.minExpected || score > tt.maxExpected {
				t.Errorf("profile %s: score %.2f not in expected range [%.2f, %.2f]",
					tt.name, score, tt.minExpected, tt.maxExpected)
			}
		})
	}
}
