// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/abuse/trust.go
// PURPOSE: Trust scoring for users
// INPUTS / OUTPUTS: N/A
// DEPENDS ON: store/redis.go
// USED BY: session.go
// RULES: Trust is based on previous successful interactions
// DO NOT: N/A
package abuse

import (
	"context"
	"fairdrop/api/internal/config"
	"fairdrop/api/internal/store"
)

type TrustManager struct {
	rdb *store.RedisClient
	cfg *config.Config
}

func NewTrustManager(rdb *store.RedisClient, cfg *config.Config) *TrustManager {
	return &TrustManager{rdb: rdb, cfg: cfg}
}

func (tm *TrustManager) GetTrustScore(ctx context.Context, userID string) float64 {
	score, _ := tm.rdb.Client.Get(ctx, "trust:"+userID).Int64()
	if score == 0 {
		return 1.0
	}
	return float64(score) / 100.0
}

func (tm *TrustManager) RecordInteraction(ctx context.Context, userID string, success bool) {
	if success {
		tm.rdb.Client.Incr(ctx, "trust:"+userID)
	} else {
		tm.rdb.Client.Decr(ctx, "trust:"+userID)
	}
}
