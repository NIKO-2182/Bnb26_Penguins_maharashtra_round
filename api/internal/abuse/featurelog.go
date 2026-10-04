// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/abuse/featurelog.go
// PURPOSE: Persist the behavioural feature vector per session so the ledger and offline analysis have real training data
// INPUTS / OUTPUTS: Appends a feature snapshot to "feat:<token>" and to a capped global list "feat:all"
// DEPENDS ON: store/redis.go, signals.go
// USED BY: handlers/verify.go, handlers/claim.go, httpx/router.go (export endpoint)
// RULES: MUST NOT include ground-truth labels (user_type) in decision-path data
// DO NOT: Read or store X-Sim ground-truth headers here
package abuse

import (
	"context"
	"encoding/json"
	"time"
)

// FeatureRecord is one observation of one session's behaviour. It is the unit
// of training data: features in, no label. Labels are attached later at export
// time from the ledger's evaluation-only user_type field, never here.
type FeatureRecord struct {
	UserID    string  `json:"user_id"`
	Token     string  `json:"token"`
	IP        string  `json:"ip"`
	Subnet    string  `json:"subnet"`
	Phase     string  `json:"phase"` // verify | claim
	Trust     float64 `json:"trust"`
	Timestamp int64   `json:"ts"`

	SolveTimeMs        int64   `json:"solve_time_ms"`
	RequestCount       int     `json:"request_count"`
	GapCount           int     `json:"gap_count"`
	TimingVariance     float64 `json:"timing_variance"`
	BurstCount         int     `json:"burst_count"`
	IPSessionCount     int64   `json:"ip_session_count"`
	SubnetSessionCount int64   `json:"subnet_session_count"`
	MeanGapMs          float64 `json:"mean_gap_ms"`
	MinGapMs           float64 `json:"min_gap_ms"`
	MaxGapMs           float64 `json:"max_gap_ms"`
	GapStdDevMs        float64 `json:"gap_stddev_ms"`
	CV                 float64 `json:"cv"`
	BurstRatio         float64 `json:"burst_ratio"`
	GapsPerSecond      float64 `json:"gaps_per_second"`
	SessionAgeMs       int64   `json:"session_age_ms"`
}

// featAllMax bounds the global feature list. Redis lists are unbounded and this
// one is written on every verify and claim, so it needs a ceiling the ledger
// itself does not.
const featAllMax = 200000

// RecordFeatures writes the feature vector for a session. The session-keyed
// list "feat:<token>" gives a per-session timeline; the capped global list
// "feat:all" is what the export endpoint streams for training.
func (tm *TrustManager) RecordFeatures(ctx context.Context, token, userID, ip, subnet, phase string, trust float64, sigs Signals) {
	rec := FeatureRecord{
		UserID:             userID,
		Token:              token,
		IP:                 ip,
		Subnet:             subnet,
		Phase:              phase,
		Trust:              trust,
		Timestamp:          time.Now().UnixMilli(),
		SolveTimeMs:        sigs.SolveTimeMs,
		RequestCount:       sigs.RequestCount,
		GapCount:           sigs.GapCount,
		TimingVariance:     sigs.TimingVariance,
		BurstCount:         sigs.BurstCount,
		IPSessionCount:     sigs.IPSessionCount,
		SubnetSessionCount: sigs.SubnetSessionCount,
		MeanGapMs:          sigs.MeanGapMs,
		MinGapMs:           sigs.MinGapMs,
		MaxGapMs:           sigs.MaxGapMs,
		GapStdDevMs:        sigs.GapStdDevMs,
		CV:                 sigs.CV,
		BurstRatio:         sigs.BurstRatio,
		GapsPerSecond:      sigs.GapsPerSecond,
		SessionAgeMs:       sigs.SessionAgeMs,
	}

	data, err := json.Marshal(rec)
	if err != nil {
		return
	}

	if token != "" {
		tm.rdb.Client.LPush(ctx, "feat:"+token, data)
		tm.rdb.Client.LTrim(ctx, "feat:"+token, 0, 49)
		tm.rdb.Client.Expire(ctx, "feat:"+token, 2*time.Hour)
	}

	tm.rdb.Client.LPush(ctx, "feat:all", data)
	tm.rdb.Client.LTrim(ctx, "feat:all", 0, featAllMax-1)
}

// ExtractNow recomputes the current signal vector for a session without changing
// trust. Used at claim time, where the behavioural picture has evolved since
// verify.
func (tm *TrustManager) ExtractNow(ctx context.Context, token, ip, subnet string, solveTimeMs int64) Signals {
	return tm.extractor.Extract(ctx, token, ip, subnet, solveTimeMs)
}

// ExportFeatureRecords reads the global feature list for offline training.
func (tm *TrustManager) ExportFeatureRecords(ctx context.Context, limit int64) ([]FeatureRecord, error) {
	if limit <= 0 {
		limit = featAllMax
	}
	raw, err := tm.rdb.Client.LRange(ctx, "feat:all", 0, limit-1).Result()
	if err != nil {
		return nil, err
	}
	out := make([]FeatureRecord, 0, len(raw))
	for _, r := range raw {
		var rec FeatureRecord
		if err := json.Unmarshal([]byte(r), &rec); err == nil {
			out = append(out, rec)
		}
	}
	return out, nil
}

// CountFeatureRecords reports how many observations are stored.
func (tm *TrustManager) CountFeatureRecords(ctx context.Context) int64 {
	n, _ := tm.rdb.Client.LLen(ctx, "feat:all").Result()
	return n
}