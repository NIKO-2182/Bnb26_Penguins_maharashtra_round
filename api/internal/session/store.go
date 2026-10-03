// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/session/store.go
// PURPOSE: Session state management in Redis
// INPUTS / OUTPUTS: N/A
// DEPENDS ON: store/redis.go
// USED BY: session.go
// RULES: Sessions must have a TTL
// DO NOT: N/A
package session

import (
	"context"
	"time"

	"fairdrop/api/internal/store"
)

func SaveSession(ctx context.Context, rdb *store.RedisClient, token string, sess *Session) {
	rdb.Client.Set(ctx, store.KeySession+token, sess, 1*time.Hour)
}

func GetSession(ctx context.Context, rdb *store.RedisClient, token string) (*Session, error) {
	_, err := rdb.Client.Get(ctx, store.KeySession+token).Result()
	if err != nil {
		return nil, err
	}
	return &Session{UserID: token, Status: "new"}, nil
}
