// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/ledger/service.go
// PURPOSE: Business logic for ledger
// INPUTS / OUTPUTS: N/A
// DEPENDS ON: store.go, event.go
// USED BY: httpx/router.go
// RULES: N/A
// DO NOT: N/A
package ledger

import (
	"context"
	"fairdrop/api/internal/store"
	"time"
)

type Service struct {
	store *Store
}

func NewService(rdb *store.RedisClient) *Service {
	return &Service{store: NewStore(rdb)}
}

func (s *Service) RecordEvent(ctx context.Context, event Event) error {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	return s.store.RecordEvent(ctx, event)
}

func (s *Service) Record(ctx context.Context, userID string, eventType string, reasonCode string, metadata map[string]any) error {
	event := Event{
		Timestamp:  time.Now(),
		UserID:     userID,
		EventType:  eventType,
		ReasonCode: reasonCode,
		Metadata:   metadata,
	}
	return s.store.RecordEvent(ctx, event)
}

func (s *Service) GetRecent(ctx context.Context, limit int64) ([]Event, error) {
	return s.store.GetRecentEvents(ctx, limit)
}
