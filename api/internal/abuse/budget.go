// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/abuse/budget.go
// PURPOSE: Label-free fairness controller adjusting trust scores based on borderline score ratios
// INPUTS / OUTPUTS: ShouldRelax returns bool when borderline score ratio is elevated
// DEPENDS ON: store/redis.go, config
// USED BY: trust.go
// RULES: Must maintain zero ground-truth header dependency
// DO NOT: Read X-Sim user type headers in decision code
package abuse

import (
	"context"
	"strconv"

	"fairdrop/api/internal/config"
	"fairdrop/api/internal/store"
)

type BudgetManager struct {
	rdb *store.RedisClient
	cfg *config.Config
}

func NewBudgetManager(rdb *store.RedisClient, cfg *config.Config) *BudgetManager {
	return &BudgetManager{rdb: rdb, cfg: cfg}
}

func (bm *BudgetManager) RecordBorderlineScore(ctx context.Context) {
	bm.rdb.Client.Incr(ctx, "budget:borderline_count")
	bm.rdb.Client.Expire(ctx, "budget:borderline_count", 600)
}

func (bm *BudgetManager) RecordVerifiedSession(ctx context.Context) {
	bm.rdb.Client.Incr(ctx, "budget:verified_count")
	bm.rdb.Client.Expire(ctx, "budget:verified_count", 600)
}

func (bm *BudgetManager) ShouldRelax(ctx context.Context) bool {
	borderlineRaw, _ := bm.rdb.Client.Get(ctx, "budget:borderline_count").Result()
	verifiedRaw, _ := bm.rdb.Client.Get(ctx, "budget:verified_count").Result()

	borderline, _ := strconv.ParseFloat(borderlineRaw, 64)
	verified, _ := strconv.ParseFloat(verifiedRaw, 64)

	if verified < 10 {
		return false
	}

	// If >15% of verified sessions have borderline trust scores (0.2-0.4), relax scoring
	return (borderline / verified) > 0.15
}
