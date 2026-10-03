// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/session/store.go
// PURPOSE: Session state management in Redis with JSON serialization
// INPUTS / OUTPUTS: N/A
// DEPENDS ON: store/redis.go
// USED BY: session.go
// RULES: Sessions must have a TTL and serialize/deserialize JSON correctly
// DO NOT: Hardcode empty/static session state
package session

import (
	"context"
	"encoding/json"
	"time"

	"fairdrop/api/internal/store"
)

func SaveSession(ctx context.Context, rdb *store.RedisClient, token string, sess *Session) {
	if sess.State == "" {
		sess.State = sess.Status
	}
	data, err := json.Marshal(sess)
	if err != nil {
		return
	}
	rdb.Client.Set(ctx, store.KeySession+token, data, 1*time.Hour)
}

func GetSession(ctx context.Context, rdb *store.RedisClient, token string) (*Session, error) {
	val, err := rdb.Client.Get(ctx, store.KeySession+token).Result()
	if err != nil {
		return nil, err
	}
	var sess Session
	if err := json.Unmarshal([]byte(val), &sess); err != nil {
		return nil, err
	}
	if sess.State == "" {
		sess.State = sess.Status
	}
	return &sess, nil
}
