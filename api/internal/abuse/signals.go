// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/abuse/signals.go
// PURPOSE: Feature extraction for trust scoring with sample count guards
// INPUTS / OUTPUTS: Extract returns Signals struct for session
// DEPENDS ON: store/redis.go
// USED BY: trust.go
// RULES: Require minimum sample count for variance and sustained burst checks
// DO NOT: Read X-Sim ground-truth headers
package abuse

import (
	"context"
	"math"
	"strconv"
	"time"

	"fairdrop/api/internal/store"
)

type Signals struct {
	SolveTimeMs        int64   `json:"solve_time_ms"`
	TimingVariance     float64 `json:"timing_variance"`
	GapCount           int     `json:"gap_count"`
	BurstCount         int     `json:"burst_count"`
	IPSessionCount     int64   `json:"ip_session_count"`
	SubnetSessionCount int64   `json:"subnet_session_count"`
}

type SignalExtractor struct {
	rdb *store.RedisClient
}

func NewSignalExtractor(rdb *store.RedisClient) *SignalExtractor {
	return &SignalExtractor{rdb: rdb}
}

func (s *SignalExtractor) RecordRequest(ctx context.Context, token, ip, subnet string) {
	nowMs := time.Now().UnixMilli()

	tsKey := "sig:ts:" + token
	s.rdb.Client.LPush(ctx, tsKey, nowMs)
	s.rdb.Client.LTrim(ctx, tsKey, 0, 19)
	s.rdb.Client.Expire(ctx, tsKey, 600)

	if ip != "" {
		s.rdb.Client.Incr(ctx, "sig:ip:"+ip)
		s.rdb.Client.Expire(ctx, "sig:ip:"+ip, 600)
	}
	if subnet != "" {
		s.rdb.Client.Incr(ctx, "sig:subnet:"+subnet)
		s.rdb.Client.Expire(ctx, "sig:subnet:"+subnet, 600)
	}
}

func (s *SignalExtractor) Extract(ctx context.Context, token, ip, subnet string, solveTimeMs int64) Signals {
	tsKey := "sig:ts:" + token
	rawTs, _ := s.rdb.Client.LRange(ctx, tsKey, 0, -1).Result()

	var gaps []float64
	var burstCount int
	if len(rawTs) >= 2 {
		for i := 0; i < len(rawTs)-1; i++ {
			t1, _ := strconv.ParseFloat(rawTs[i], 64)
			t2, _ := strconv.ParseFloat(rawTs[i+1], 64)
			gap := math.Abs(t1 - t2)
			gaps = append(gaps, gap)
			if gap < 100 { // burst click < 100ms
				burstCount++
			}
		}
	}

	variance := 0.0
	if len(gaps) >= 4 { // Minimum 4 gaps required for variance calculation
		var sum float64
		for _, g := range gaps {
			sum += g
		}
		mean := sum / float64(len(gaps))

		var sqDiffSum float64
		for _, g := range gaps {
			sqDiffSum += (g - mean) * (g - mean)
		}
		variance = sqDiffSum / float64(len(gaps))
	}

	ipCount, _ := s.rdb.Client.Get(ctx, "sig:ip:"+ip).Int64()
	subnetCount, _ := s.rdb.Client.Get(ctx, "sig:subnet:"+subnet).Int64()

	return Signals{
		SolveTimeMs:        solveTimeMs,
		TimingVariance:     variance,
		GapCount:           len(gaps),
		BurstCount:         burstCount,
		IPSessionCount:     ipCount,
		SubnetSessionCount: subnetCount,
	}
}
