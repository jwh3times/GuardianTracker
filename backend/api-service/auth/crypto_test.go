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
	tc, err := NewTokenCipher(validKey(t, 0), "")
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
	tc1, _ := NewTokenCipher(validKey(t, 0), "")
	tc2, _ := NewTokenCipher(validKey(t, 100), "")

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
	tc, _ := NewTokenCipher(validKey(t, 0), "")

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
	tc, _ := NewTokenCipher(validKey(t, 0), "")

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
	tcOld, err := NewTokenCipher(prevKeyStr, "")
	if err != nil {
		t.Fatalf("NewTokenCipher (old): %v", err)
	}
	plaintext := []byte("old-token-value")
	aad := []byte("member-id")
	blob, kv, err := tcOld.Encrypt(plaintext, aad)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	// kv from old cipher is 1; but the new cipher's current keyVersion is also 1,
	// so we simulate the previous-key path by using a different keyVersion.
	// We need to test the path where keyVersion != current.
	// Manually set kv to a value that triggers the previous-key path.
	oldKV := kv + 1 // Force mismatch so tc.previous is used.

	// New cipher has current=currKey and previous=prevKey.
	tcNew, err := NewTokenCipher(currKeyStr, prevKeyStr)
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
	got, err := tcNew.Decrypt(blob, aad, oldKV)
	if err != nil {
		t.Fatalf("Decrypt with previous key: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Errorf("previous-key round-trip mismatch: got %q, want %q", got, plaintext)
	}
}

func TestNewTokenCipher_EmptyKeyReturnsNil(t *testing.T) {
	tc, err := NewTokenCipher("", "")
	if err != nil {
		t.Errorf("expected no error for empty key, got: %v", err)
	}
	if tc != nil {
		t.Errorf("expected nil cipher for empty key, got non-nil")
	}
}
