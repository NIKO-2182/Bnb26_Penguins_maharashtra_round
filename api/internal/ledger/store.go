// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/ledger/store.go
// PURPOSE: Persistence layer for the ledger
// INPUTS / OUTPUTS: N/A
// DEPENDS ON: store/redis.go, event.go
// USED BY: service.go
// RULES: N/A
// DO NOT: N/A
package ledger

import (
	"context"
	"encoding/json"
	"fairdrop/api/internal/store"
)

type Store struct {
	rdb *store.RedisClient
}

func NewStore(rdb *store.RedisClient) *Store {
	return &Store{rdb: rdb}
}

func (s *Store) RecordEvent(ctx context.Context, event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// Use a list to store events, or a sorted set if we want to query by timestamp
	key := "ledger:events"
	return s.rdb.Client.LPush(ctx, key, data).Err()
}

func (s *Store) GetRecentEvents(ctx context.Context, limit int64) ([]Event, error) {
	key := "ledger:events"
	results, err := s.rdb.Client.LRange(ctx, key, 0, limit-1).Result()
	if err != nil {
		return nil, err
	}

	events := make([]Event, 0, len(results))
	for _, res := range results {
		var event Event
		if err := json.Unmarshal([]byte(res), &event); err == nil {
			events = append(events, event)
		}
	}
	return events, nil
}
