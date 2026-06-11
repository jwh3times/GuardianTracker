package bungie

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

// Client handles communication with the Bungie API.
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	limiter    *rate.Limiter
}

// NewClient creates a new Bungie API client with rate limiting.
func NewClient(apiKey, baseURL string, rps, burst int) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    baseURL,
		apiKey:     apiKey,
		limiter:    rate.NewLimiter(rate.Limit(rps), burst),
	}
}

// BungieError represents a structured error from the Bungie API.
type BungieError struct {
	ErrorCode       int
	ErrorStatus     string
	Message         string
	ThrottleSeconds int
}

func (e *BungieError) Error() string {
	return fmt.Sprintf("Bungie API error %d (%s): %s", e.ErrorCode, e.ErrorStatus, e.Message)
}

func (c *Client) doRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")
	return c.httpClient.Do(req)
}

func (c *Client) doRequestWithRetry(ctx context.Context, req *http.Request, maxRetries int) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		reqClone := req.Clone(ctx)
		resp, err := c.doRequest(ctx, reqClone)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt+1) * time.Second)
			continue
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			waitTime := time.Duration(attempt+1) * time.Second
			if s, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil {
				waitTime = time.Duration(s) * time.Second
			}
			time.Sleep(waitTime)
			lastErr = fmt.Errorf("rate limited by Bungie API")
			continue
		}
		if resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			time.Sleep(time.Duration(attempt+1) * time.Second)
			continue
		}
		return resp, nil
	}
	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

func parseResponse[T any](resp *http.Response) (*T, error) {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	var base BungieResponse
	if err := json.Unmarshal(body, &base); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	if base.ErrorCode != 1 {
		return nil, &BungieError{
			ErrorCode:       base.ErrorCode,
			ErrorStatus:     base.ErrorStatus,
			Message:         base.Message,
			ThrottleSeconds: base.ThrottleSeconds,
		}
	}
	var result T
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response data: %w", err)
	}
	return &result, nil
}

// GetManifest retrieves the current manifest metadata.
func (c *Client) GetManifest(ctx context.Context) (*ManifestResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/Destiny2/Manifest/", c.baseURL), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := c.doRequestWithRetry(ctx, req, 3)
	if err != nil {
		return nil, err
	}
	return parseResponse[ManifestResponse](resp)
}

// GetProfile retrieves a user's Destiny 2 profile for the specified components.
func (c *Client) GetProfile(ctx context.Context, membershipType int, membershipID, accessToken string, components []int) (*ProfileResponse, error) {
	compStrs := make([]string, len(components))
	for i, comp := range components {
		compStrs[i] = strconv.Itoa(comp)
	}
	url := fmt.Sprintf("%s/Destiny2/%d/Profile/%s/?components=%s", c.baseURL, membershipType, membershipID, strings.Join(compStrs, ","))
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	resp, err := c.doRequestWithRetry(ctx, req, 3)
	if err != nil {
		return nil, err
	}
	return parseResponse[ProfileResponse](resp)
}

// GetCharacters retrieves a user's Destiny 2 characters (component 200).
func (c *Client) GetCharacters(ctx context.Context, membershipType int, membershipID, accessToken string) (*CharactersResponse, error) {
	url := fmt.Sprintf("%s/Destiny2/%d/Profile/%s/?components=%d", c.baseURL, membershipType, membershipID, ComponentCharacters)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	resp, err := c.doRequestWithRetry(ctx, req, 3)
	if err != nil {
		return nil, err
	}
	return parseResponse[CharactersResponse](resp)
}

// GetPublicMilestones fetches current weekly milestone definitions (no auth needed).
func (c *Client) GetPublicMilestones(ctx context.Context) (map[string]PublicMilestone, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/Destiny2/Milestones/", c.baseURL), nil)
	if err != nil {
		return nil, fmt.Errorf("GetPublicMilestones: %w", err)
	}
	resp, err := c.doRequestWithRetry(ctx, req, 2)
	if err != nil {
		return nil, err
	}
	r, err := parseResponse[PublicMilestonesResponse](resp)
	if err != nil {
		return nil, err
	}
	return r.Response, nil
}

