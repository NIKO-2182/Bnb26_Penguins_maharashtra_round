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
	"net"
	"time"
)

type RateLimiter struct {
	rdb *store.RedisClient
	cfg *config.Config
}

func NewRateLimiter(rdb *store.RedisClient, cfg *config.Config) *RateLimiter {
	return &RateLimiter{rdb: rdb, cfg: cfg}
}

func GetSubnet(ipStr string) string {
	host := ipStr
	if h, _, err := net.SplitHostPort(ipStr); err == nil {
		host = h
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return host
	}
	if ip4 := ip.To4(); ip4 != nil {
		return net.IPv4(ip4[0], ip4[1], ip4[2], 0).String() + "/24"
	}
	return ip.Mask(net.CIDRMask(64, 128)).String() + "/64"
}

func (rl *RateLimiter) CheckCooldown(ctx context.Context, userID string) bool {
	cooldownKey := "cooldown:" + userID
	exists, _ := rl.rdb.Client.Exists(ctx, cooldownKey).Result()
	return exists > 0
}

func (rl *RateLimiter) TriggerCooldown(ctx context.Context, userID string, duration time.Duration) {
	rl.rdb.Client.Set(ctx, "cooldown:"+userID, "1", duration)
}

func (rl *RateLimiter) TrackIPSession(ctx context.Context, ip, sessionToken string) int64 {
	key := "ip:sessions:" + ip
	rl.rdb.Client.SAdd(ctx, key, sessionToken)
	rl.rdb.Client.Expire(ctx, key, 10*time.Minute)
	count, _ := rl.rdb.Client.SCard(ctx, key).Result()
	return count
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
