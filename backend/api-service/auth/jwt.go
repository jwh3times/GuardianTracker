package auth

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWTClaims are the claims stored in every app-issued JWT.
type JWTClaims struct {
	// LegacyUserID preserves the historical user_id claim for already-issued
	// clients. Its value is a Destiny membership ID, never a users-row ID; new
	// code must use MembershipID instead.
	LegacyUserID   string `json:"user_id"`
	DisplayName    string `json:"display_name"`
	MembershipID   string `json:"membership_id"`
	MembershipType int    `json:"membership_type"`
	Platform       string `json:"platform"`
	TokenType      string `json:"token_type"`    // "access" or "refresh"
	TokenVersion   int    `json:"tver"`          // bumped on sign-out-everywhere to revoke all tokens
	SessionID      string `json:"sid,omitempty"` // per-device refresh session id (empty for pre-session tokens)
	jwt.RegisteredClaims
}

// JWT handles token generation and validation for the application.
type JWT struct {
	secret            string
	accessTTL         time.Duration
	refreshExpiryDays int
}

// NewJWT creates a JWT helper with the legacy hour-based access-token setting.
// New code should use NewJWTWithTTL so sub-hour lifetimes can be configured.
func NewJWT(secret string, expiryHours, refreshExpiryDays int) *JWT {
	return NewJWTWithTTL(secret, time.Duration(expiryHours)*time.Hour, refreshExpiryDays)
}

// NewJWTWithTTL creates a JWT helper with the given secret and access-token
// duration. Refresh-token lifetime remains configured in whole days.
func NewJWTWithTTL(secret string, accessTTL time.Duration, refreshExpiryDays int) *JWT {
	return &JWT{
		secret:            secret,
		accessTTL:         accessTTL,
		refreshExpiryDays: refreshExpiryDays,
	}
}

// GenerateAccessToken creates a signed JWT access token for the Destiny membership. sessionID
// binds the token to a per-device session so this-device logout can revoke it.
func (j *JWT) GenerateAccessToken(membership *DestinyMembership, tokenVersion int, sessionID string) (string, error) {
	claims := JWTClaims{
		LegacyUserID:   membership.MembershipID,
		DisplayName:    membership.DisplayName,
		MembershipID:   membership.MembershipID,
		MembershipType: membership.MembershipType,
		Platform:       GetPlatformName(membership.MembershipType),
		TokenType:      "access",
		TokenVersion:   tokenVersion,
		SessionID:      sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "guardian-tracker",
			Subject:   membership.MembershipID,
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(j.secret))
}

// GenerateRefreshToken creates a signed JWT refresh token for the Destiny membership, bound
// to sessionID. It also returns the token's jti so the caller can record it as the
// session's current refresh jti for reuse detection (compare-and-swap on rotation;
// see db.UserStore.RotateSession).
func (j *JWT) GenerateRefreshToken(membership *DestinyMembership, tokenVersion int, sessionID string) (token string, jti string, err error) {
	jti = uuid.NewString()
	claims := JWTClaims{
		LegacyUserID:   membership.MembershipID,
		DisplayName:    membership.DisplayName,
		MembershipID:   membership.MembershipID,
		MembershipType: membership.MembershipType,
		Platform:       GetPlatformName(membership.MembershipType),
		TokenType:      "refresh",
		TokenVersion:   tokenVersion,
		SessionID:      sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24 * time.Duration(j.refreshExpiryDays))),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "guardian-tracker",
			Subject:   membership.MembershipID,
		},
	}
	token, err = jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(j.secret))
	return token, jti, err
}

// RefreshTokenTTL is the lifetime of issued refresh tokens, used to set a session's
// expiry to match the refresh token it tracks.
func (j *JWT) RefreshTokenTTL() time.Duration {
	return time.Hour * 24 * time.Duration(j.refreshExpiryDays)
}

// ValidateToken parses and validates a JWT string, returning its claims.
func (j *JWT) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(j.secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}
	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}

// ExtractBearerToken extracts the token value from an "Authorization: Bearer <token>" header.
func ExtractBearerToken(authHeader string) string {
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return parts[1]
}

// GetPlatformName maps a Bungie membership type integer to a human-readable string.
func GetPlatformName(membershipType int) string {
	switch membershipType {
	case 1:
		return "xbox"
	case 2:
		return "psn"
	case 3:
		return "steam"
	case 4:
		return "blizzard"
	case 5:
		return "stadia"
	case 6:
		return "epic"
	default:
		return "unknown"
	}
}
