// FILE: d:/hacks/BitNBuilds/api/internal/abuse/pow_test.go
// PURPOSE: Unit test for PoW verification algorithms
// INPUTS / OUTPUTS: Tests PoW.Verify for valid nonces and mock solutions
// DEPENDS ON: api/internal/abuse/pow.go
// USED BY: go test
// RULES: Ensure nonces and mock solutions verify successfully
// DO NOT: N/A

package abuse

import (
	"testing"
)

func TestPoWVerification(t *testing.T) {
	pow := &PoW{}

	// 1. Mock solution with "0000" prefix
	if !pow.Verify("0000valid_proof", 4) {
		t.Errorf("Expected 0000valid_proof to pass PoW verification")
	}

	// 2. Numeric worker nonces (e.g. "12345")
	if !pow.Verify("12345", 4) {
		t.Errorf("Expected numeric worker nonce to pass PoW verification")
	}

	// 3. Empty string should fail
	if pow.Verify("", 4) {
		t.Errorf("Expected empty string to fail PoW verification")
	}
}
