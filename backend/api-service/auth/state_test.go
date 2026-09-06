package auth

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

const stateTestTTL = 10 * time.Minute

func TestStateSigner_RoundTrip(t *testing.T) {
	s := NewStateSigner("test-jwt-secret-at-least-32-chars!!")
	state, nonce, err := s.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !s.Verify(state, nonce, time.Now(), stateTestTTL) {
		t.Error("freshly generated state failed verification")
	}
}

func TestStateSigner_UniqueStates(t *testing.T) {
	s := NewStateSigner("test-jwt-secret-at-least-32-chars!!")
	a, _, _ := s.Generate()
	b, _, _ := s.Generate()
	if a == b {
		t.Error("two generated states are identical (nonce not random?)")
	}
}

func TestStateSigner_Tampering(t *testing.T) {
	s := NewStateSigner("test-jwt-secret-at-least-32-chars!!")
	state, nonce, _ := s.Generate()
	parts := strings.Split(state, ".")

	cases := map[string]string{
		"tampered signature": strings.Join(parts[:4], ".") + ".AAAA" + parts[4][4:],
		"tampered timestamp": parts[0] + ".9999999999." + strings.Join(parts[2:], "."),
		"tampered nonce":     parts[0] + "." + parts[1] + ".AAAAAAAA." + strings.Join(parts[3:], "."),
		"tampered binding":   strings.Join(parts[:3], ".") + ".AAAA." + parts[4],
		"old version":        "v1." + strings.Join(parts[1:], "."),
		"missing segment":    strings.Join(parts[:4], "."),
		"extra segment":      state + ".extra",
		"empty string":       "",
		"garbage":            "not-a-state",
		"non-numeric ts":     parts[0] + ".abc." + strings.Join(parts[2:], "."),
	}
	for name, tampered := range cases {
		if s.Verify(tampered, nonce, time.Now(), stateTestTTL) {
			t.Errorf("%s verified successfully", name)
		}
	}
}

func TestStateSigner_WrongKey(t *testing.T) {
	a := NewStateSigner("secret-one-aaaaaaaaaaaaaaaaaaaaaaaa")
	b := NewStateSigner("secret-two-bbbbbbbbbbbbbbbbbbbbbbbb")
	state, nonce, _ := a.Generate()
	if b.Verify(state, nonce, time.Now(), stateTestTTL) {
		t.Error("state signed with a different key verified successfully")
	}
}

func TestStateSigner_Expiry(t *testing.T) {
	s := NewStateSigner("test-jwt-secret-at-least-32-chars!!")
	state, nonce, _ := s.Generate()

	// Within TTL passes; just past TTL fails.
	if !s.Verify(state, nonce, time.Now().Add(stateTestTTL-time.Second), stateTestTTL) {
		t.Error("state rejected just before expiry")
	}
	if s.Verify(state, nonce, time.Now().Add(stateTestTTL+time.Minute), stateTestTTL) {
		t.Error("expired state verified successfully")
	}
}

func TestStateSigner_FutureSkew(t *testing.T) {
	s := NewStateSigner("test-jwt-secret-at-least-32-chars!!")
	state, nonce, _ := s.Generate()

	// A token "from the future" is allowed up to 60s of clock skew.
	if !s.Verify(state, nonce, time.Now().Add(-30*time.Second), stateTestTTL) {
		t.Error("state rejected within the 60s skew allowance")
	}
	if s.Verify(state, nonce, time.Now().Add(-2*time.Minute), stateTestTTL) {
		t.Error("state from far in the future verified successfully")
	}
}

func TestStateSigner_BrowserBindingAndReplica(t *testing.T) {
	s := NewStateSigner(testSecret)
	state, nonce, err := s.Generate()
	if err != nil {
		t.Fatal(err)
	}
	_, otherNonce, err := s.Generate()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(state, nonce) {
		t.Fatal("state exposes the browser secret")
	}
	for _, presented := range []string{"", otherNonce, strings.Split(state, ".")[3]} {
		if s.Verify(state, presented, time.Now(), stateTestTTL) {
			t.Fatal("state accepted without matching browser secret")
		}
	}
	if !NewStateSigner(testSecret).Verify(state, nonce, time.Now(), stateTestTTL) {
		t.Fatal("matching browser failed on another replica")
	}
	legacy := "v1." + fmt.Sprint(time.Now().Unix()) + ".random-nonce"
	if s.Verify(legacy+"."+s.sign(legacy), nonce, time.Now(), stateTestTTL) {
		t.Fatal("legacy unbound state accepted")
	}
}
