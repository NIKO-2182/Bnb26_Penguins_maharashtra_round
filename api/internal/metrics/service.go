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

	// Fairness Metrics
	events, _ := s.ledger.GetRecent(ctx, 1000)
	_ = events
	
	// Placeholder values for now until we have better data
	return map[string]any{
		"rps":                    rps,
		"429s":                   count429,
		"seats_left":             seatsLeft,
		"pool_size":              poolSize,
		"bot_share":              0.0,
		"human_win_rate":         0.0,
		"expected_win_rate":      0.0,
		"human_false_rejection_rate": 0.0,
		"allocation_spread":      0.0,
	}, nil
}
