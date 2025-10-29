package main

import (
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// Configuration holds all environment variables
type Config struct {
	Port               string
	GoEnv              string
	BungieAPIKey       string
	BungieAPIBaseURL   string
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
		Port:               getEnv("PORT", "8082"),
		GoEnv:              getEnv("GO_ENV", "development"),
		BungieAPIKey:       os.Getenv("BUNGIE_API_KEY"),
		BungieAPIBaseURL:   getEnv("BUNGIE_API_BASE_URL", "https://www.bungie.net/Platform"),
		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:4000"),
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
	if config.BungieAPIKey == "" {
		log.Println("WARNING: BUNGIE_API_KEY is not set")
		log.Println("Please create a .env file based on .env.example")
		if config.GoEnv == "production" {
			log.Fatal("Cannot start in production without BUNGIE_API_KEY")
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
			"service":   "bungie-service",
			"timestamp": gin.H{"time": "now"},
		})
	})

	// API routes placeholder
	api := router.Group("/api")
	{
		api.GET("/collections/:membershipType/:membershipId", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"weapons": gin.H{
					"total":     500,
					"collected": 450,
					"missing":   []gin.H{},
				},
				"armor": gin.H{
					"total":     300,
					"collected": 280,
					"missing":   []gin.H{},
				},
				"exotics": gin.H{
					"total":     100,
					"collected": 85,
					"missing":   []gin.H{},
				},
			})
		})

		api.GET("/items/search", func(c *gin.Context) {
			c.JSON(200, []gin.H{})
		})

		api.GET("/weekly/recommendations", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"vendors":    []gin.H{},
				"activities": []gin.H{},
				"pursuits":   []gin.H{},
			})
		})
	}

	// Start server
	port := config.Port

	log.Printf("Bungie service starting on port %s", port)
	log.Printf("Environment: %s", config.GoEnv)
	log.Printf("Bungie API Base URL: %s", config.BungieAPIBaseURL)

	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
