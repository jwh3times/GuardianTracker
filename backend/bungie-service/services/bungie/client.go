package bungie

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/time/rate"
)

// Client handles communication with the Bungie API
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	limiter    *rate.Limiter
}

// NewClient creates a new Bungie API client
func NewClient(apiKey, baseURL string, rps int, burst int) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
		apiKey:  apiKey,
		limiter: rate.NewLimiter(rate.Limit(rps), burst),
	}
}

// BungieError represents an error from the Bungie API
type BungieError struct {
	ErrorCode       int
	ErrorStatus     string
	Message         string
	ThrottleSeconds int
}

func (e *BungieError) Error() string {
	return fmt.Sprintf("Bungie API error %d (%s): %s", e.ErrorCode, e.ErrorStatus, e.Message)
}

// doRequest performs an HTTP request with rate limiting and error handling
func (c *Client) doRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
	// Wait for rate limiter
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	// Add required headers
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}

	return resp, nil
}

// doRequestWithRetry performs a request with retry on transient failures
func (c *Client) doRequestWithRetry(ctx context.Context, req *http.Request, maxRetries int) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Clone request for retry (body needs to be re-readable)
		reqClone := req.Clone(ctx)

		resp, err := c.doRequest(ctx, reqClone)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt+1) * time.Second)
			continue
		}

		// Handle rate limiting (429)
		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			retryAfter := resp.Header.Get("Retry-After")
			waitTime := time.Duration(attempt+1) * time.Second
			if seconds, err := strconv.Atoi(retryAfter); err == nil {
				waitTime = time.Duration(seconds) * time.Second
			}
			time.Sleep(waitTime)
			lastErr = fmt.Errorf("rate limited by Bungie API")
			continue
		}

		// Handle server errors (5xx)
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

// parseResponse reads and parses a Bungie API response
func parseResponse[T any](resp *http.Response) (*T, error) {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// First check for Bungie API errors
	var baseResp BungieResponse
	if err := json.Unmarshal(body, &baseResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API-level errors
	if baseResp.ErrorCode != 1 { // 1 = Success
		return nil, &BungieError{
			ErrorCode:       baseResp.ErrorCode,
			ErrorStatus:     baseResp.ErrorStatus,
			Message:         baseResp.Message,
			ThrottleSeconds: baseResp.ThrottleSeconds,
		}
	}

	// Parse the full response
	var result T
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response data: %w", err)
	}

	return &result, nil
}

// GetManifest retrieves the current manifest metadata
func (c *Client) GetManifest(ctx context.Context) (*ManifestResponse, error) {
	url := fmt.Sprintf("%s/Destiny2/Manifest/", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.doRequestWithRetry(ctx, req, 3)
	if err != nil {
		return nil, err
	}

	return parseResponse[ManifestResponse](resp)
}

// GetProfile retrieves a user's Destiny 2 profile with specified components
func (c *Client) GetProfile(ctx context.Context, membershipType int, membershipID string, accessToken string, components []int) (*ProfileResponse, error) {
	// Build components query param
	componentsStr := ""
	for i, comp := range components {
		if i > 0 {
			componentsStr += ","
		}
		componentsStr += strconv.Itoa(comp)
	}

	url := fmt.Sprintf("%s/Destiny2/%d/Profile/%s/?components=%s",
		c.baseURL, membershipType, membershipID, componentsStr)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add OAuth token for user-specific data
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	resp, err := c.doRequestWithRetry(ctx, req, 3)
	if err != nil {
		return nil, err
	}

	return parseResponse[ProfileResponse](resp)
}

// GetCharacters retrieves a user's Destiny 2 characters (component 200)
func (c *Client) GetCharacters(ctx context.Context, membershipType int, membershipID string, accessToken string) (*CharactersResponse, error) {
	url := fmt.Sprintf("%s/Destiny2/%d/Profile/%s/?components=%d",
		c.baseURL, membershipType, membershipID, ComponentCharacters)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add OAuth token for user-specific data
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	resp, err := c.doRequestWithRetry(ctx, req, 3)
	if err != nil {
		return nil, err
	}

	return parseResponse[CharactersResponse](resp)
}

// DownloadFile downloads a file from the given URL to a byte slice
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

// Profile component constants
const (
	ComponentProfiles            = 100
	ComponentCharacters          = 200
	ComponentCharacterInventories = 201
	ComponentCharacterEquipment  = 205
	ComponentItemInstances       = 300
	ComponentItemPerks           = 302
	ComponentItemStats           = 304
	ComponentItemSockets         = 305
	ComponentCollectibles        = 800
	ComponentRecords             = 900
	ComponentMetrics             = 1100
)
