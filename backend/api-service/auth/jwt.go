package auth

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims are the claims stored in every app-issued JWT.
type JWTClaims struct {
	UserID         string `json:"user_id"`
	DisplayName    string `json:"display_name"`
	MembershipID   string `json:"membership_id"`
	MembershipType int    `json:"membership_type"`
	Platform       string `json:"platform"`
	TokenType      string `json:"token_type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

// JWT handles token generation and validation for the application.
type JWT struct {
	secret            string
	expiryHours       int
	refreshExpiryDays int
}

// NewJWT creates a JWT helper with the given secret and expiry settings.
func NewJWT(secret string, expiryHours, refreshExpiryDays int) *JWT {
	return &JWT{
		secret:            secret,
		expiryHours:       expiryHours,
		refreshExpiryDays: refreshExpiryDays,
	}
}

// GenerateAccessToken creates a signed JWT access token for the given user.
func (j *JWT) GenerateAccessToken(user *BungieUserProfile) (string, error) {
	claims := JWTClaims{
		UserID:         user.MembershipID,
		DisplayName:    user.DisplayName,
		MembershipID:   user.MembershipID,
		MembershipType: user.MembershipType,
		Platform:       GetPlatformName(user.MembershipType),
		TokenType:      "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * time.Duration(j.expiryHours))),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "guardian-tracker",
			Subject:   user.MembershipID,
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(j.secret))
}

// GenerateRefreshToken creates a signed JWT refresh token for the given user.
func (j *JWT) GenerateRefreshToken(user *BungieUserProfile) (string, error) {
	claims := JWTClaims{
		UserID:         user.MembershipID,
		DisplayName:    user.DisplayName,
		MembershipID:   user.MembershipID,
		MembershipType: user.MembershipType,
		Platform:       GetPlatformName(user.MembershipType),
		TokenType:      "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24 * time.Duration(j.refreshExpiryDays))),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "guardian-tracker",
			Subject:   user.MembershipID,
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(j.secret))
}

// ValidateToken parses and validates a JWT string, returning its claims.
func (j *JWT) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
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
