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

// StateSigner binds stateless signed OAuth state to an independent browser cookie.
// v2.<unix-ts>.<state-nonce>.<SHA256(cookie-nonce)>.<HMAC> survives replica changes
// without accepting a transaction initiated in a different browser.
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

// Generate returns a fresh signed state and an independent browser nonce. The
// nonce must be delivered only in the transaction cookie, never in JSON or URLs.
func (s *StateSigner) Generate() (state, browserNonce string, err error) {
	nonce := make([]byte, 64)
	if _, err := rand.Read(nonce); err != nil {
		return "", "", fmt.Errorf("state nonce: %w", err)
	}
	browserNonce = base64.RawURLEncoding.EncodeToString(nonce[32:])
	binding := sha256.Sum256([]byte(browserNonce))
	payload := fmt.Sprintf("v2.%d.%s.%s", time.Now().Unix(), base64.RawURLEncoding.EncodeToString(nonce[:32]), base64.RawURLEncoding.EncodeToString(binding[:]))
	return payload + "." + s.sign(payload), browserNonce, nil
}

// Verify checks the token's signature and that it was issued within ttl of now
// (allowing 60s of clock skew into the future).
func (s *StateSigner) Verify(state, browserNonce string, now time.Time, ttl time.Duration) bool {
	parts := strings.Split(state, ".")
	if len(parts) != 5 || parts[0] != "v2" || browserNonce == "" {
		return false
	}
	binding := sha256.Sum256([]byte(browserNonce))
	if !hmac.Equal([]byte(parts[3]), []byte(base64.RawURLEncoding.EncodeToString(binding[:]))) {
		return false
	}
	payload := strings.Join(parts[:4], ".")
	if !hmac.Equal([]byte(s.sign(payload)), []byte(parts[4])) {
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
