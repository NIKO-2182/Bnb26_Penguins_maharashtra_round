// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/session/token.go
// PURPOSE: HMAC-signed admission tokens carrying a non-repudiable queue position
// INPUTS / OUTPUTS: CreateTicket/ParseTicket/VerifyToken
// DEPENDS ON: N/A
// USED BY: session.go, handlers/join.go, handlers/claim.go
// RULES: Queue position must be signed; retry must never mint a new position
// DO NOT: Use standard Base64 containing '+' or '/' which break URL query params
package session

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

// Ticket is an admission ticket: WHO, WHEN they arrived, and their position in
// the admission sequence, all covered by one HMAC.
//
// The position is the point of the whole exercise. Retrying does NOT mint a
// new ticket -- the client keeps the one it was issued at /join -- so a retry
// flood cannot buy a better place in the queue. The signature covers the
// position, so a client cannot claim an earlier one by editing the string.
type Ticket struct {
	UserID     string `json:"user_id"`
	Seq        int64  `json:"seq"`    // monotonic admission sequence
	JoinedUnix int64  `json:"joined"` // seconds since epoch at admission
	Signature  string `json:"-"`
}

// CreateToken keeps the original userID:signature format for compatibility with
// existing clients and stored sessions.
func CreateToken(userID string, secret string) string {
	data := userID
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	signature := base64.RawURLEncoding.EncodeToString(h.Sum(nil))
	return fmt.Sprintf("%s:%s", data, signature)
}

func VerifyToken(token, secret string) (string, error) {
	parts := strings.Split(token, ":")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid token format")
	}
	userID := parts[0]
	signature := parts[len(parts)-1]

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(userID))
	expectedSignature := base64.RawURLEncoding.EncodeToString(h.Sum(nil))

	if signature != expectedSignature && signature != base64.StdEncoding.EncodeToString(h.Sum(nil)) {
		return "", fmt.Errorf("invalid signature")
	}
	return userID, nil
}

// admissionPayload is the exact string the HMAC covers for a ticket.
func admissionPayload(userID string, seq, joinedUnix int64) string {
	return fmt.Sprintf("%s|%d|%d", userID, seq, joinedUnix)
}

// CreateTicket mints a signed admission ticket at the given sequence position.
func CreateTicket(userID string, seq, joinedUnix int64, secret string) Ticket {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(admissionPayload(userID, seq, joinedUnix)))
	return Ticket{
		UserID:     userID,
		Seq:        seq,
		JoinedUnix: joinedUnix,
		Signature:  base64.RawURLEncoding.EncodeToString(h.Sum(nil)),
	}
}

// Encode serialises the ticket for transport to the client.
func (t Ticket) Encode() string {
	return fmt.Sprintf("%s|%d|%d|%s", t.UserID, t.Seq, t.JoinedUnix, t.Signature)
}

// ParseTicket decodes a ticket without verifying it. Call Verify before trusting
// any field.
func ParseTicket(s string) (Ticket, error) {
	parts := strings.Split(s, "|")
	if len(parts) != 4 {
		return Ticket{}, fmt.Errorf("invalid ticket format")
	}
	seq, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return Ticket{}, fmt.Errorf("invalid ticket seq: %w", err)
	}
	joined, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return Ticket{}, fmt.Errorf("invalid ticket timestamp: %w", err)
	}
	return Ticket{UserID: parts[0], Seq: seq, JoinedUnix: joined, Signature: parts[3]}, nil
}

// Verify checks the ticket's signature over its own contents.
func (t Ticket) Verify(secret string) error {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(admissionPayload(t.UserID, t.Seq, t.JoinedUnix)))
	expected := base64.RawURLEncoding.EncodeToString(h.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(t.Signature)) {
		return fmt.Errorf("invalid ticket signature")
	}
	return nil
}
