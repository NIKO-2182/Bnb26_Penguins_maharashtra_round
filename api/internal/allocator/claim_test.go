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
	"fmt"
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
	resetClaimState(t, ctx, rdb)
	rdb.Client.Set(ctx, store.KeySeatsLeft, 500, 0)

	numClaims := 10000
	done := make(chan bool)

	// Each claimer gets its own IP so the per-IP ceiling is not what is being
	// measured here -- this test exists to prove the Lua script cannot oversell
	// or double-grant under concurrency.
	for i := 0; i < numClaims; i++ {
		go func(id int) {
			claimToken := fmt.Sprintf("token-%d", id)
			ip := fmt.Sprintf("10.%d.%d.%d", (id/65536)%256, (id/256)%256, id%256)
			ClaimSeat(ctx, rdb, claimToken, 1.0, 0.0, ip, true)
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

// resetClaimState clears every key the allocator touches. The per-IP attempt
// counters persist for an hour by design, so without this a rerun inherits the
// previous run's budget and fails for the wrong reason.
func resetClaimState(t *testing.T, ctx context.Context, rdb *store.RedisClient, keys ...string) {
	t.Helper()
	all := []string{store.KeySeatsLeft, store.KeySeatsClaimed}
	for _, k := range keys {
		all = append(all, k)
	}
	// Sweep any leftover ipattempts:/attempts: keys from previous runs.
	iter := rdb.Client.Scan(ctx, 0, "ipattempts:*", 1000).Iterator()
	for iter.Next(ctx) {
		all = append(all, iter.Val())
	}
	iter2 := rdb.Client.Scan(ctx, 0, "attempts:*", 1000).Iterator()
	for iter2.Next(ctx) {
		all = append(all, iter2.Val())
	}
	rdb.Client.Del(ctx, all...)
}

// A legitimate retry must still be able to win. Linear decay (trust/n) made
// attempt 4 worth ~0.21 and rejected 608 real humans for trying again; the
// free allowance exists so an honest user keeps a fair shot at several draws.
func TestLegitimateRetryCanStillWin(t *testing.T) {
	ctx := context.Background()
	rdb := &store.RedisClient{
		Client: redis.NewClient(&redis.Options{Addr: "localhost:6379"}),
	}

	resetClaimState(t, ctx, rdb)
	rdb.Client.Set(ctx, store.KeySeatsLeft, 100, 0)

	// Trust 0.85, and force failure on the early draws by using a random value
	// above trust. Attempts within the free allowance must stay undiluted, so
	// they still fail here -- proving the draws are genuinely attempted rather
	// than silently blocked, and that the token survives to try again.
	for i := 1; i <= FreeAttempts; i++ {
		res, attempt, err := ClaimSeat(ctx, rdb, "retry-token", 0.85, 0.99, "172.31.5.5", true)
		if err != nil {
			t.Fatalf("claim %d: %v", i, err)
		}
		if attempt != i {
			t.Errorf("expected attempt %d, got %d", i, attempt)
		}
		if res.Status == "rejected_attempt_limit" {
			t.Fatalf("attempt %d inside the free allowance was hard-blocked", i)
		}
	}

	// Now win with a favourable draw.
	res, _, err := ClaimSeat(ctx, rdb, "retry-token", 0.85, 0.10, "172.31.5.5", true)
	if err != nil {
		t.Fatalf("final claim: %v", err)
	}
	if res.Status != "seat_granted" {
		t.Errorf("expected seat_granted on a favourable draw, got %s", res.Status)
	}
}

// Past the free allowance the draw must actually shrink, or the allowance is
// just an unbounded retry budget.
func TestRetryChanceDecaysAfterFreeAllowance(t *testing.T) {
	ctx := context.Background()
	rdb := &store.RedisClient{
		Client: redis.NewClient(&redis.Options{Addr: "localhost:6379"}),
	}

	resetClaimState(t, ctx, rdb)
	rdb.Client.Set(ctx, store.KeySeatsLeft, 100, 0)

	// Trust 0.85. Effective chance is trust/over where over = attempt-Free.
	// At attempt 5 over=1 so it is still undiluted; by attempt 8 over=4 the
	// chance is 0.85/4 = 0.2125, so a 0.5 draw must now lose.
	for i := 0; i < FreeAttempts+3; i++ {
		ClaimSeat(ctx, rdb, "decay-token", 0.85, 0.99, "172.31.6.6", true)
	}
	res, attempt, _ := ClaimSeat(ctx, rdb, "decay-token", 0.85, 0.5, "172.31.6.6", true)
	if attempt != FreeAttempts+4 {
		t.Fatalf("expected attempt %d, got %d", FreeAttempts+4, attempt)
	}
	if res.Status != "rejected_low_trust" {
		t.Errorf("expected a decayed draw to fail at 0.5 (effective ~0.21), got %s", res.Status)
	}
}

// The decay must bottom out at the floor rather than decaying to zero, so a
// persistent client is never permanently locked out.
func TestDecayFloorsInsteadOfVanishing(t *testing.T) {
	ctx := context.Background()
	rdb := &store.RedisClient{
		Client: redis.NewClient(&redis.Options{Addr: "localhost:6379"}),
	}

	resetClaimState(t, ctx, rdb)
	rdb.Client.Set(ctx, store.KeySeatsLeft, 100, 0)

	// A draw below the floor (0.15) at the very last permitted attempt.
	for i := 0; i < MaxDrawAttempts-1; i++ {
		ClaimSeat(ctx, rdb, "floor-token", 0.85, 0.99, "172.31.7.7", true)
	}
	res, attempt, _ := ClaimSeat(ctx, rdb, "floor-token", 0.85, 0.01, "172.31.7.7", true)
	if attempt != MaxDrawAttempts {
		t.Fatalf("expected final attempt %d, got %d", MaxDrawAttempts, attempt)
	}
	if res.Status != "seat_granted" {
		t.Errorf("a near-certain draw should still succeed at the floor, got %s", res.Status)
	}
}

// Sessions below the trust floor must not draw at all. Previously a 0.05-trust
// bot still won one seat roughly every 20 attempts, so a flood eventually landed
// seats purely through persistence.
func TestLowTrustCannotBruteForceSeats(t *testing.T) {
	ctx := context.Background()
	rdb := &store.RedisClient{
		Client: redis.NewClient(&redis.Options{Addr: "localhost:6379"}),
	}

	resetClaimState(t, ctx, rdb)
	rdb.Client.Set(ctx, store.KeySeatsLeft, 100, 0)

	granted := 0
	for i := 0; i < 500; i++ {
		// Fresh token every time (defeats per-session counters), low trust.
		res, _, err := ClaimSeat(ctx, rdb, fmt.Sprintf("lowtrust-%d", i), 0.10, 0.0, fmt.Sprintf("192.168.%d.%d", i/256, i%256), true)
		if err != nil {
			t.Fatalf("claim error: %v", err)
		}
		if res.Status == "seat_granted" {
			granted++
		}
	}

	if granted != 0 {
		t.Errorf("Below-floor trust must never win a seat, got %d", granted)
	}
}

// A single-IP flood must be capped even when it mints a fresh session per
// request, which is how bot_naive evades per-session attempt limits.
func TestSingleIPFloodIsCapped(t *testing.T) {
	ctx := context.Background()
	rdb := &store.RedisClient{
		Client: redis.NewClient(&redis.Options{Addr: "localhost:6379"}),
	}

	resetClaimState(t, ctx, rdb)
	rdb.Client.Set(ctx, store.KeySeatsLeft, 100, 0)

	granted := 0
	for i := 0; i < 2000; i++ {
		res, _, err := ClaimSeat(ctx, rdb, fmt.Sprintf("flood-%d", i), 0.95, 0.0, "203.0.113.1", true)
		if err != nil {
			t.Fatalf("claim error: %v", err)
		}
		if res.Status == "seat_granted" {
			granted++
		}
	}

	if granted > MaxIPDrawAttempts {
		t.Errorf("Single IP won %d seats, above the %d ceiling", granted, MaxIPDrawAttempts)
	}
}

// fcfs is the control arm: it must not inherit the fairness protections.
func TestFcfsBypassesCeilings(t *testing.T) {
	ctx := context.Background()
	rdb := &store.RedisClient{
		Client: redis.NewClient(&redis.Options{Addr: "localhost:6379"}),
	}

	resetClaimState(t, ctx, rdb)
	rdb.Client.Set(ctx, store.KeySeatsLeft, 50, 0)

	granted := 0
	// Low trust and way over the per-IP ceiling: fcfs should still hand seats.
	for i := 0; i < 300; i++ {
		res, _, err := ClaimSeat(ctx, rdb, fmt.Sprintf("fcfs-%d", i), 0.05, 0.0, "198.51.100.1", false)
		if err != nil {
			t.Fatalf("claim error: %v", err)
		}
		if res.Status == "seat_granted" {
			granted++
		}
	}

	if granted != 50 {
		t.Errorf("fcfs should sell exactly its 50 seats, got %d", granted)
	}
}
