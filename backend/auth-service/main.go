package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// Configuration holds all environment variables
type Config struct {
	Port               string
	GoEnv              string
	BungieClientID     string
	BungieClientSecret string
	BungieAPIKey       string
	AuthRedirectURI    string
	JWTSecret          string
	CORSAllowedOrigins string
	InternalAPIKey     string // For service-to-service auth
}

// Global config
var config Config

// CSRF state store with expiration (thread-safe)
var stateStore = struct {
	sync.RWMutex
	states map[string]time.Time
}{states: make(map[string]time.Time)}

const stateExpiration = 10 * time.Minute

func init() {
	// Load configuration on startup
	config = loadConfig()
}

// generateState creates a cryptographically secure random state for CSRF protection
func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	state := base64.URLEncoding.EncodeToString(b)

	// Store state with expiration
	stateStore.Lock()
	stateStore.states[state] = time.Now().Add(stateExpiration)
	stateStore.Unlock()

	return state, nil
}

// validateAndConsumeState checks if a state is valid and removes it (one-time use)
func validateAndConsumeState(state string) bool {
	stateStore.Lock()
	defer stateStore.Unlock()

	expiry, exists := stateStore.states[state]
	if !exists {
		return false
	}

	// Remove state (one-time use)
	delete(stateStore.states, state)

	// Check if expired
	return time.Now().Before(expiry)
}

// cleanupExpiredStates removes expired states periodically
func cleanupExpiredStates() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		stateStore.Lock()
		now := time.Now()
		for state, expiry := range stateStore.states {
			if now.After(expiry) {
				delete(stateStore.states, state)
			}
		}
		stateStore.Unlock()
	}
}

// loadConfig loads and validates configuration from environment
func loadConfig() Config {
	return Config{
		Port:               getEnv("PORT", "8081"),
		GoEnv:              getEnv("GO_ENV", "development"),
		BungieClientID:     os.Getenv("BUNGIE_CLIENT_ID"),
		BungieClientSecret: os.Getenv("BUNGIE_CLIENT_SECRET"),
		BungieAPIKey:       os.Getenv("BUNGIE_API_KEY"),
		AuthRedirectURI:    getEnv("AUTH_REDIRECT_URI", "http://localhost:3000/auth/callback"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000"),
		InternalAPIKey:     getEnv("INTERNAL_API_KEY", "dev-internal-key"),
	}
}

// getEnv gets environment variable with fallback
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// validateEnvironment ensures required environment variables are set
func validateEnvironment() {
	required := map[string]string{
		"BUNGIE_CLIENT_ID": config.BungieClientID,
		// BUNGIE_CLIENT_SECRET is optional for public OAuth apps
		"BUNGIE_API_KEY": config.BungieAPIKey,
		"JWT_SECRET":     config.JWTSecret,
	}

	missing := []string{}
	for key, value := range required {
		if value == "" {
			missing = append(missing, key)
		}
	}

	if len(missing) > 0 {
		log.Printf("WARNING: Missing required environment variables: %v", missing)
		log.Println("Please create a .env file based on .env.example")
		if config.GoEnv == "production" {
			log.Fatal("Cannot start in production without required environment variables")
		}
	}

	// Validate JWT secret strength in production
	if config.GoEnv == "production" && len(config.JWTSecret) < 32 {
		log.Fatal("JWT_SECRET must be at least 32 characters in production")
	}
}

// internalAPIKeyMiddleware validates the internal API key for service-to-service auth
func internalAPIKeyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-Internal-API-Key")

		if apiKey == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": "Internal API key required"})
			return
		}

		if apiKey != config.InternalAPIKey {
			c.AbortWithStatusJSON(403, gin.H{"error": "Invalid internal API key"})
			return
		}

		c.Next()
	}
}

