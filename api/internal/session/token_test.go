// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/session/token_test.go
// PURPOSE: Verify admission tickets cannot be forged or queue-inflated by retrying
// INPUTS / OUTPUTS: Test results
// DEPENDS ON: token.go
// USED BY: go test
// RULES: Queue position must be covered by the signature
// DO NOT: N/A
package session

import "testing"

const secret = "test-secret"

func TestTicketRoundTrip(t *testing.T) {
	tk := CreateTicket("alice", 42, 1700000000, secret)
	if err := tk.Verify(secret); err != nil {
		t.Fatalf("freshly minted ticket should verify: %v", err)
	}

	parsed, err := ParseTicket(tk.Encode())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.UserID != "alice" || parsed.Seq != 42 || parsed.JoinedUnix != 1700000000 {
		t.Errorf("round trip lost data: %+v", parsed)
	}
	if err := parsed.Verify(secret); err != nil {
		t.Errorf("parsed ticket should verify: %v", err)
	}
}

// The whole point of signing the position: a client must not be able to claim
// an earlier place in the queue by editing the string.
func TestQueuePositionCannotBeForged(t *testing.T) {
	tk := CreateTicket("mallory", 9999, 1700000000, secret)

	tk.Seq = 1 // try to jump to the front
	if err := tk.Verify(secret); err == nil {
		t.Error("a ticket with an edited queue position MUST fail verification")
	}

	tk.Seq = 9999
	tk.UserID = "victim"
	if err := tk.Verify(secret); err == nil {
		t.Error("a ticket with a swapped identity MUST fail verification")
	}

	tk.UserID = "mallory"
	tk.JoinedUnix = 1
	if err := tk.Verify(secret); err == nil {
		t.Error("a ticket with a backdated timestamp MUST fail verification")
	}
}

func TestTicketRejectsWrongSecret(t *testing.T) {
	tk := CreateTicket("alice", 1, 1700000000, secret)
	if err := tk.Verify("attacker-secret"); err == nil {
		t.Error("ticket must not verify under a different secret")
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	for _, bad := range []string{"", "nonsense", "a|b|c", "a|notanum|1|sig", "a|1|notanum|sig"} {
		if _, err := ParseTicket(bad); err == nil {
			t.Errorf("ParseTicket(%q) should have failed", bad)
		}
	}
}

// The same user re-joining must be representable, but the position is whatever
// was signed -- the point is that it cannot be chosen by the client.
func TestDistinctPositionsAreDistinctTickets(t *testing.T) {
	a := CreateTicket("alice", 1, 1700000000, secret)
	b := CreateTicket("alice", 2, 1700000001, secret)
	if a.Encode() == b.Encode() {
		t.Error("tickets with different positions must differ")
	}
	if err := a.Verify(secret); err != nil {
		t.Errorf("first ticket invalid: %v", err)
	}
	if err := b.Verify(secret); err != nil {
		t.Errorf("second ticket invalid: %v", err)
	}
}

// Legacy tokens must keep working so existing clients and stored sessions are
// not invalidated by the ticket format.
func TestLegacyTokenStillVerifies(t *testing.T) {
	legacy := CreateToken("bob", secret)
	userID, err := VerifyToken(legacy, secret)
	if err != nil {
		t.Fatalf("legacy token should verify: %v", err)
	}
	if userID != "bob" {
		t.Errorf("expected bob, got %q", userID)
	}
}

func TestLegacyTokenRejectsForgery(t *testing.T) {
	if _, err := VerifyToken("bob:forgedsignature", secret); err == nil {
		t.Error("forged legacy token must be rejected")
	}
	if _, err := VerifyToken("malformed", secret); err == nil {
		t.Error("malformed token must be rejected")
	}
}
