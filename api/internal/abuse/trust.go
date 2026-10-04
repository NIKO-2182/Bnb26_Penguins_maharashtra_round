// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/abuse/trust.go
// PURPOSE: Dynamic behavior-based trust scoring with guarded soft signals and shared IP ceilings
// INPUTS / OUTPUTS: Returns trust float64 [0.0 - 1.0]
// DEPENDS ON: store/redis.go, config, signals.go, budget.go
// USED BY: session.go, handlers/claim.go
// RULES: Must maintain high scores (>0.5) for frustrated, slow, and shared IP human profiles
// DO NOT: Read X-Sim ground-truth headers in decision code
package abuse

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"fairdrop/api/internal/config"
	"fairdrop/api/internal/store"
)

type TrustManager struct {
	rdb       *store.RedisClient
	cfg       *config.Config
	extractor *SignalExtractor
	budget    *BudgetManager
}

func NewTrustManager(rdb *store.RedisClient, cfg *config.Config) *TrustManager {
	return &TrustManager{
		rdb:       rdb,
		cfg:       cfg,
		extractor: NewSignalExtractor(rdb),
		budget:    NewBudgetManager(rdb, cfg),
	}
}

func (tm *TrustManager) GetTrustScore(ctx context.Context, userID string) float64 {
	val, err := tm.rdb.Client.Get(ctx, "trust:"+userID).Result()
	if err == nil && val != "" {
		if s, parseErr := strconv.ParseFloat(val, 64); parseErr == nil {
			return s
		}
	}
	return 0.1
}

func (tm *TrustManager) SetTrustScore(ctx context.Context, userID string, score float64) {
	if score < 0.0 {
		score = 0.0
	}
	if score > 1.0 {
		score = 1.0
	}
	tm.rdb.Client.Set(ctx, "trust:"+userID, fmt.Sprintf("%.2f", score), 1*time.Hour)
}

func (tm *TrustManager) RecordInteraction(ctx context.Context, userID string, success bool) {
	if success {
		tm.SetTrustScore(ctx, userID, 0.95)
	} else {
		tm.SetTrustScore(ctx, userID, 0.05)
	}
}

// EvaluateSessionTrust scores a session AND returns the raw signal vector it
// scored from. The signals used to be computed, consumed, and discarded, which
// meant the ledger recorded only the resulting scalar -- leaving no feature
// matrix for offline analysis or a future classifier. Callers must persist the
// returned Signals; the score alone is not enough to explain or audit a decision.
func (tm *TrustManager) EvaluateSessionTrust(ctx context.Context, token, ip, subnet string, solveTimeMs int64) (float64, Signals) {
	sigs := tm.extractor.Extract(ctx, token, ip, subnet, solveTimeMs)

	// Base score starts high for PoW-verified sessions
	score := 0.85

	// Signal 1: Soft PoW solve time penalty (scaling, never hard rejection alone)
	if sigs.SolveTimeMs > 0 {
		if sigs.SolveTimeMs < 100 {
			score -= 0.25 // Extremely fast solve soft penalty
		} else if sigs.SolveTimeMs >= 1000 && sigs.SolveTimeMs <= 15000 {
			score += 0.10 // Normal human solve time bonus
		}
	}

	// Signal 2: Timing variance with minimum 4 gap requirement
	if sigs.GapCount >= 4 {
		if sigs.TimingVariance > 300 {
			score += 0.10 // Human irregularity bonus
		} else if sigs.TimingVariance < 20 {
			score -= 0.15 // Uniform automated bot gap penalty
		}
	}

	// Signal 3: Sustained burst penalty (penalize bursts only when coupled with sub-human solve time or uniform bot gaps)
	if sigs.BurstCount > 3 && (sigs.SolveTimeMs < 200 || (sigs.GapCount >= 4 && sigs.TimingVariance < 20)) {
		score -= 0.15
	}

	// Signal 4: Shared IP ceiling (supports shared_ip scenario: 200 humans / 5 IPs)
	if sigs.IPSessionCount > 150 {
		score -= 0.10 // Soft penalty for ultra-dense shared IP
	}

	// Signal 5: Label-free budget relaxation check
	if tm.budget.ShouldRelax(ctx) {
		score += 0.15
	}

	// Clamp to [0.05, 0.95]
	if score < 0.05 {
		score = 0.05
	}
	if score > 0.95 {
		score = 0.95
	}

	// Record stats for budget controller
	tm.budget.RecordVerifiedSession(ctx)
	if score >= 0.2 && score <= 0.4 {
		tm.budget.RecordBorderlineScore(ctx)
	}

	tm.SetTrustScore(ctx, token, score)
	return score, sigs
}

func (tm *TrustManager) RecordRequestSignal(ctx context.Context, token, ip, subnet string) {
	tm.extractor.RecordRequest(ctx, token, ip, subnet)
}

func (tm *TrustManager) GetIPCount(ctx context.Context, ip string) int64 {
	if ip == "" {
		return 0
	}
	count, _ := tm.rdb.Client.Get(ctx, "sig:ip:"+ip).Int64()
	return count
}