// corsMiddleware handles CORS
func corsMiddleware() gin.HandlerFunc {
	allowedOrigins := strings.Split(config.CORSAllowedOrigins, ",")

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if strings.TrimSpace(allowedOrigin) == origin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
		} else if config.GoEnv != "production" {
			// Allow all origins in development
			c.Header("Access-Control-Allow-Origin", "*")
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Reload config after loading .env
	config = loadConfig()

	// Validate required environment variables
	validateEnvironment()

	// Start background cleanup of expired CSRF states
	go cleanupExpiredStates()

	// Setup Gin router
	if config.GoEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Add CORS middleware
	router.Use(corsMiddleware())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "ok",
			"service":   "auth-service",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Ready check for K8s
	router.GET("/ready", func(c *gin.Context) {
		c.JSON(200, gin.H{"ready": true})
	})

	// API routes
	api := router.Group("/api")
	{
		// Bungie OAuth routes with CSRF protection
		api.GET("/auth/bungie", func(c *gin.Context) {
			// Generate CSRF state
			state, err := generateState()
			if err != nil {
				log.Printf("Error generating CSRF state: %v", err)
				c.JSON(500, gin.H{"error": "Failed to initialize authentication"})
				return
			}

			authURL := fmt.Sprintf(
				"https://www.bungie.net/en/OAuth/Authorize?client_id=%s&response_type=code&redirect_uri=%s&state=%s",
				config.BungieClientID,
				url.QueryEscape(config.AuthRedirectURI),
				url.QueryEscape(state),
			)

			if config.GoEnv != "production" {
				log.Printf("Generated OAuth URL with state")
			}

			c.JSON(200, gin.H{
				"authUrl": authURL,
				"state":   state,
			})
		})

		api.POST("/auth/bungie/callback", func(c *gin.Context) {
			code := c.PostForm("code")
			state := c.PostForm("state")

			// Validate code
			if code == "" {
				log.Printf("OAuth callback error: No authorization code provided")
				c.JSON(400, gin.H{"error": "Authorization code is required"})
				return
			}

			// Validate code length to prevent DoS
			if len(code) > 500 {
				log.Printf("OAuth callback error: Authorization code too long")
				c.JSON(400, gin.H{"error": "Invalid authorization code"})
				return
			}

			// Validate CSRF state
			if state == "" {
				log.Printf("OAuth callback error: No state parameter provided")
				c.JSON(400, gin.H{"error": "State parameter is required"})
				return
			}

			if !validateAndConsumeState(state) {
				log.Printf("OAuth callback error: Invalid or expired state")
				c.JSON(400, gin.H{"error": "Invalid or expired state. Please try logging in again."})
				return
			}

			if config.GoEnv != "production" {
				log.Printf("Processing OAuth callback (state validated)")
			}

			// Exchange code for access token
			tokenResp, err := exchangeCodeForToken(code)
			if err != nil {
				log.Printf("Error exchanging code for token: %v", err)
				// Don't expose internal error details
				c.JSON(500, gin.H{"error": "Failed to complete authentication"})
				return
			}

			// Get user profile from Bungie API
			userProfile, err := getBungieUserProfile(tokenResp.AccessToken)
			if err != nil {
				log.Printf("Error getting user profile: %v", err)
				c.JSON(500, gin.H{"error": "Failed to retrieve user profile"})
				return
			}

			// Store Bungie tokens for later use by other services
			now := time.Now()
			bungieTokenStore.Store(userProfile.MembershipID, &BungieTokens{
				AccessToken:           tokenResp.AccessToken,
				RefreshToken:          tokenResp.RefreshToken,
				AccessTokenExpiresAt:  now.Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
				RefreshTokenExpiresAt: now.Add(90 * 24 * time.Hour), // Bungie refresh tokens last ~90 days
				MembershipID:          userProfile.MembershipID,
			})

			if config.GoEnv != "production" {
				log.Printf("Successfully retrieved user profile for: %s", userProfile.DisplayName)
			}

			// Generate JWT access token
			accessToken, err := GenerateAccessToken(userProfile)
			if err != nil {
				log.Printf("Error generating access token: %v", err)
				c.JSON(500, gin.H{"error": "Failed to create session"})
				return
			}

			// Generate JWT refresh token
			refreshToken, err := GenerateRefreshToken(userProfile)
			if err != nil {
				log.Printf("Error generating refresh token: %v", err)
				c.JSON(500, gin.H{"error": "Failed to create session"})
				return
			}

			c.JSON(200, gin.H{
				"token":        accessToken,
				"refreshToken": refreshToken,
				"user": gin.H{
					"id":             userProfile.MembershipID,
					"displayName":    userProfile.DisplayName,
					"membershipId":   userProfile.MembershipID,
					"membershipType": userProfile.MembershipType,
					"platform":       getPlatformName(userProfile.MembershipType),
				},
			})
		})

		// Token refresh endpoint
		api.POST("/auth/refresh", func(c *gin.Context) {
			var requestBody struct {
				RefreshToken string `json:"refreshToken" binding:"required"`
			}

			if err := c.ShouldBindJSON(&requestBody); err != nil {
				c.JSON(400, gin.H{"error": "Refresh token is required"})
				return
			}

			// Validate the refresh token
			claims, err := ValidateToken(requestBody.RefreshToken)
			if err != nil {
				log.Printf("Invalid refresh token: %v", err)
				c.JSON(401, gin.H{"error": "Invalid or expired refresh token"})
				return
			}

			// Verify it's actually a refresh token
			if claims.TokenType != "refresh" {
				c.JSON(401, gin.H{"error": "Invalid token type, refresh token required"})
				return
			}

			// Create a user profile from the claims to generate new tokens
			userProfile := &BungieUserProfile{
				MembershipID:   claims.MembershipID,
				DisplayName:    claims.DisplayName,
				MembershipType: claims.MembershipType,
			}

			// Generate new access token
			newAccessToken, err := GenerateAccessToken(userProfile)
			if err != nil {
				log.Printf("Error generating new access token: %v", err)
				c.JSON(500, gin.H{"error": "Failed to refresh session"})
				return
			}

			// Generate new refresh token (rotation strategy)
			newRefreshToken, err := GenerateRefreshToken(userProfile)
			if err != nil {
				log.Printf("Error generating new refresh token: %v", err)
				c.JSON(500, gin.H{"error": "Failed to refresh session"})
				return
			}

			c.JSON(200, gin.H{
				"token":        newAccessToken,
				"refreshToken": newRefreshToken,
				"user": gin.H{
					"id":             claims.MembershipID,
					"displayName":    claims.DisplayName,
					"membershipId":   claims.MembershipID,
					"membershipType": claims.MembershipType,
					"platform":       claims.Platform,
				},
			})
		})

		// Validate token endpoint (protected route example)
		api.GET("/auth/validate", AuthMiddleware(), func(c *gin.Context) {
			membershipID, _ := c.Get("membership_id")
			displayName, _ := c.Get("display_name")
			membershipType, _ := c.Get("membership_type")
			platform, _ := c.Get("platform")

			c.JSON(200, gin.H{
				"valid": true,
				"user": gin.H{
					"id":             membershipID,
					"displayName":    displayName,
					"membershipId":   membershipID,
					"membershipType": membershipType,
					"platform":       platform,
				},
			})
		})

		// Auth profile endpoint (for GraphQL service)
		api.GET("/auth/profile", AuthMiddleware(), func(c *gin.Context) {
			membershipID, _ := c.Get("membership_id")
			displayName, _ := c.Get("display_name")
			membershipType, _ := c.Get("membership_type")
			platform, _ := c.Get("platform")

			c.JSON(200, gin.H{
				"user": gin.H{
					"id":             membershipID,
					"displayName":    displayName,
					"membershipId":   membershipID,
					"membershipType": membershipType,
					"platform":       platform,
				},
			})
		})

		// Wishlist routes (protected)
		api.POST("/wishlist", AuthMiddleware(), func(c *gin.Context) {
			membershipID, _ := c.Get("membership_id")

			// Validate input
			itemID := c.PostForm("itemId")
			if itemID == "" || len(itemID) > 20 {
				c.JSON(400, gin.H{"error": "Invalid item ID"})
				return
			}

			c.JSON(200, gin.H{
				"message": "Item added to wishlist successfully",
				"itemId":  itemID,
				"userId":  membershipID,
			})
		})

		api.GET("/wishlist", AuthMiddleware(), func(c *gin.Context) {
			membershipID, _ := c.Get("membership_id")

			c.JSON(200, []gin.H{
				{
					"id":       "1",
					"itemId":   "1498876634",
					"itemName": "Fatebringer",
					"itemType": "Hand Cannon",
					"addedAt":  "2025-08-02T22:30:00Z",
					"userId":   membershipID,
				},
				{
					"id":       "2",
					"itemId":   "3653573172",
					"itemName": "Vex Mythoclast",
					"itemType": "Fusion Rifle",
					"addedAt":  "2025-08-01T15:20:00Z",
					"userId":   membershipID,
				},
			})
		})

		api.DELETE("/wishlist/:id", AuthMiddleware(), func(c *gin.Context) {
			itemId := c.Param("id")
			membershipID, _ := c.Get("membership_id")

			// Validate item ID
			if len(itemId) > 50 {
				c.JSON(400, gin.H{"error": "Invalid item ID"})
				return
			}

			c.JSON(200, gin.H{
				"message": "Item removed from wishlist",
				"itemId":  itemId,
				"userId":  membershipID,
			})
		})
	}

	// Internal API routes (for service-to-service communication)
	internal := router.Group("/internal")
	internal.Use(internalAPIKeyMiddleware())
	{
		// Get Bungie access token for a user (used by bungie-service)
		internal.GET("/bungie-token/:membershipId", func(c *gin.Context) {
			membershipID := c.Param("membershipId")

			if membershipID == "" {
				c.JSON(400, gin.H{"error": "Membership ID is required"})
				return
			}

			// Get valid token (auto-refreshes if needed)
			accessToken, err := bungieTokenStore.GetValidToken(membershipID)
			if err != nil {
				log.Printf("Failed to get Bungie token for %s: %v", membershipID, err)
				c.JSON(404, gin.H{
					"error": "No valid Bungie token found for user",
					"code":  "TOKEN_NOT_FOUND",
				})
				return
			}

			c.JSON(200, gin.H{
				"accessToken":  accessToken,
				"membershipId": membershipID,
			})
		})

		// Health check for internal services
		internal.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status":  "ok",
				"service": "auth-service-internal",
			})
		})
	}

	// Create HTTP server with timeouts
	port := config.Port
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Auth service starting on port %s", port)
		log.Printf("Environment: %s", config.GoEnv)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}