// GetCharacterVendors fetches vendor inventory for a specific character (requires auth; component 402).
func (c *Client) GetCharacterVendors(ctx context.Context, membershipType int, membershipID, characterID, accessToken string) (*CharacterVendorsResponse, error) {
	url := fmt.Sprintf("%s/Destiny2/%d/Profile/%s/Character/%s/Vendors/?components=%d",
		c.baseURL, membershipType, membershipID, characterID, ComponentVendorSales)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("GetCharacterVendors: %w", err)
	}
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	resp, err := c.doRequestWithRetry(ctx, req, 2)
	if err != nil {
		return nil, err
	}
	return parseResponse[CharacterVendorsResponse](resp)
}

// GetPublicVendors fetches the public vendor inventory (no auth needed; components 400+402).
func (c *Client) GetPublicVendors(ctx context.Context) (*PublicVendorsResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/Destiny2/Vendors/?components=400,402", c.baseURL), nil)
	if err != nil {
		return nil, fmt.Errorf("GetPublicVendors: %w", err)
	}
	resp, err := c.doRequestWithRetry(ctx, req, 2)
	if err != nil {
		return nil, err
	}
	return parseResponse[PublicVendorsResponse](resp)
}

// GetRecords fetches profile records (component 900) for a user.
func (c *Client) GetRecords(ctx context.Context, membershipType int, membershipID, accessToken string) (*RecordsProfileResponse, error) {
	url := fmt.Sprintf("%s/Destiny2/%d/Profile/%s/?components=%d", c.baseURL, membershipType, membershipID, ComponentRecords)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	resp, err := c.doRequestWithRetry(ctx, req, 3)
	if err != nil {
		return nil, err
	}
	return parseResponse[RecordsProfileResponse](resp)
}

// GetCommonSettings fetches Destiny 2 core settings (API-key only, no auth needed).
// The settings endpoint lives at /Platform/Settings/ (one level above /Platform/Destiny2/).
func (c *Client) GetCommonSettings(ctx context.Context) (*CoreSettings, error) {
	settingsURL := strings.TrimSuffix(c.baseURL, "/Destiny2") + "/Settings/"
	req, err := http.NewRequestWithContext(ctx, "GET", settingsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("GetCommonSettings: %w", err)
	}
	resp, err := c.doRequestWithRetry(ctx, req, 2)
	if err != nil {
		return nil, fmt.Errorf("GetCommonSettings: %w", err)
	}
	defer resp.Body.Close()
	var r CoreSettingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("GetCommonSettings decode: %w", err)
	}
	if r.ErrorCode != 1 {
		return nil, &BungieError{ErrorCode: r.ErrorCode, ErrorStatus: r.ErrorStatus, Message: r.Message}
	}
	s := r.Response.Destiny2CoreSettings
	return &CoreSettings{
		ActiveSealsRootNodeHash:     s.ActiveSealsRootNodeHash,
		LegacySealsRootNodeHash:     s.LegacySealsRootNodeHash,
		ExoticCatalystsRootNodeHash: s.ExoticCatalystsRootNodeHash,
		CraftingRootNodeHash:        s.CraftingRootNodeHash,
	}, nil
}

// DownloadFile downloads a file from the given URL into a byte slice.
func (c *Client) DownloadFile(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := c.doRequestWithRetry(ctx, req, 3)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	return data, nil
}

// Profile component constants.
const (
	ComponentProfiles             = 100
	ComponentCharacters           = 200
	ComponentCharacterInventories = 201
	ComponentCharacterEquipment   = 205
	ComponentItemInstances        = 300
	ComponentItemPerks            = 302
	ComponentItemStats            = 304
	ComponentItemSockets          = 305
	ComponentCollectibles         = 800
	ComponentRecords              = 900
	ComponentMetrics              = 1100
	ComponentVendorSales          = 402
)
