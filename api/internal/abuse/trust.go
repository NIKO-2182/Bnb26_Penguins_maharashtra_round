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
// EvaluateSessionTrust scores a session AND returns the raw signal vector it
// scored from. The signals used to be computed, consumed, and discarded, which
// meant the ledger recorded only the resulting scalar -- leaving no feature
// matrix for offline analysis or a future classifier. Callers must persist the
// returned Signals; the score alone is not enough to explain or audit a decision.
func (tm *TrustManager) EvaluateSessionTrust(ctx context.Context, token, ip, subnet string, solveTimeMs int64) (float64, Signals) {
	sigs := tm.extractor.Extract(ctx, token, ip, subnet, solveTimeMs)

	// Base score starts high for PoW-verified sessions
	score := 0.85

	// Signal 1: PoW solve time penalty
	// Real humans take at least 350ms to solve PoW (typically 400ms-15000ms).
	// Machine solvers solve PoW in 5ms-100ms.
	if sigs.SolveTimeMs > 0 {
		if sigs.SolveTimeMs < 50 {
			score -= 0.60 // Extremely fast automated solver (5-10ms)
		} else if sigs.SolveTimeMs <= 150 {
			score -= 0.55 // Fast automated solver (e.g. 80-100ms)
		} else if sigs.SolveTimeMs < 300 {
			score -= 0.35 // Suspiciously fast solve
		} else if sigs.SolveTimeMs >= 350 && sigs.SolveTimeMs <= 15000 {
			score += 0.10 // Normal human solve time bonus
		}
	}

	// Signal 2: Timing variance and CV
	if sigs.GapCount >= 2 {
		if sigs.CV > 0.25 || sigs.TimingVariance > 300 {
			score += 0.05 // Human irregularity bonus
		} else if (sigs.CV > 0 && sigs.CV < 0.15) || (sigs.GapCount >= 4 && sigs.TimingVariance < 20) {
			score -= 0.20 // Uniform automated bot gap penalty
		}
	}

	// Signal 3: Sustained burst penalty (penalize bursts only when coupled with sub-human solve time or uniform bot gaps)
	if sigs.BurstCount > 3 && (sigs.SolveTimeMs < 300 || (sigs.CV > 0 && sigs.CV < 0.15) || (sigs.GapCount >= 4 && sigs.TimingVariance < 20)) {
		score -= 0.20
	}

	// Signal 4: Shared IP ceiling (supports shared_ip scenario: 200 humans / 5 IPs)
	if sigs.IPSessionCount > 150 {
		score -= 0.10 // Soft penalty for ultra-dense shared IP
	}

	// Signal 5: Label-free budget relaxation check (relax only for human-profile solve times)
	if tm.budget.ShouldRelax(ctx) && (sigs.SolveTimeMs >= 350 || sigs.SolveTimeMs == 0) {
		score += 0.10
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

// EvaluateClaimTrust performs a comprehensive claim-time evaluation using both PoW
// solve duration and human think time (click delay between verification and claim).
func (tm *TrustManager) EvaluateClaimTrust(ctx context.Context, token, ip, subnet string, solveTimeMs, clickDelayMs int64) (float64, Signals) {
	score, sigs := tm.EvaluateSessionTrust(ctx, token, ip, subnet, solveTimeMs)

	// Sub-human click reaction time: humans take at least 150ms to perceive and click.
	// Bots click in 10ms - 50ms.
	if clickDelayMs > 0 {
		if clickDelayMs < 80 {
			score -= 0.40 // Inhuman click speed
		} else if clickDelayMs >= 150 && solveTimeMs >= 350 {
			score += 0.05 // Human click think-time
		}
	}

	if score < 0.05 {
		score = 0.05
	}
	if score > 0.95 {
		score = 0.95
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
