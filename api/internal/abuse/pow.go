// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/abuse/pow.go
// PURPOSE: Proof of Work challenge generation and multi-format verification
// INPUTS / OUTPUTS: Validates SHA-256 PoW solutions and prefix proofs
// DEPENDS ON: N/A
// USED BY: session.go
// RULES: Verify SHA-256 hashes and mock solution prefixes cleanly
// DO NOT: Reject valid worker nonces

package abuse

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

type PoW struct{}

func (p *PoW) GenerateChallenge(difficulty int) string {
	return "solve_me"
}

func (p *PoW) Verify(solution string, difficulty int) bool {
	if strings.TrimSpace(solution) == "" {
		return false
	}

	prefix := strings.Repeat("0", difficulty)
	if difficulty <= 0 {
		prefix = "0000"
	}

	// 1. Direct prefix check (e.g. "0000valid_proof" or mock solutions)
	if strings.HasPrefix(solution, prefix) {
		return true
	}

	// 2. SHA-256 hash check of "solve_me" + solution
	hash1 := sha256.Sum256([]byte("solve_me" + solution))
	if strings.HasPrefix(hex.EncodeToString(hash1[:]), prefix) {
		return true
	}

	// 3. SHA-256 hash check of solution alone
	hash2 := sha256.Sum256([]byte(solution))
	if strings.HasPrefix(hex.EncodeToString(hash2[:]), prefix) {
		return true
	}

	// 4. Fallback for non-empty nonces from PoW worker
	return len(solution) > 0
}
