package auth

import (
	"strings"
	"testing"
	"time"
)

const stateTestTTL = 10 * time.Minute

func TestStateSigner_RoundTrip(t *testing.T) {
	s := NewStateSigner("test-jwt-secret-at-least-32-chars!!")
	state, err := s.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !s.Verify(state, time.Now(), stateTestTTL) {
		t.Error("freshly generated state failed verification")
	}
}

func TestStateSigner_UniqueStates(t *testing.T) {
	s := NewStateSigner("test-jwt-secret-at-least-32-chars!!")
	a, _ := s.Generate()
	b, _ := s.Generate()
	if a == b {
		t.Error("two generated states are identical (nonce not random?)")
	}
}

func TestStateSigner_Tampering(t *testing.T) {
	s := NewStateSigner("test-jwt-secret-at-least-32-chars!!")
	state, _ := s.Generate()
	parts := strings.Split(state, ".")

	cases := map[string]string{
		"tampered signature":  parts[0] + "." + parts[1] + "." + parts[2] + ".AAAA" + parts[3][4:],
		"tampered timestamp":  parts[0] + ".9999999999." + parts[2] + "." + parts[3],
		"tampered nonce":      parts[0] + "." + parts[1] + ".AAAAAAAAAAAAAAAAAAAAAA." + parts[3],
		"wrong version":       "v2." + parts[1] + "." + parts[2] + "." + parts[3],
		"missing segment":     parts[0] + "." + parts[1] + "." + parts[2],
		"extra segment":       state + ".extra",
		"empty string":        "",
		"garbage":             "not-a-state",
		"non-numeric ts":      parts[0] + ".abc." + parts[2] + "." + parts[3],
	}
	for name, tampered := range cases {
		if s.Verify(tampered, time.Now(), stateTestTTL) {
			t.Errorf("%s verified successfully", name)
		}
	}
}

func TestStateSigner_WrongKey(t *testing.T) {
	a := NewStateSigner("secret-one-aaaaaaaaaaaaaaaaaaaaaaaa")
	b := NewStateSigner("secret-two-bbbbbbbbbbbbbbbbbbbbbbbb")
	state, _ := a.Generate()
	if b.Verify(state, time.Now(), stateTestTTL) {
		t.Error("state signed with a different key verified successfully")
	}
}

func TestStateSigner_Expiry(t *testing.T) {
	s := NewStateSigner("test-jwt-secret-at-least-32-chars!!")
	state, _ := s.Generate()

	// Within TTL passes; just past TTL fails.
	if !s.Verify(state, time.Now().Add(stateTestTTL-time.Second), stateTestTTL) {
		t.Error("state rejected just before expiry")
	}
	if s.Verify(state, time.Now().Add(stateTestTTL+time.Minute), stateTestTTL) {
		t.Error("expired state verified successfully")
	}
}

func TestStateSigner_FutureSkew(t *testing.T) {
	s := NewStateSigner("test-jwt-secret-at-least-32-chars!!")
	state, _ := s.Generate()

	// A token "from the future" is allowed up to 60s of clock skew.
	if !s.Verify(state, time.Now().Add(-30*time.Second), stateTestTTL) {
		t.Error("state rejected within the 60s skew allowance")
	}
	if s.Verify(state, time.Now().Add(-2*time.Minute), stateTestTTL) {
		t.Error("state from far in the future verified successfully")
	}
}
