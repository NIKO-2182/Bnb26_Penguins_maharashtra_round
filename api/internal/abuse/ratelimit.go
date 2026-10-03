// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/abuse/ratelimit.go
// PURPOSE: IP and Token based rate limiting
// INPUTS / OUTPUTS: N/A
// DEPENDS ON: store/redis.go
// USED BY: handlers/join.go, handlers/verify.go
// RULES: Use Redis to track requests
// DO NOT: N/A
package abuse

import (
	"context"
	"fairdrop/api/internal/config"
	"fairdrop/api/internal/store"
	"time"
)

type RateLimiter struct {
	rdb *store.RedisClient
	cfg *config.Config
}

func NewRateLimiter(rdb *store.RedisClient, cfg *config.Config) *RateLimiter {
	return &RateLimiter{rdb: rdb, cfg: cfg}
}

func (rl *RateLimiter) IsAllowed(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	fullKey := "rl:" + key
	count, err := rl.rdb.Client.Incr(ctx, fullKey).Result()
	if err != nil {
		return false, err
	}

	if count == 1 {
		rl.rdb.Client.Expire(ctx, fullKey, window)
	}

	// Increment Total Requests for RPS calculation
	rl.rdb.Client.Incr(ctx, "metrics:total_reqs").Err()

	if count > int64(limit) {
		rl.rdb.Client.Incr(ctx, "metrics:429s").Err()
		return false, nil
	}

	return true, nil
}
