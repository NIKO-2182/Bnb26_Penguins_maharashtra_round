// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/ledger/chain_test.go
// PURPOSE: Verify the hash chain is append-only, atomic under concurrency, and tamper-evident
// INPUTS / OUTPUTS: Test results
// DEPENDS ON: store.go, event.go
// USED BY: go test
// RULES: Chain must start at genesis and verify end to end
// DO NOT: N/A
package ledger

import (
	"context"
	"sync"
	"testing"

	"fairdrop/api/internal/store"
	"github.com/redis/go-redis/v9"
)

func testStore(t *testing.T) (*Store, context.Context) {
	t.Helper()
	ctx := context.Background()
	rdb := &store.RedisClient{
		Client: redis.NewClient(&redis.Options{Addr: "localhost:6379"}),
	}
	s := NewStore(rdb)
	rdb.Client.Del(ctx, "ledger:events", ChainStateKey, ChainSeqKey)
	return s, ctx
}

func TestChainAppendsAndVerifies(t *testing.T) {
	s, ctx := testStore(t)

	for i := 0; i < 25; i++ {
		ev := Event{
			UserID:    "user-" + string(rune('a'+i%26)),
			UserType:  "human",
			EventType: EventTypeClaim,
			ReasonCode: ReasonSeatGranted,
			TrustScore: 0.85,
		}
		if err := s.RecordEvent(ctx, ev); err != nil {
			t.Fatalf("record %d: %v", i, err)
		}
	}

	v, err := s.VerifyChain(ctx, 0)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !v.OK {
		t.Fatalf("chain should be valid, got error: %s (broken at %d)", v.Error, v.BrokenAt)
	}
	if v.Checked != 25 {
		t.Errorf("expected 25 events checked, got %d", v.Checked)
	}
	if v.HeadHash == "" || v.HeadHash == GenesisHash {
		t.Errorf("head hash should be a real hash, got %q", v.HeadHash)
	}
}

// The canonical form is computed in Lua on write and in Go on verify. If the
// two ever disagree, this is the test that catches it.
func TestHashesAreReproducibleInGo(t *testing.T) {
	s, ctx := testStore(t)

	ev := Event{
		UserID:       "someone",
		UserType:     "human",
		HumanProfile: "human_normal",
		IP:           "203.0.113.9",
		Subnet:       "203.0.113.0/24",
		TrustScore:   0.85,
		EventType:    EventTypeClaim,
		ReasonCode:   ReasonSeatGranted,
		Metadata:     map[string]any{"attempt": 2, "trust_score": 0.85},
	}
	if err := s.RecordEvent(ctx, ev); err != nil {
		t.Fatalf("record: %v", err)
	}

	all, err := s.GetAllEvents(ctx)
	if err != nil || len(all) != 1 {
		t.Fatalf("expected 1 event, got %d (err %v)", len(all), err)
	}
	got := all[0]
	if recomputed := got.ComputeHash(); recomputed != got.Hash {
		t.Errorf("Go cannot reproduce the Lua hash:\n  stored:      %s\n  recomputed:  %s", got.Hash, recomputed)
	}
	if got.PrevHash != GenesisHash {
		t.Errorf("first event should point at genesis, got %q", got.PrevHash)
	}
}

// Concurrent appenders must produce a single unbroken chain. Without the atomic
// Lua append, racing writers would both read the same head and fork the chain.
func TestChainSurvivesConcurrentWriters(t *testing.T) {
	s, ctx := testStore(t)

	const n = 200
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_ = s.RecordEvent(ctx, Event{
				UserID:    "concurrent",
				UserType:  "bot",
				EventType: EventTypeClaim,
				ReasonCode: ReasonSoldOut,
				Metadata:  map[string]any{"id": id},
			})
		}(i)
	}
	wg.Wait()

	v, err := s.VerifyChain(ctx, 0)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !v.OK {
		t.Fatalf("chain forked under concurrency: %s at %d", v.Error, v.BrokenAt)
	}
	if v.Checked != n {
		t.Errorf("expected %d events, got %d", n, v.Checked)
	}
}

// Tampering with a stored event must be detectable.
func TestTamperingIsDetected(t *testing.T) {
	s, ctx := testStore(t)

	for i := 0; i < 10; i++ {
		_ = s.RecordEvent(ctx, Event{
			UserID:    "victim",
			EventType: EventTypeClaim,
			ReasonCode: ReasonRejectedRateLimitCooldown,
		})
	}

	// Confirm clean first.
	if v, _ := s.VerifyChain(ctx, 0); !v.OK {
		t.Fatalf("baseline chain invalid: %s", v.Error)
	}

	// Rewrite one event's reason code directly in Redis, as a tamperer would.
	raw, _ := s.rdb.Client.LRange(ctx, "ledger:events", 0, 0).Result()
	if len(raw) == 0 {
		t.Fatal("no events to tamper with")
	}
	tampered := replaceReason(raw[0], ReasonSeatGranted)
	s.rdb.Client.LSet(ctx, "ledger:events", 0, tampered)

	v, _ := s.VerifyChain(ctx, 0)
	if v.OK {
		t.Error("tampering was NOT detected -- chain verified clean after modifying an event")
	}
}

// replaceReason does a naive string swap, which is exactly what tampering looks like.
func replaceReason(s, newReason string) string {
	old := `"reason_code":"` + ReasonRejectedRateLimitCooldown + `"`
	return replaceAll(s, old, `"reason_code":"`+newReason+`"`)
}

func replaceAll(s, old, new string) string {
	out := ""
	for {
		i := indexOf(s, old)
		if i < 0 {
			return out + s
		}
		out += s[:i] + new
		s = s[i+len(old):]
	}
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}