// FILE: /Users/phodeco/Tech/hacks/BitNBuilds/api/internal/session/token.go
// PURPOSE: HMAC-signed token creation and verification
// INPUTS / OUTPUTS: N/A
// DEPENDS ON: N/A
// USED BY: session.go, handlers/join.go
// RULES: Tokens must be HMAC signed
// DO NOT: N/A
package session

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
)

func CreateToken(userID string, secret string) string {
	data := userID
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return fmt.Sprintf("%s:%s", data, signature)
}

func VerifyToken(token, secret string) (string, error) {
	parts := strings.Split(token, ":")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid token format")
	}
	userID := parts[0]
	signature := parts[1]

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(userID))
	expectedSignature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	if signature != expectedSignature {
		return "", fmt.Errorf("invalid signature")
	}
	return userID, nil
}
