package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// TokenCipher encrypts/decrypts Bungie OAuth tokens using AES-256-GCM.
// It supports key rotation via an optional previous key (decrypt-only).
type TokenCipher struct {
	current            cipher.AEAD
	keyVersion         int16
	previous           cipher.AEAD // nil if no previous key
	previousKeyVersion int16
}

// NewTokenCipher constructs a TokenCipher from base64-encoded 32-byte keys.
// base64PrevKey may be empty (no rotation). Returns a nil cipher (not an error)
// when base64Key is empty — callers use nil to mean "no encryption configured".
func NewTokenCipher(base64Key string, keyVersion int16, base64PrevKey string, previousKeyVersion int16) (*TokenCipher, error) {
	if base64Key == "" {
		if base64PrevKey != "" || previousKeyVersion != 0 {
			return nil, fmt.Errorf("crypto: previous key requires a current key")
		}
		return nil, nil
	}
	if keyVersion <= 0 {
		return nil, fmt.Errorf("crypto: TOKEN_ENCRYPTION_KEY_VERSION must be a positive SMALLINT")
	}
	key, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return nil, fmt.Errorf("crypto: invalid TOKEN_ENCRYPTION_KEY (not base64): %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("crypto: TOKEN_ENCRYPTION_KEY must be 32 bytes (got %d)", len(key))
	}
	current, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	tc := &TokenCipher{current: current, keyVersion: keyVersion}
	if base64PrevKey != "" {
		if previousKeyVersion <= 0 {
			return nil, fmt.Errorf("crypto: TOKEN_ENCRYPTION_KEY_PREVIOUS_VERSION must be a positive SMALLINT")
		}
		if previousKeyVersion == keyVersion {
			return nil, fmt.Errorf("crypto: current and previous key versions must differ")
		}
		prevKey, err := base64.StdEncoding.DecodeString(base64PrevKey)
		if err != nil {
			return nil, fmt.Errorf("crypto: invalid TOKEN_ENCRYPTION_KEY_PREVIOUS: %w", err)
		}
		if len(prevKey) != 32 {
			return nil, fmt.Errorf("crypto: TOKEN_ENCRYPTION_KEY_PREVIOUS must be 32 bytes")
		}
		prev, err := newGCM(prevKey)
		if err != nil {
			return nil, err
		}
		tc.previous = prev
		tc.previousKeyVersion = previousKeyVersion
	} else if previousKeyVersion != 0 {
		return nil, fmt.Errorf("crypto: TOKEN_ENCRYPTION_KEY_PREVIOUS_VERSION requires TOKEN_ENCRYPTION_KEY_PREVIOUS")
	}
	return tc, nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: aes.NewCipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: cipher.NewGCM: %w", err)
	}
	return gcm, nil
}

// Encrypt encrypts plaintext with the current key. aad is the additional authenticated data
// (use membership_id as bytes to bind the ciphertext to its row).
// Returns nonce||ciphertext||tag and the current key version.
func (c *TokenCipher) Encrypt(plaintext, aad []byte) ([]byte, int16, error) {
	nonce := make([]byte, c.current.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, 0, fmt.Errorf("crypto: rand nonce: %w", err)
	}
	ct := c.current.Seal(nonce, nonce, plaintext, aad)
	return ct, c.keyVersion, nil
}

// Decrypt decrypts a blob produced by Encrypt. A ciphertext version must match
// the configured current or previous version exactly; unknown versions are
// rejected rather than being tried against the previous key.
func (c *TokenCipher) Decrypt(blob, aad []byte, keyVersion int16) ([]byte, error) {
	var aead cipher.AEAD
	switch keyVersion {
	case c.keyVersion:
		aead = c.current
	case c.previousKeyVersion:
		if c.previous == nil || c.previousKeyVersion <= 0 {
			return nil, fmt.Errorf("crypto: unknown key version %d", keyVersion)
		}
		aead = c.previous
	default:
		return nil, fmt.Errorf("crypto: unknown key version %d", keyVersion)
	}
	if len(blob) < aead.NonceSize() {
		return nil, fmt.Errorf("crypto: ciphertext too short")
	}
	nonce, ct := blob[:aead.NonceSize()], blob[aead.NonceSize():]
	plain, err := aead.Open(nil, nonce, ct, aad)
	if err != nil {
		return nil, fmt.Errorf("crypto: decrypt failed: %w", err)
	}
	return plain, nil
}
