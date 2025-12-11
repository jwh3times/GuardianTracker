package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Client handles communication with the auth service
type Client struct {
	baseURL        string
	internalAPIKey string
	jwtSecret      string
	httpClient     *http.Client
}

// NewClient creates a new auth service client
func NewClient(baseURL, internalAPIKey, jwtSecret string) *Client {
	return &Client{
		baseURL:        strings.TrimSuffix(baseURL, "/"),
		internalAPIKey: internalAPIKey,
		jwtSecret:      jwtSecret,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// TokenResponse represents the response from the internal token endpoint
type TokenResponse struct {
	AccessToken  string `json:"accessToken"`
	MembershipID string `json:"membershipId"`
}

// JWTClaims represents the claims in the app JWT
type JWTClaims struct {
	MembershipID   string `json:"membership_id"`
	DisplayName    string `json:"display_name"`
	MembershipType int    `json:"membership_type"`
	Platform       string `json:"platform"`
	TokenType      string `json:"token_type"`
	jwt.RegisteredClaims
}

// ValidateAppJWT validates the application JWT and returns the claims
func (c *Client) ValidateAppJWT(tokenString string) (*JWTClaims, error) {
	if c.jwtSecret == "" {
		return nil, fmt.Errorf("JWT secret not configured")
	}

	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(c.jwtSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Verify it's an access token (not a refresh token)
	if claims.TokenType == "refresh" {
		return nil, fmt.Errorf("refresh token not allowed, use access token")
	}

	return claims, nil
}

// GetBungieToken fetches the Bungie OAuth token for a user from auth-service
func (c *Client) GetBungieToken(membershipID string) (string, error) {
	url := fmt.Sprintf("%s/internal/bungie-token/%s", c.baseURL, membershipID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Internal-API-Key", c.internalAPIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch token from auth service: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == 404 {
		return "", fmt.Errorf("no Bungie token found for user %s - user may need to re-authenticate", membershipID)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("auth service returned status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	return tokenResp.AccessToken, nil
}

// ExtractBearerToken extracts the token from an Authorization header
func ExtractBearerToken(authHeader string) string {
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}
	return parts[1]
}
