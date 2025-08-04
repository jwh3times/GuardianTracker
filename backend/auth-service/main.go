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

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Setup Gin router
	if os.Getenv("GO_ENV") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Add CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

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
			clientID := os.Getenv("BUNGIE_CLIENT_ID")
			redirectURI := os.Getenv("REACT_APP_AUTH_REDIRECT_URI")

			if clientID == "" {
				clientID = "30139"
			}
			if redirectURI == "" {
				redirectURI = "http://localhost:3000/auth/callback"
			}

			authURL := fmt.Sprintf(
				"https://www.bungie.net/en/OAuth/Authorize?client_id=%s&response_type=code&redirect_uri=%s",
				clientID,
				url.QueryEscape(redirectURI),
			)

			log.Printf("Generated OAuth URL: %s", authURL)
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

			// Generate our own JWT token (simplified for now)
			token := fmt.Sprintf("bungie-jwt-%s", userProfile.MembershipID)

			c.JSON(200, gin.H{
				"token": token,
				"user": gin.H{
					"id":             userProfile.MembershipID,
					"displayName":    userProfile.DisplayName,
					"membershipId":   userProfile.MembershipID,
					"membershipType": userProfile.MembershipType,
					"platform":       getPlatformName(userProfile.MembershipType),
				},
			})
		})

		// Wishlist routes
		api.POST("/wishlist", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "Item added to wishlist successfully",
				"itemId":  c.PostForm("itemId"),
			})
		})

		api.GET("/wishlist", func(c *gin.Context) {
			c.JSON(200, []gin.H{
				{
					"id":       "1",
					"itemId":   "1498876634",
					"itemName": "Fatebringer",
					"itemType": "Hand Cannon",
					"addedAt":  "2025-08-02T22:30:00Z",
				},
				{
					"id":       "2",
					"itemId":   "3653573172",
					"itemName": "Vex Mythoclast",
					"itemType": "Fusion Rifle",
					"addedAt":  "2025-08-01T15:20:00Z",
				},
			})
		})

		api.DELETE("/wishlist/:id", func(c *gin.Context) {
			itemId := c.Param("id")
			c.JSON(200, gin.H{
				"message": "Item removed from wishlist",
				"itemId":  itemId,
			})
		})
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("Auth service starting on port %s", port)
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
	clientID := os.Getenv("BUNGIE_CLIENT_ID")
	if clientID == "" {
		clientID = "30139"
	}

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("client_id", clientID)

	log.Printf("Exchanging code for token with client ID: %s", clientID)

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
	apiKey := os.Getenv("BUNGIE_API_KEY")
	if apiKey == "" {
		apiKey = "38bb3aae8e6347198e8e9263e86c3584"
	}

	log.Printf("Fetching user profile from Bungie API")

	req, err := http.NewRequest("GET", "https://www.bungie.net/Platform/User/GetMembershipsForCurrentUser/", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-API-Key", apiKey)

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
