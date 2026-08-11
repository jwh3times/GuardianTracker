package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// bungieTokenURL is Bungie's OAuth token endpoint (overridable in tests).
const bungieTokenURL = "https://www.bungie.net/platform/app/oauth/token/"

// bungieDefaultRefreshTTL is how long a Bungie refresh token lasts when the
// token response omits refresh_expires_in. One rule, one place: it used to be
// written once in the login handler and once in TokenStore, where the two copies
// could drift with nothing to catch it.
const bungieDefaultRefreshTTL = 90 * 24 * time.Hour

// bungieOAuth talks to Bungie's OAuth token endpoint.
//
// One type for both grants, because the authorization-code exchange (login) and
// the refresh-token exchange (keeping a Bungie session alive) are the same
// request with a different grant_type and the same response parsing.
type bungieOAuth struct {
	clientID     string
	clientSecret string
	tokenURL     string
	client       *http.Client
}

func newBungieOAuth(clientID, clientSecret, tokenURL string) *bungieOAuth {
	if tokenURL == "" {
		tokenURL = bungieTokenURL
	}
	return &bungieOAuth{
		clientID:     clientID,
		clientSecret: clientSecret,
		tokenURL:     tokenURL,
		client:       &http.Client{Timeout: 30 * time.Second},
	}
}

// exchangeCode trades an authorization code from the OAuth callback for tokens.
func (o *bungieOAuth) exchangeCode(ctx context.Context, code string) (*BungieTokens, error) {
	return o.exchange(ctx, url.Values{
		"grant_type": {"authorization_code"},
		"code":       {code},
	})
}

// refreshTokens trades a Bungie refresh token for a fresh pair. Bungie rotates
// the refresh token too, so the result replaces both.
func (o *bungieOAuth) refreshTokens(ctx context.Context, refreshToken string) (*BungieTokens, error) {
	return o.exchange(ctx, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	})
}

// exchange posts a grant to the token endpoint and parses the response. The
// returned tokens carry no MembershipID — the caller knows which user this is.
//
// The response body is never included in an error: it is an OAuth error
// document on failure, but the same code path sees token material on success,
// and application logs must not carry credentials.
func (o *bungieOAuth) exchange(ctx context.Context, grant url.Values) (*BungieTokens, error) {
	grant.Set("client_id", o.clientID)
	if o.clientSecret != "" {
		grant.Set("client_secret", o.clientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.tokenURL, strings.NewReader(grant.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bungie %s grant failed with status %d", grant.Get("grant_type"), resp.StatusCode)
	}

	var tr struct {
		AccessToken      string `json:"access_token"`
		RefreshToken     string `json:"refresh_token"`
		ExpiresIn        int    `json:"expires_in"`
		RefreshExpiresIn int    `json:"refresh_expires_in"`
	}
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	refreshTTL := time.Duration(tr.RefreshExpiresIn) * time.Second
	if refreshTTL <= 0 {
		refreshTTL = bungieDefaultRefreshTTL
	}
	now := time.Now()
	return &BungieTokens{
		AccessToken:           tr.AccessToken,
		RefreshToken:          tr.RefreshToken,
		AccessTokenExpiresAt:  now.Add(time.Duration(tr.ExpiresIn) * time.Second),
		RefreshTokenExpiresAt: now.Add(refreshTTL),
	}, nil
}

// bungieProfile fetches the Destiny memberships for the freshly authorized user
// and picks the one the app should track.
func (o *bungieOAuth) bungieProfile(ctx context.Context, apiBaseURL, apiKey, accessToken string) (*BungieUserProfile, error) {
	reqURL := strings.TrimSuffix(apiBaseURL, "/") + "/User/GetMembershipsForCurrentUser/"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-API-Key", apiKey)

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("profile fetch failed with status %d", resp.StatusCode)
	}

	var apiResp struct {
		Response struct {
			DestinyMemberships []struct {
				MembershipType int    `json:"membershipType"`
				MembershipID   string `json:"membershipId"`
				DisplayName    string `json:"displayName"`
			} `json:"destinyMemberships"`
			PrimaryMembershipID string `json:"primaryMembershipId"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse profile response: %w", err)
	}
	if len(apiResp.Response.DestinyMemberships) == 0 {
		return nil, fmt.Errorf("no Destiny memberships found for user")
	}

	// Cross-save: prefer the primary membership when Bungie reports one;
	// DestinyMemberships[0] is otherwise arbitrary platform order.
	m := apiResp.Response.DestinyMemberships[0]
	if pid := apiResp.Response.PrimaryMembershipID; pid != "" {
		for _, dm := range apiResp.Response.DestinyMemberships {
			if dm.MembershipID == pid {
				m = dm
				break
			}
		}
	}
	return &BungieUserProfile{
		MembershipID:   m.MembershipID,
		DisplayName:    m.DisplayName,
		MembershipType: m.MembershipType,
	}, nil
}
