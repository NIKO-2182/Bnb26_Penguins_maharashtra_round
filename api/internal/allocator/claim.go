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

func ClaimSeat(ctx context.Context, rdb *store.RedisClient, claimToken string) (*ClaimResult, error) {
	// Note: claimToken in this context refers to the unique identifier for the claim request
	// which we also store in the 'seats:claimed' set.
	res, err := rdb.Eval(`
		local seats_left = redis.call('get', 'seats:left')
		if not seats_left or tonumber(seats_left) <= 0 then
			return "sold_out"
		end

		if redis.call('sismember', 'seats:claimed', ARGV[1]) == 1 then
			return "duplicate_claim"
		end

		redis.call('decr', 'seats:left')
		redis.call('sadd', 'seats:claimed', ARGV[1])

		return "seat_granted"
	`, []string{claimToken})

	if err != nil {
		return nil, fmt.Errorf("lua error: %w", err)
	}

	return &ClaimResult{Status: res[0].(string)}, nil
}
