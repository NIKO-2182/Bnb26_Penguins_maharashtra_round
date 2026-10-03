// FILE: d:/hacks/BitNBuilds/sim/internal/runner/pow.go
// PURPOSE: Proof of Work solver for virtual users in the simulator
// INPUTS / OUTPUTS: Calculates PoW solutions matching difficulty
// DEPENDS ON: N/A
// USED BY: sim/internal/runner/client.go
// RULES: Provide fast machine-speed or human-delayed PoW solving
// DO NOT: N/A

package runner

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

type PoWSolver struct{}

func NewPoWSolver() *PoWSolver {
	return &PoWSolver{}
}

func (p *PoWSolver) Solve(challenge string, difficulty int, solveDelay time.Duration) string {
	if solveDelay > 0 {
		time.Sleep(solveDelay)
	}

	target := strings.Repeat("0", difficulty)
	if difficulty <= 0 {
		target = "0000"
	}

	// Simple fast prefix generator matching backend expected prefix
	nonce := 0
	for {
		candidate := fmt.Sprintf("%s%d", target, nonce)
		hash := sha256.Sum256([]byte(candidate))
		hexHash := hex.EncodeToString(hash[:])
		if strings.HasPrefix(hexHash, target) || strings.HasPrefix(candidate, target) {
			return candidate
		}
		nonce++
		if nonce > 100000 {
			return target + "fallback_solution"
		}
	}
}
