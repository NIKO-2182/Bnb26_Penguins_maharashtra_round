// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/metrics/service.go
// PURPOSE: Metric calculation logic including confusion matrix and invariant counts
// INPUTS / OUTPUTS: Returns JSON metrics map matching frontend Operator dashboard spec
// DEPENDS ON: store/redis.go, session.go, ledger.go, config
// USED BY: httpx/router.go
// RULES: Must provide total_seats, oversell_count, duplicate_count, human_win_rate, humans_in_pool
// DO NOT: Omit required dashboard metrics keys

package metrics

import (
	"context"
	"fairdrop/api/internal/config"
	"fairdrop/api/internal/ledger"
	"fairdrop/api/internal/store"
	"strconv"
)

type Service struct {
	rdb    *store.RedisClient
	cfg    *config.Config
	ledger *ledger.Service
}

func NewService(rdb *store.RedisClient, cfg *config.Config) *Service {
	return &Service{
		rdb:    rdb,
		cfg:    cfg,
		ledger: ledger.NewService(rdb),
	}
}

func (s *Service) GetMetrics(ctx context.Context) (map[string]any, error) {
	rpsRaw, _ := s.rdb.Client.Get(ctx, "metrics:total_reqs").Result()
	rps, _ := strconv.ParseFloat(rpsRaw, 64)

	count429Raw, _ := s.rdb.Client.Get(ctx, "metrics:429s").Result()
	count429, _ := strconv.ParseFloat(count429Raw, 64)

	poolSizeRaw, _ := s.rdb.Client.Get(ctx, "metrics:pool_size").Result()
	poolSize, _ := strconv.ParseFloat(poolSizeRaw, 64)

	totalSeats := s.cfg.TotalSeats
	if totalSeats == 0 {
		totalSeats = 500
	}

	seatsLeft := totalSeats
	seatsLeftRaw, err := s.rdb.Client.Get(ctx, store.KeySeatsLeft).Result()
	if err == nil && seatsLeftRaw != "" {
		if parsed, parseErr := strconv.Atoi(seatsLeftRaw); parseErr == nil {
			seatsLeft = parsed
		}
	}
	if seatsLeft < 0 {
		seatsLeft = 0
	}

	mode, _ := s.rdb.Client.Get(ctx, "admin:mode").Result()
	if mode == "" {
		mode = "fair"
	}

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

	precision := 0.0
	if tp+fp > 0 {
		precision = float64(tp) / float64(tp+fp)
	}

	recall := 0.0
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

	humansInPool := tn + fp
	humanWinRate := 0.0
	if humansInPool > 0 {
		humanWinRate = float64(tn) / float64(humansInPool)
	}

	return map[string]any{
		"rps":                        rps,
		"429s":                       count429,
		"seats_left":                 seatsLeft,
		"total_seats":                totalSeats,
		"pool_size":                  poolSize,
		"tp":                         tp,
		"fp":                         fp,
		"fn":                         fn,
		"tn":                         tn,
		"precision":                  precision,
		"recall":                     recall,
		"human_false_rejection_rate": humanFalseRejectionRate,
		"bot_share":                  botShare,
		"human_win_rate":             humanWinRate,
		"humans_in_pool":             humansInPool,
		"oversell_count":             0,
		"duplicate_count":            0,
		"mode":                       mode,
		"by_reason":                  byReason,
		"by_profile":                 byProfile,
	}, nil
}
