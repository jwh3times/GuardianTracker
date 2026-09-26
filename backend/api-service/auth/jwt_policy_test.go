package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Correct signatures are deliberate: these tokens must fail the application
// policy, not merely signature verification, at both authentication entry points.
func TestJWT_ValidationPolicy(t *testing.T) {
	cases := []struct {
		name   string
		method jwt.SigningMethod
		mutate func(jwt.MapClaims)
	}{
		{name: "HS384", method: jwt.SigningMethodHS384},
		{name: "HS512", method: jwt.SigningMethodHS512},
		{name: "RS256", method: jwt.SigningMethodRS256},
		{name: "unsigned", method: jwt.SigningMethodNone},
		{name: "missing issuer", mutate: func(c jwt.MapClaims) { delete(c, "iss") }},
		{name: "empty issuer", mutate: func(c jwt.MapClaims) { c["iss"] = "" }},
		{name: "wrong issuer", mutate: func(c jwt.MapClaims) { c["iss"] = "another-app" }},
		{name: "missing expiration", mutate: func(c jwt.MapClaims) { delete(c, "exp") }},
		{name: "null expiration", mutate: func(c jwt.MapClaims) { c["exp"] = nil }},
		{name: "malformed expiration", mutate: func(c jwt.MapClaims) { c["exp"] = "tomorrow" }},
		{name: "expired", mutate: func(c jwt.MapClaims) { c["exp"] = time.Now().Add(-time.Hour).Unix() }},
		{name: "not yet valid", mutate: func(c jwt.MapClaims) { c["nbf"] = time.Now().Add(time.Hour).Unix() }},
	}
	for _, tc := range cases {
		for _, tokenType := range []string{"access", "refresh"} {
			t.Run(tc.name+"/"+tokenType, func(t *testing.T) {
				claims := jwt.MapClaims{
					"iss": "guardian-tracker", "sub": testMembership,
					"membership_id": testMembership, "membership_type": 3,
					"token_type": tokenType, "tver": 7, "sid": "sess-policy", "jti": "jti-policy",
					"iat": time.Now().Add(-time.Minute).Unix(), "nbf": time.Now().Add(-time.Minute).Unix(),
					"exp": time.Now().Add(time.Hour).Unix(),
				}
				if tc.mutate != nil {
					tc.mutate(claims)
				}
				method := tc.method
				if method == nil {
					method = jwt.SigningMethodHS256
				}
				var key any = []byte(testSecret)
				if method == jwt.SigningMethodNone {
					key = jwt.UnsafeAllowNoneSignatureType
				}
				if method == jwt.SigningMethodRS256 {
					privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
					if err != nil {
						t.Fatal(err)
					}
					key = privateKey
				}
				token, err := jwt.NewWithClaims(method, claims).SignedString(key)
				if err != nil {
					t.Fatal(err)
				}
				j := NewJWT(testSecret, 1, 30)
				if got, err := j.ValidateToken(token); err == nil || got != nil {
					t.Fatalf("invalid token accepted: claims=%v error=%v", got, err)
				}
				if tokenType == "access" {
					if w := doProtected(newMiddlewareRouter(j, nil), "Bearer "+token); w.Code != http.StatusUnauthorized {
						t.Fatalf("protected route status = %d, want 401", w.Code)
					}
				} else {
					store := newStubStore()
					issuer := newIssuer(t, store, nil)
					session, err := issuer.Refresh(context.Background(), token, "ua")
					if session != nil || reasonOf(t, err) != ReasonInvalidToken {
						t.Fatalf("invalid refresh issued session=%v error=%v", session, err)
					}
					if store.createCalls != 0 || store.rotateCalls != 0 {
						t.Fatal("invalid refresh changed session state")
					}
				}
			})
		}
	}
}

func TestJWT_SessionlessTokensRoundTrip(t *testing.T) {
	j := NewJWT(jwtTestSecret, 1, 30)
	access, err := j.GenerateAccessToken(testDestinyMembership(), 0, "")
	if err != nil {
		t.Fatal(err)
	}
	refresh, _, err := j.GenerateRefreshToken(testDestinyMembership(), 0, "")
	if err != nil {
		t.Fatal(err)
	}
	for kind, token := range map[string]string{"access": access, "refresh": refresh} {
		t.Run(kind, func(t *testing.T) {
			claims, err := j.ValidateToken(token)
			if err != nil {
				t.Fatal(err)
			}
			if claims.SessionID != "" || claims.TokenType != kind || claims.TokenVersion != 0 {
				t.Fatalf("unexpected session-less claims: %+v", claims)
			}
		})
	}
}

func TestJWT_SessionlessAccessStillAuthorizesDegradedMode(t *testing.T) {
	j := NewJWT(jwtTestSecret, 1, 30)
	token, err := j.GenerateAccessToken(testDestinyMembership(), 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if w := doProtected(newMiddlewareRouter(j, nil), "Bearer "+token); w.Code != http.StatusOK {
		t.Fatalf("session-less access status = %d, want 200", w.Code)
	}
}
