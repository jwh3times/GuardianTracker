package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// StateSigner issues and verifies stateless HMAC-signed OAuth state parameters.
//
// Unlike the previous in-memory store, signed states survive restarts and work
// with any number of replicas (B7) — there is no server-side state to lose.
// The tradeoff: a state token is no longer strictly single-use; it is replayable
// within its TTL. That is acceptable here because the Bungie authorization code
// it accompanies is single-use, and the previous store was not browser-bound
// either. Documented in SECURITY.md.
//
// Token format: "v1.<unix-ts>.<nonce>.<sig>" where nonce is 16 random bytes and
// sig = HMAC-SHA256(key, "v1.<unix-ts>.<nonce>"), both base64url (no padding).
type StateSigner struct {
	key []byte
}

// NewStateSigner derives a dedicated signing key from secret (the JWT secret)
// via HMAC with a fixed domain-separation label, so OAuth states can never be
// confused with anything else signed by the same secret.
func NewStateSigner(secret string) *StateSigner {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte("guardian-tracker/oauth-state/v1"))
	return &StateSigner{key: mac.Sum(nil)}
}

func (s *StateSigner) sign(payload string) string {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// Generate returns a fresh signed state token.
func (s *StateSigner) Generate() (string, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("state nonce: %w", err)
	}
	payload := fmt.Sprintf("v1.%d.%s", time.Now().Unix(), base64.RawURLEncoding.EncodeToString(nonce))
	return payload + "." + s.sign(payload), nil
}

// Verify checks the token's signature and that it was issued within ttl of now
// (allowing 60s of clock skew into the future).
func (s *StateSigner) Verify(state string, now time.Time, ttl time.Duration) bool {
	parts := strings.Split(state, ".")
	if len(parts) != 4 || parts[0] != "v1" {
		return false
	}
	payload := parts[0] + "." + parts[1] + "." + parts[2]
	if !hmac.Equal([]byte(s.sign(payload)), []byte(parts[3])) {
		return false
	}
	ts, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return false
	}
	issued := time.Unix(ts, 0)
	if issued.After(now.Add(60 * time.Second)) {
		return false
	}
	return now.Sub(issued) <= ttl
}
