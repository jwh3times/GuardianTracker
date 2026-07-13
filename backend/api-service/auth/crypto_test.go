package auth

import (
	"bytes"
	"encoding/base64"
	"testing"
)

// validKey returns a base64-encoded 32-byte key for testing.
func validKey(t *testing.T, seed byte) string {
	t.Helper()
	key := make([]byte, 32)
	for i := range key {
		key[i] = seed + byte(i)
	}
	return base64.StdEncoding.EncodeToString(key)
}

func TestTokenCipher_RoundTrip(t *testing.T) {
	tc, err := NewTokenCipher(validKey(t, 0), 1, "", 0)
	if err != nil {
		t.Fatalf("NewTokenCipher: %v", err)
	}

	plaintext := []byte("my-bungie-access-token-abc123")
	aad := []byte("membership-123")

	blob, kv, err := tc.Encrypt(plaintext, aad)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	got, err := tc.Decrypt(blob, aad, kv)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Errorf("round-trip mismatch: got %q, want %q", got, plaintext)
	}
}

func TestTokenCipher_WrongKeyFails(t *testing.T) {
	tc1, _ := NewTokenCipher(validKey(t, 0), 1, "", 0)
	tc2, _ := NewTokenCipher(validKey(t, 100), 1, "", 0)

	plaintext := []byte("secret-token")
	aad := []byte("member-id")

	blob, kv, err := tc1.Encrypt(plaintext, aad)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	_, err = tc2.Decrypt(blob, aad, kv)
	if err == nil {
		t.Error("expected decrypt with wrong key to fail, but it succeeded")
	}
}

func TestTokenCipher_WrongAADFails(t *testing.T) {
	tc, _ := NewTokenCipher(validKey(t, 0), 1, "", 0)

	plaintext := []byte("secret-token")
	aad := []byte("correct-member-id")

	blob, kv, err := tc.Encrypt(plaintext, aad)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	_, err = tc.Decrypt(blob, []byte("wrong-member-id"), kv)
	if err == nil {
		t.Error("expected decrypt with wrong AAD to fail, but it succeeded")
	}
}

func TestTokenCipher_NonceUniqueness(t *testing.T) {
	tc, _ := NewTokenCipher(validKey(t, 0), 1, "", 0)

	plaintext := []byte("same-plaintext-every-time")
	aad := []byte("member-id")

	const n = 100
	blobs := make([][]byte, n)
	for i := range blobs {
		blob, _, err := tc.Encrypt(plaintext, aad)
		if err != nil {
			t.Fatalf("Encrypt[%d]: %v", i, err)
		}
		blobs[i] = blob
	}

	for i := range n {
		for j := i + 1; j < n; j++ {
			if bytes.Equal(blobs[i], blobs[j]) {
				t.Errorf("blobs[%d] == blobs[%d]: nonces are not unique", i, j)
			}
		}
	}
}

func TestTokenCipher_PreviousKeyDecrypt(t *testing.T) {
	prevKeyStr := validKey(t, 0)
	currKeyStr := validKey(t, 100)

	// Encrypt with the old key (simulate data that was written before rotation)
	tcOld, err := NewTokenCipher(prevKeyStr, 1, "", 0)
	if err != nil {
		t.Fatalf("NewTokenCipher (old): %v", err)
	}
	plaintext := []byte("old-token-value")
	aad := []byte("member-id")
	blob, kv, err := tcOld.Encrypt(plaintext, aad)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	// New cipher writes version 2 and decrypts only exact version-1 rows with the
	// explicitly configured previous key.
	tcNew, err := NewTokenCipher(currKeyStr, 2, prevKeyStr, 1)
	if err != nil {
		t.Fatalf("NewTokenCipher (new): %v", err)
	}

	// Encrypt something fresh with the current key.
	_, _, err = tcNew.Encrypt(plaintext, aad)
	if err != nil {
		t.Fatalf("Encrypt with new key: %v", err)
	}

	// Now decrypt the old blob using the previous-key path.
	// The old blob was encrypted with prevKey. tcNew.previous is also prevKey.
	got, err := tcNew.Decrypt(blob, aad, kv)
	if err != nil {
		t.Fatalf("Decrypt with previous key: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Errorf("previous-key round-trip mismatch: got %q, want %q", got, plaintext)
	}
}

func TestNewTokenCipher_EmptyKeyReturnsNil(t *testing.T) {
	tc, err := NewTokenCipher("", 0, "", 0)
	if err != nil {
		t.Errorf("expected no error for empty key, got: %v", err)
	}
	if tc != nil {
		t.Errorf("expected nil cipher for empty key, got non-nil")
	}
}

func TestTokenCipher_UsesConfiguredCurrentVersion(t *testing.T) {
	tc, err := NewTokenCipher(validKey(t, 0), 7, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	_, version, err := tc.Encrypt([]byte("token"), []byte("member"))
	if err != nil {
		t.Fatal(err)
	}
	if version != 7 {
		t.Fatalf("Encrypt version = %d, want 7", version)
	}
}

func TestTokenCipher_RejectsUnknownVersion(t *testing.T) {
	oldKey := validKey(t, 0)
	currentKey := validKey(t, 100)
	old, err := NewTokenCipher(oldKey, 1, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	blob, _, err := old.Encrypt([]byte("old-token"), []byte("member"))
	if err != nil {
		t.Fatal(err)
	}
	rotated, err := NewTokenCipher(currentKey, 2, oldKey, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rotated.Decrypt(blob, []byte("member"), 3); err == nil {
		t.Fatal("unknown version decrypted with the previous key")
	}
}

func TestNewTokenCipher_RejectsInvalidVersionPairs(t *testing.T) {
	key := validKey(t, 0)
	previous := validKey(t, 100)
	tests := []struct {
		name               string
		currentVersion     int16
		previousKey        string
		previousKeyVersion int16
	}{
		{name: "non-positive current", currentVersion: 0},
		{name: "missing previous version", currentVersion: 2, previousKey: previous},
		{name: "same versions", currentVersion: 2, previousKey: previous, previousKeyVersion: 2},
		{name: "version without previous key", currentVersion: 2, previousKeyVersion: 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewTokenCipher(key, tc.currentVersion, tc.previousKey, tc.previousKeyVersion); err == nil {
				t.Fatal("NewTokenCipher unexpectedly succeeded")
			}
		})
	}
}
