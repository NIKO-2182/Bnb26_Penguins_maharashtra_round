// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/allocator/claim.go
// PURPOSE: Executes the claim logic via Lua
// INPUTS / OUTPUTS: N/A
// DEPENDS ON: store/redis.go
// USED BY: handlers/claim.go
// RULES: Must call Lua script for atomicity
// DO NOT: N/A
package allocator

import (
	"context"
	"fmt"

	"fairdrop/api/internal/store"
)

type ClaimResult struct {
	Status string
}

func ClaimSeat(ctx context.Context, rdb *store.RedisClient, claimToken string, trustScore float64, randomVal float64) (*ClaimResult, error) {
	res, err := rdb.Client.Eval(ctx, `
		local seats_left = redis.call('get', 'seats:left')
		if not seats_left or tonumber(seats_left) <= 0 then
			return "sold_out"
		end

		local claim_token = ARGV[1]
		local random_val = tonumber(ARGV[2])
		local trust_score = tonumber(ARGV[3])

		if redis.call('sismember', 'seats:claimed', claim_token) == 1 then
			return "duplicate_claim"
		end

		if random_val > trust_score then
			return "rejected_low_trust"
		end

		redis.call('decr', 'seats:left')
		redis.call('sadd', 'seats:claimed', claim_token)

		return "seat_granted"
	`, []string{}, claimToken, fmt.Sprintf("%.6f", randomVal), fmt.Sprintf("%.6f", trustScore)).Result()

	if err != nil {
		return nil, fmt.Errorf("lua error: %w", err)
	}

	statusStr, ok := res.(string)
	if !ok {
		return nil, fmt.Errorf("unexpected lua result type")
	}

	return &ClaimResult{Status: statusStr}, nil
}
