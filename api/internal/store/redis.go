// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/store/redis.go
// PURPOSE: Redis client setup and key constants
// INPUTS / OUTPUTS: N/A
// DEPENDS ON: github.com/redis/go-redis/v9
// USED BY: allocator, session, abuse
// RULES: Single place for all keys
// DO NOT: N/A
package store

import (
	"context"
	"github.com/redis/go-redis/v9"
)

const (
	KeySeatsLeft     = "seats:left"
	KeySeatsClaimed  = "seats:claimed"
	KeySession       = "sess:"
	KeyPool          = "pool:"
	KeyRateLimitIp   = "rl:ip:"
	KeyRateLimitTok  = "rl:tok:"
)

type RedisClient struct {
	Client *redis.Client
}

func NewRedisClient(rawURL string) *RedisClient {
	opts, err := redis.ParseURL(rawURL)
	if err != nil {
		opts = &redis.Options{
			Addr: rawURL,
		}
	}
	rdb := redis.NewClient(opts)
	return &RedisClient{Client: rdb}
}

func (r *RedisClient) Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd {
	return r.Client.Eval(ctx, script, keys, args...)
}
