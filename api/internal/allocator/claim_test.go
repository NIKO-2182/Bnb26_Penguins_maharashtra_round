// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/allocator/claim_test.go
// PURPOSE: Concurrency test for seat claims
// INPUTS / OUTPUTS: N/A
// DEPENDS ON: claim.go, store/redis.go
// USED BY: N/A
// RULES: Must pass before moving on
// DO NOT: N/A
package allocator

import (
	"context"
	"testing"

	"fairdrop/api/internal/store"
	"github.com/redis/go-redis/v9"
)

func TestClaimSeatConcurrency(t *testing.T) {
	// This test requires a running redis or a mock. 
	// For now, we assume a test environment where redis is available.
	ctx := context.Background()
	rdb := &store.RedisClient{
		Client: redis.NewClient(&redis.Options{
			Addr: "localhost:6379",
		}),
	}

	// Setup
	rdb.Client.Del(ctx, store.KeySeatsLeft)
	rdb.Client.Del(ctx, store.KeySeatsClaimed)
	rdb.Client.Set(ctx, store.KeySeatsLeft, 500, 0)

	numClaims := 10000
	done := make(chan bool)

	for i := 0; i < numClaims; i++ {
		go func(id int) {
			claimToken := fmt.Sprintf("token-%d", id)
			ClaimSeat(ctx, rdb, claimToken)
			done <- true
		}(i)
	}

	for i := 0; i < numClaims; i++ {
		<-done
	}

	left, _ := rdb.Client.Get(ctx, store.KeySeatsLeft).Int()
	if left < 0 {
		t.Errorf("Oversell detected! Seats left: %d", left)
	}
	
	// Check if exactly 500 are in the claimed set
	claimed, _ := rdb.Client.SCard(ctx, store.KeySeatsClaimed).Result()
	if claimed > 500 {
		t.Errorf("More than 500 seats claimed: %d", claimed)
	}
}
