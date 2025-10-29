package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

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
}

// Global config
var config Config

func init() {
	// Load configuration on startup
	config = loadConfig()
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
		"BUNGIE_CLIENT_ID":     config.BungieClientID,
		"BUNGIE_CLIENT_SECRET": config.BungieClientSecret,
		"BUNGIE_API_KEY":       config.BungieAPIKey,
		"JWT_SECRET":           config.JWTSecret,
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

	// Validate required environment variables
	validateEnvironment()

	// Setup Gin router
	if os.Getenv("GO_ENV") == "production" {
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
			"timestamp": gin.H{"time": "now"},
		})
	})

	// API routes
	api := router.Group("/api")
	{
		// Bungie OAuth routes
		api.GET("/auth/bungie", func(c *gin.Context) {
			authURL := fmt.Sprintf(
				"https://www.bungie.net/en/OAuth/Authorize?client_id=%s&response_type=code&redirect_uri=%s",
				config.BungieClientID,
				url.QueryEscape(config.AuthRedirectURI),
			)

			log.Printf("Generated OAuth URL for client ID: %s", config.BungieClientID)
			c.JSON(200, gin.H{"authUrl": authURL})
		})

		api.POST("/auth/bungie/callback", func(c *gin.Context) {
			code := c.PostForm("code")
			if code == "" {
				log.Printf("OAuth callback error: No authorization code provided")
				c.JSON(400, gin.H{"error": "Authorization code is required"})
				return
			}

			log.Printf("Processing OAuth callback with code: %s", code[:10]+"...")

			// Exchange code for access token
			tokenResp, err := exchangeCodeForToken(code)
			if err != nil {
				log.Printf("Error exchanging code for token: %v", err)
				c.JSON(500, gin.H{"error": "Failed to exchange authorization code", "details": err.Error()})
				return
			}

			log.Printf("Successfully obtained access token")

			// Get user profile from Bungie API
			userProfile, err := getBungieUserProfile(tokenResp.AccessToken)
			if err != nil {
				log.Printf("Error getting user profile: %v", err)
				c.JSON(500, gin.H{"error": "Failed to get user profile", "details": err.Error()})
				return
			}

			log.Printf("Successfully retrieved user profile for: %s", userProfile.DisplayName)

			// Generate JWT access token
			accessToken, err := GenerateAccessToken(userProfile)
			if err != nil {
				log.Printf("Error generating access token: %v", err)
				c.JSON(500, gin.H{"error": "Failed to generate access token", "details": err.Error()})
				return
			}

			// Generate JWT refresh token
			refreshToken, err := GenerateRefreshToken(userProfile)
			if err != nil {
				log.Printf("Error generating refresh token: %v", err)
				c.JSON(500, gin.H{"error": "Failed to generate refresh token", "details": err.Error()})
				return
			}

			log.Printf("Successfully generated JWT tokens for user: %s", userProfile.MembershipID)

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
				c.JSON(500, gin.H{"error": "Failed to generate new access token"})
				return
			}

			// Optionally generate a new refresh token (rotation strategy)
			newRefreshToken, err := GenerateRefreshToken(userProfile)
			if err != nil {
				log.Printf("Error generating new refresh token: %v", err)
				c.JSON(500, gin.H{"error": "Failed to generate new refresh token"})
				return
			}

			log.Printf("Successfully refreshed tokens for user: %s", claims.MembershipID)

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
			// If middleware passes, token is valid
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

		// Wishlist routes (now protected)
		api.POST("/wishlist", AuthMiddleware(), func(c *gin.Context) {
			membershipID, _ := c.Get("membership_id")

			c.JSON(200, gin.H{
				"message": "Item added to wishlist successfully",
				"itemId":  c.PostForm("itemId"),
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

			c.JSON(200, gin.H{
				"message": "Item removed from wishlist",
				"itemId":  itemId,
				"userId":  membershipID,
			})
		})
	}

	// Start server
	port := config.Port

	log.Printf("Auth service starting on port %s", port)
	log.Printf("Environment: %s", config.GoEnv)
	log.Printf("OAuth Redirect URI: %s", config.AuthRedirectURI)

	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
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
	data.Set("client_secret", config.BungieClientSecret)

	log.Printf("Exchanging code for token with client ID: %s", config.BungieClientID)

	req, err := http.NewRequest("POST", "https://www.bungie.net/platform/app/oauth/token/", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	log.Printf("Token exchange response status: %d", resp.StatusCode)
	if resp.StatusCode != 200 {
		log.Printf("Token exchange response body: %s", string(body))
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %v", err)
	}

	return &tokenResp, nil
}

// Get user profile from Bungie API
func getBungieUserProfile(accessToken string) (*BungieUserProfile, error) {
	log.Printf("Fetching user profile from Bungie API")

	req, err := http.NewRequest("GET", "https://www.bungie.net/Platform/User/GetMembershipsForCurrentUser/", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-API-Key", config.BungieAPIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	log.Printf("User profile response status: %d", resp.StatusCode)
	if resp.StatusCode != 200 {
		log.Printf("User profile response body: %s", string(body))
		return nil, fmt.Errorf("profile fetch failed with status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp BungieAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse profile response: %v", err)
	}

	// Get the first Destiny membership
	if len(apiResp.Response.DestinyMemberships) > 0 {
		membership := apiResp.Response.DestinyMemberships[0]
		log.Printf("Found Destiny membership: %s (%s)", membership.DisplayName, getPlatformName(membership.MembershipType))
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
