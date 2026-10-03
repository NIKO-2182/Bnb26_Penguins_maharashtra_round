// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/session/session.go
// PURPOSE: Session logic layer
// INPUTS / OUTPUTS: N/A
// DEPENDS ON: token.go, store.go, abuse/ratelimit.go, abuse/pow.go, abuse/trust.go
// USED BY: handlers/join.go
// RULES: N/A
// DO NOT: N/A
package session

import (
	"context"
	"errors"
	"fairdrop/api/internal/abuse"
	"fairdrop/api/internal/config"
	"fairdrop/api/internal/store"
	"time"
)

type Session struct {
	UserID   string
	Status   string // new|challenged|verified|pooled|selected|claimed|not_selected|rejected
	Trust    float64
	PoWProof string
}

type Manager struct {
	rdb         *store.RedisClient
	cfg         *config.Config
	rateLimiter *abuse.RateLimiter
	pow         *abuse.PoW
	trust       *abuse.TrustManager
}

func NewManager(rdb *store.RedisClient, cfg *config.Config) *Manager {
	return &Manager{
		rdb:         rdb,
		cfg:         cfg,
		rateLimiter: abuse.NewRateLimiter(rdb, cfg),
		pow:         &abuse.PoW{},
		trust:       abuse.NewTrustManager(rdb, cfg),
	}
}

func (m *Manager) Join(ctx context.Context, userID string) (string, error) {
	allowed, _ := m.rateLimiter.IsAllowed(ctx, userID, 100, 1*time.Minute)
	if !allowed {
		return "", errors.New("rate limit exceeded")
	}

	token := CreateToken(userID, m.cfg.HMACSecret)
	sess := &Session{
		UserID: userID,
		Status: "new",
		Trust:  m.trust.GetTrustScore(ctx, userID),
	}
	SaveSession(ctx, m.rdb, token, sess)
	return token, nil
}

func (m *Manager) Verify(ctx context.Context, token, solution string) error {
	sess, err := GetSession(ctx, m.rdb, token)
	if err != nil {
		return err
	}

	if !m.pow.Verify(solution, 4) {
		return errors.New("invalid proof")
	}

	sess.Status = "verified"
	SaveSession(ctx, m.rdb, token, sess)
	return nil
}
