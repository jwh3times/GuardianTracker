package bungie

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"guardian-tracker/api-service/observability"

	"golang.org/x/time/rate"
)

// Client handles communication with the Bungie API.
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	limiter    *rate.Limiter

	// cdnBaseURL is the host for static content downloads (the manifest zip's
	// mobileWorldContentPaths are host-relative). Overridable for E2E.
	cdnBaseURL string
	// downloadClient has a long timeout for the multi-hundred-MB manifest zip —
	// the shared 30s API client would abort mid-download on slow egress.
	downloadClient *http.Client
}

// NewClient creates a new Bungie API client with rate limiting.
func NewClient(apiKey, baseURL string, rps, burst int) *Client {
	return &Client{
		httpClient:     &http.Client{Timeout: 30 * time.Second},
		baseURL:        baseURL,
		apiKey:         apiKey,
		limiter:        rate.NewLimiter(rate.Limit(rps), burst),
		cdnBaseURL:     "https://www.bungie.net",
		downloadClient: &http.Client{Timeout: 10 * time.Minute},
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
			if err := sleepCtx(ctx, time.Duration(attempt+1)*time.Second); err != nil {
				return nil, err
			}
			continue
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			waitTime := time.Duration(attempt+1) * time.Second
			if s, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil {
				waitTime = time.Duration(s) * time.Second
			}
			lastErr = fmt.Errorf("rate limited by Bungie API")
			if err := sleepCtx(ctx, waitTime); err != nil {
				return nil, err
			}
			continue
		}
		if resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			if err := sleepCtx(ctx, time.Duration(attempt+1)*time.Second); err != nil {
				return nil, err
			}
			continue
		}
		return resp, nil
	}
	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// sleepCtx waits for d, aborting early when the request context is cancelled
// (a disconnected client must not pin a handler in a retry backoff).
func sleepCtx(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
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

// GetCharacterVendors fetches vendor summaries and inventory for a specific
// character (requires auth; components 400 and 402).
func (c *Client) GetCharacterVendors(ctx context.Context, membershipType int, membershipID, characterID, accessToken string) (*CharacterVendorsResponse, error) {
	url := fmt.Sprintf("%s/Destiny2/%d/Profile/%s/Character/%s/Vendors/?components=%d,%d",
		c.baseURL, membershipType, membershipID, characterID, ComponentVendors, ComponentVendorSales)
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

// SetCDNBaseURL overrides the static-content host (E2E/fake-Bungie).
func (c *Client) SetCDNBaseURL(u string) {
	if u != "" {
		c.cdnBaseURL = strings.TrimSuffix(u, "/")
	}
}

// DownloadFileToPath streams a file to dest without buffering it in memory
// (the manifest zip is large). Simple 3-attempt retry; respects ctx.
func (c *Client) DownloadFileToPath(ctx context.Context, url, dest string) error {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		lastErr = c.downloadToPathOnce(ctx, url, dest)
		if lastErr == nil {
			return nil
		}
		observability.Logger(ctx).LogAttrs(ctx, slog.LevelWarn, "manifest download attempt failed",
			slog.Int("attempt", attempt+1),
			slog.Int("max_attempts", 3),
			observability.Err(lastErr),
		)
	}
	return lastErr
}

func (c *Client) downloadToPathOnce(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-API-Key", c.apiKey)
	resp, err := c.downloadClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", dest, err)
	}
	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", dest, err)
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		os.Remove(dest)
		return fmt.Errorf("failed to write %s: %w", dest, err)
	}
	return out.Close()
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
	ComponentVendors              = 400
	ComponentVendorSales          = 402
)
