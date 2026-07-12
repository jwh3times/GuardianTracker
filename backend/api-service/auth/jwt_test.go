package auth

import (
	"strings"
	"testing"
	"time"
)

const jwtTestSecret = "test-jwt-secret-at-least-32-chars!!"

func testProfile() *BungieUserProfile {
	return &BungieUserProfile{
		MembershipID:   "4611686018467260757",
		DisplayName:    "TestGuardian",
		MembershipType: 3,
	}
}

func TestJWT_AccessTokenRoundTrip(t *testing.T) {
	j := NewJWT(jwtTestSecret, 24, 30)
	tok, err := j.GenerateAccessToken(testProfile(), 5, "sess-abc")
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	claims, err := j.ValidateToken(tok)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if claims.TokenType != "access" {
		t.Errorf("TokenType = %q, want access", claims.TokenType)
	}
	if claims.SessionID != "sess-abc" {
		t.Errorf("SessionID = %q, want sess-abc", claims.SessionID)
	}
	if claims.MembershipID != "4611686018467260757" {
		t.Errorf("MembershipID = %q", claims.MembershipID)
	}
	if claims.TokenVersion != 5 {
		t.Errorf("TokenVersion = %d, want 5", claims.TokenVersion)
	}
	if claims.Platform != "steam" {
		t.Errorf("Platform = %q, want steam", claims.Platform)
	}
	if claims.ID == "" {
		t.Error("jti claim missing")
	}
}

func TestJWT_AccessTokenUsesConfiguredTTL(t *testing.T) {
	const accessTTL = 30 * time.Minute
	j := NewJWTWithTTL(jwtTestSecret, accessTTL, 30)
	tok, err := j.GenerateAccessToken(testProfile(), 1, "sess-ttl")
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	claims, err := j.ValidateToken(tok)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	got := claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time)
	if got < accessTTL-time.Second || got > accessTTL+time.Second {
		t.Errorf("access token lifetime = %v, want approximately %v", got, accessTTL)
	}
}

func TestJWT_RefreshTokenType(t *testing.T) {
	j := NewJWT(jwtTestSecret, 24, 30)
	tok, jti, err := j.GenerateRefreshToken(testProfile(), 1, "sess-xyz")
	if err != nil {
		t.Fatalf("GenerateRefreshToken: %v", err)
	}
	if jti == "" {
		t.Error("GenerateRefreshToken returned empty jti")
	}
	claims, err := j.ValidateToken(tok)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if claims.TokenType != "refresh" {
		t.Errorf("TokenType = %q, want refresh", claims.TokenType)
	}
	if claims.ID != jti {
		t.Errorf("returned jti %q != token jti claim %q", jti, claims.ID)
	}
}

func TestJWT_WrongSecretRejected(t *testing.T) {
	a := NewJWT(jwtTestSecret, 24, 30)
	b := NewJWT("a-completely-different-secret-32ch!", 24, 30)
	tok, _ := a.GenerateAccessToken(testProfile(), 1, "")
	if _, err := b.ValidateToken(tok); err == nil {
		t.Fatal("token signed with a different secret validated successfully")
	}
}

func TestJWT_ExpiredTokenRejected(t *testing.T) {
	j := NewJWT(jwtTestSecret, -1, 30) // expiry one hour in the past
	tok, _ := j.GenerateAccessToken(testProfile(), 1, "")
	if _, err := j.ValidateToken(tok); err == nil {
		t.Fatal("expired token validated successfully")
	}
}

func TestJWT_TamperedTokenRejected(t *testing.T) {
	j := NewJWT(jwtTestSecret, 24, 30)
	tok, _ := j.GenerateAccessToken(testProfile(), 1, "")
	// Flip a character in the payload segment.
	parts := strings.Split(tok, ".")
	payload := []byte(parts[1])
	if payload[10] == 'A' {
		payload[10] = 'B'
	} else {
		payload[10] = 'A'
	}
	tampered := parts[0] + "." + string(payload) + "." + parts[2]
	if _, err := j.ValidateToken(tampered); err == nil {
		t.Fatal("tampered token validated successfully")
	}
}

func TestExtractBearerToken(t *testing.T) {
	cases := []struct {
		name   string
		header string
		want   string
	}{
		{"empty", "", ""},
		{"standard", "Bearer abc123", "abc123"},
		{"lowercase scheme", "bearer abc123", "abc123"},
		{"uppercase scheme", "BEARER abc123", "abc123"},
		{"wrong scheme", "Basic abc123", ""},
		{"no token", "Bearer", ""},
		{"no space", "Bearerabc123", ""},
		{"token with space", "Bearer abc 123", "abc 123"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExtractBearerToken(tc.header); got != tc.want {
				t.Errorf("ExtractBearerToken(%q) = %q, want %q", tc.header, got, tc.want)
			}
		})
	}
}

func TestGetPlatformName(t *testing.T) {
	cases := map[int]string{1: "xbox", 2: "psn", 3: "steam", 4: "blizzard", 5: "stadia", 6: "epic", 99: "unknown"}
	for in, want := range cases {
		if got := GetPlatformName(in); got != want {
			t.Errorf("GetPlatformName(%d) = %q, want %q", in, got, want)
		}
	}
}