// Bungie OAuth data structures
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	MembershipID string `json:"membership_id"`
}

type BungieUserProfile struct {
	MembershipID   string `json:"membershipId"`
	DisplayName    string `json:"displayName"`
	MembershipType int    `json:"membershipType"`
}

type BungieAPIResponse struct {
	Response struct {
		DestinyMemberships []struct {
			MembershipType int    `json:"membershipType"`
			MembershipID   string `json:"membershipId"`
			DisplayName    string `json:"displayName"`
		} `json:"destinyMemberships"`
	} `json:"Response"`
}

// Exchange authorization code for access token
func exchangeCodeForToken(code string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("client_id", config.BungieClientID)
	// Only include client_secret if configured (confidential apps)
	if config.BungieClientSecret != "" {
		data.Set("client_secret", config.BungieClientSecret)
	}

	req, err := http.NewRequest("POST", "https://www.bungie.net/platform/app/oauth/token/", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("token exchange failed with status %d", resp.StatusCode)
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %v", err)
	}

	return &tokenResp, nil
}

// Get user profile from Bungie API
func getBungieUserProfile(accessToken string) (*BungieUserProfile, error) {
	req, err := http.NewRequest("GET", "https://www.bungie.net/Platform/User/GetMembershipsForCurrentUser/", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-API-Key", config.BungieAPIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("profile fetch failed with status %d", resp.StatusCode)
	}

	var apiResp BungieAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse profile response: %v", err)
	}

	// Get the first Destiny membership
	if len(apiResp.Response.DestinyMemberships) > 0 {
		membership := apiResp.Response.DestinyMemberships[0]
		return &BungieUserProfile{
			MembershipID:   membership.MembershipID,
			DisplayName:    membership.DisplayName,
			MembershipType: membership.MembershipType,
		}, nil
	}

	return nil, fmt.Errorf("no Destiny memberships found for user")
}

// Get platform name from membership type
func getPlatformName(membershipType int) string {
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
