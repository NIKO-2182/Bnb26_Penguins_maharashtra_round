// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/metrics/getall_test.go
// PURPOSE: Regression guard for GetAll losing events (it silently returned zero)
// INPUTS / OUTPUTS: Test results
// DEPENDS ON: ledger/store.go, metrics/fairness.go
// USED BY: go test
// RULES: GetAll must return every stored event in chronological order
// DO NOT: N/A
package ledger

import (
	"context"
	"testing"

	"fairdrop/api/internal/store"
	"github.com/redis/go-redis/v9"
)

// GetAllEvents previously used LRange(length-1, 0), which does NOT mean
// "reverse the list" in Redis -- it returned nothing. Every downstream fairness
// number was then computed over an empty ledger and reported 0 seats won.
func TestGetAllEventsReturnsEverything(t *testing.T) {
	ctx := context.Background()
	rdb := &store.RedisClient{Client: redis.NewClient(&redis.Options{Addr: "localhost:6379"})}
	s := NewStore(rdb)
	rdb.Client.Del(ctx, "ledger:events", ChainStateKey, ChainSeqKey)

	const n = 50
	for i := 0; i < n; i++ {
		if err := s.RecordEvent(ctx, Event{
			UserID:     "u",
			EventType:  EventTypeClaim,
			ReasonCode: ReasonSoldOut,
		}); err != nil {
			t.Fatalf("record: %v", err)
		}
	}

	length, _ := rdb.Client.LLen(ctx, "ledger:events").Result()
	if length != n {
		t.Fatalf("expected %d stored, got %d", n, length)
	}

	all, err := s.GetAllEvents(ctx)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(all) != n {
		t.Fatalf("GetAll returned %d events, expected %d -- events are being lost", len(all), n)
	}

	// Chronological order: ids ascend.
	for i := 1; i < len(all); i++ {
		if all[i].ID <= all[i-1].ID {
			t.Fatalf("events not in chronological order at %d: %d then %d", i, all[i-1].ID, all[i].ID)
		}
	}
	if all[0].ID != 1 {
		t.Errorf("expected the first chronological event to be id 1, got %d", all[0].ID)
	}
}

func TestGetAllEventsEmpty(t *testing.T) {
	ctx := context.Background()
	rdb := &store.RedisClient{Client: redis.NewClient(&redis.Options{Addr: "localhost:6379"})}
	s := NewStore(rdb)
	rdb.Client.Del(ctx, "ledger:events")

	all, err := s.GetAllEvents(ctx)
	if err != nil {
		t.Fatalf("GetAll on empty ledger: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("expected 0 events, got %d", len(all))
	}
}

// A single event must still come back -- the old range bug hid even n=1.
func TestGetAllEventsSingle(t *testing.T) {
	ctx := context.Background()
	rdb := &store.RedisClient{Client: redis.NewClient(&redis.Options{Addr: "localhost:6379"})}
	s := NewStore(rdb)
	rdb.Client.Del(ctx, "ledger:events", ChainStateKey, ChainSeqKey)

	if err := s.RecordEvent(ctx, Event{UserID: "solo", EventType: EventTypeClaim, ReasonCode: ReasonSeatGranted}); err != nil {
		t.Fatalf("record: %v", err)
	}
	all, err := s.GetAllEvents(ctx)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 event, got %d", len(all))
	}
	if all[0].UserID != "solo" || all[0].ReasonCode != ReasonSeatGranted {
		t.Errorf("event content wrong: %+v", all[0])
	}
}
