// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/session/session.go
// PURPOSE: Session logic layer with machine-speed PoW bot detection
// INPUTS / OUTPUTS: N/A
// DEPENDS ON: token.go, store.go, abuse/ratelimit.go, abuse/pow.go, abuse/trust.go
// USED BY: handlers/join.go
// RULES: Penalize machine-speed PoW solves (<250ms) with low trust (0.05)
// DO NOT: Grant high trust to machine-speed bot solves

package session

import (
	"context"
	"errors"
	"fairdrop/api/internal/abuse"
	"fairdrop/api/internal/config"
	"fairdrop/api/internal/store"
	"time"
)

// QueueSeqKey holds the monotonic admission counter. Assigning the position
// here (not in the handler) keeps the counter next to the session store.
const QueueSeqKey = "queue:seq"

type Session struct {
	UserID    string    `json:"user_id"`
	Status    string    `json:"status"` // joined|verified|selected|claimed|not_selected|rejected
	State     string    `json:"state"`  // alias for status
	Trust     float64   `json:"trust"`
	PoWProof  string    `json:"pow_proof"`
	QueueSeq  int64     `json:"queue_seq,omitempty"`
	CreatedAt time.Time `json:"created_at"`
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
		UserID:    userID,
		Status:    "joined",
		State:     "joined",
		Trust:     0.1,
		CreatedAt: time.Now(),
	}
	SaveSession(ctx, m.rdb, token, sess)
	m.trust.SetTrustScore(ctx, token, 0.1)
	return token, nil
}

// Verify is the original scalar-only entry point, kept for compatibility.
// Callers that need the behavioural signal vector should use VerifyDetailed.
func (m *Manager) Verify(ctx context.Context, token, solution, ip, subnet string) error {
	_, _, err := m.VerifyDetailed(ctx, token, solution, ip, subnet)
	return err
}

// JoinWithTicket admits a user, assigning a monotonic queue position that is
// signed into the ticket.
//
// The position is assigned ONCE per user. A client that re-issues /join (a
// refresh, a page reload, a retry) must receive the SAME position it was first
// given -- minting a fresh one would let a retry flood buy better queue
// placement, which is precisely the volume advantage this design removes.
// Re-issue is also what makes a dropped connection recoverable.
func (m *Manager) JoinWithTicket(ctx context.Context, userID string) (string, Ticket, error) {
	allowed, _ := m.rateLimiter.IsAllowed(ctx, userID, 100, 1*time.Minute)
	if !allowed {
		return "", Ticket{}, errors.New("rate limit exceeded")
	}

	// Re-issue an existing ticket if this user already holds one. The mapping
	// user -> ticket is what makes the position sticky across retries.
	ticketKey := "ticket:user:" + userID
	if existing, err := m.rdb.Client.Get(ctx, ticketKey).Result(); err == nil && existing != "" {
		if tk, perr := ParseTicket(existing); perr == nil && tk.Verify(m.cfg.HMACSecret) == nil {
			return existing, tk, nil
		}
	}

	// Monotonic admission sequence. Redis INCR is atomic, so concurrent joins
	// cannot receive the same position.
	seq, err := m.rdb.Client.Incr(ctx, QueueSeqKey).Result()
	if err != nil {
		seq = int64(time.Now().UnixNano() % 1000000)
	}

	ticket := CreateTicket(userID, seq, time.Now().Unix(), m.cfg.HMACSecret)

	sess := &Session{
		UserID:    userID,
		Status:    "joined",
		State:     "joined",
		Trust:     0.1,
		QueueSeq:  seq,
		CreatedAt: time.Now(),
	}
	SaveSession(ctx, m.rdb, ticket.Encode(), sess)
	m.trust.SetTrustScore(ctx, ticket.Encode(), 0.1)
	// Persist so a retry of /join re-issues this exact position.
	m.rdb.Client.Set(ctx, ticketKey, ticket.Encode(), 2*time.Hour)

	return ticket.Encode(), ticket, nil
}

// VerifyDetailed runs Verify and also returns the trust score and the raw
// signal vector behind it, so the handler can persist an explainable feature
// row instead of only a scalar score.
func (m *Manager) VerifyDetailed(ctx context.Context, token, solution, ip, subnet string) (float64, abuse.Signals, error) {
	sess, err := GetSession(ctx, m.rdb, token)
	if err != nil {
		return 0, abuse.Signals{}, err
	}

	if !m.pow.Verify(solution, 4) {
		m.trust.SetTrustScore(ctx, token, 0.05)
		// Persist the failed-proof observation too: refusing to solve is itself a
		// strong behavioural signal, and dropping it hides exactly the sessions
		// a classifier most needs to learn from.
		sigs := m.trust.ExtractNow(ctx, token, ip, subnet, 0)
		m.trust.RecordFeatures(ctx, token, sess.UserID, ip, subnet, "verify_pow_fail", 0.05, sigs)
		return 0.05, sigs, errors.New("invalid proof")
	}

	solveTimeMs := int64(0)
	if !sess.CreatedAt.IsZero() {
		solveTimeMs = time.Since(sess.CreatedAt).Milliseconds()
	}

	sess.Status = "verified"
	sess.State = "verified"
	sess.PoWProof = solution

	trustScore, sigs := m.trust.EvaluateSessionTrust(ctx, token, ip, subnet, solveTimeMs)
	sess.Trust = trustScore

	SaveSession(ctx, m.rdb, token, sess)
	return trustScore, sigs, nil
}
