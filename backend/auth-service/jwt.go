package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims represents the claims stored in the JWT token
type JWTClaims struct {
	UserID         string `json:"user_id"`
	DisplayName    string `json:"display_name"`
	MembershipID   string `json:"membership_id"`
	MembershipType int    `json:"membership_type"`
	Platform       string `json:"platform"`
	TokenType      string `json:"token_type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

// GenerateAccessToken creates a new JWT access token
func GenerateAccessToken(user *BungieUserProfile) (string, error) {
	// Get expiry duration from config (default 24 hours)
	expiryHours := 24
	if expiry := getEnv("JWT_EXPIRY_HOURS", "24"); expiry != "" {
		fmt.Sscanf(expiry, "%d", &expiryHours)
	}

	claims := JWTClaims{
		UserID:         user.MembershipID,
		DisplayName:    user.DisplayName,
		MembershipID:   user.MembershipID,
		MembershipType: user.MembershipType,
		Platform:       getPlatformName(user.MembershipType),
		TokenType:      "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * time.Duration(expiryHours))),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "guardian-tracker-auth-service",
			Subject:   user.MembershipID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(config.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign access token: %v", err)
	}

	return signedToken, nil
}

// GenerateRefreshToken creates a new JWT refresh token
func GenerateRefreshToken(user *BungieUserProfile) (string, error) {
	// Get expiry duration from config (default 30 days)
	expiryDays := 30
	if expiry := getEnv("JWT_REFRESH_EXPIRY_DAYS", "30"); expiry != "" {
		fmt.Sscanf(expiry, "%d", &expiryDays)
	}

	claims := JWTClaims{
		UserID:       user.MembershipID,
		MembershipID: user.MembershipID,
		TokenType:    "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24 * time.Duration(expiryDays))),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "guardian-tracker-auth-service",
			Subject:   user.MembershipID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(config.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign refresh token: %v", err)
	}

	return signedToken, nil
}

// ValidateToken validates and parses a JWT token
func ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(config.JWTSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %v", err)
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}

// ExtractTokenFromHeader extracts the bearer token from Authorization header
func ExtractTokenFromHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", fmt.Errorf("authorization header is empty")
	}

	// Expected format: "Bearer <token>"
	parts := strings.Fields(authHeader)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", fmt.Errorf("invalid authorization header format")
	}

	return parts[1], nil
}
