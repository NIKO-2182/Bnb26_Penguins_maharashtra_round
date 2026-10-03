// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/metrics/service.go
// PURPOSE: Logic for calculating metrics
// INPUTS / OUTPUTS: N/A
// DEPENDS ON: store/redis.go, session.go, ledger.go
// USED BY: httpx/router.go
// RULES: N/A
// DO NOT: N/A
package metrics

import (
	"context"
	"fairdrop/api/internal/config"
	"fairdrop/api/internal/ledger"
	"fairdrop/api/internal/store"
	"strconv"
)

type Service struct {
	rdb     *store.RedisClient
	cfg     *config.Config
	ledger  *ledger.Service
}

func NewService(rdb *store.RedisClient, cfg *config.Config) *Service {
	return &Service{
		rdb:    rdb,
		cfg:    cfg,
		ledger: ledger.NewService(rdb),
	}
}

func (s *Service) GetMetrics(ctx context.Context) (map[string]any, error) {
	// RPS: This is simplified. In a real app, we'd use a sliding window or a time-bucketed counter.
	// For now, we'll just return the total count as a placeholder or a "current" bucket if we had one.
	rpsRaw, _ := s.rdb.Client.Get(ctx, "metrics:total_reqs").Result()
	rps, _ := strconv.ParseFloat(rpsRaw, 64)

	// 429s
	count429Raw, _ := s.rdb.Client.Get(ctx, "metrics:429s").Result()
	count429, _ := strconv.ParseFloat(count429Raw, 64)

	// Pool Size: count of sessions with status "verified"
	// This requires a scan or a specific set. We'll use a simple count for now.
	poolSizeRaw, _ := s.rdb.Client.Get(ctx, "metrics:pool_size").Result()
	poolSize, _ := strconv.ParseFloat(poolSizeRaw, 64)

	// Seats Left
	grantedSeatsRaw, _ := s.rdb.Client.Get(ctx, "metrics:granted_seats").Result()
	grantedSeats, _ := strconv.ParseFloat(grantedSeatsRaw, 64)
	seatsLeft := s.cfg.TotalSeats - int(grantedSeats)

	// Fairness Metrics from Ledger Events
	events, _ := s.ledger.GetRecent(ctx, 5000)

	var tp, fp, fn, tn int64
	byReason := make(map[string]int64)
	byProfile := make(map[string]int64)

	for _, ev := range events {
		isRejected := (ev.ReasonCode == ledger.ReasonRejectedLowTrust ||
			ev.ReasonCode == ledger.ReasonRejectedRateLimitIp ||
			ev.ReasonCode == ledger.ReasonRejectedRateLimitTk ||
			ev.ReasonCode == ledger.ReasonRejectedRateLimitCooldown ||
			ev.ReasonCode == ledger.ReasonRejectedSubnetLimit ||
			ev.ReasonCode == ledger.ReasonRejectedPoWInvalid ||
			ev.ReasonCode == ledger.ReasonRejectedPoWTooFast)

		isGranted := (ev.ReasonCode == ledger.ReasonSeatGranted)

		if ev.UserType == "bot" {
			if isRejected {
				tp++
			} else if isGranted {
				fn++
			}
		} else if ev.UserType == "human" {
			if isGranted || ev.ReasonCode == ledger.ReasonAcceptedPool {
				tn++
			} else if isRejected {
				fp++
				byReason[ev.ReasonCode]++
				if ev.HumanProfile != "" {
					byProfile[ev.HumanProfile]++
				} else {
					byProfile["human_normal"]++
				}
			}
		}
	}

	precision := 1.0
	if tp+fp > 0 {
		precision = float64(tp) / float64(tp+fp)
	}

	recall := 1.0
	if tp+fn > 0 {
		recall = float64(tp) / float64(tp+fn)
	}

	humanFalseRejectionRate := 0.0
	if fp+tn > 0 {
		humanFalseRejectionRate = float64(fp) / float64(fp+tn)
	}

	botShare := 0.0
	if fn+tn > 0 {
		botShare = float64(fn) / float64(fn+tn)
	}

	return map[string]any{
		"rps":                        rps,
		"429s":                       count429,
		"seats_left":                 seatsLeft,
		"pool_size":                  poolSize,
		"tp":                         tp,
		"fp":                         fp,
		"fn":                         fn,
		"tn":                         tn,
		"precision":                  precision,
		"recall":                     recall,
		"human_false_rejection_rate": humanFalseRejectionRate,
		"bot_share":                  botShare,
		"by_reason":                  byReason,
		"by_profile":                 byProfile,
	}, nil
}
