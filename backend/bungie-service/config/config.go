package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration
type Config struct {
	// Server settings
	Port  string
	GoEnv string

	// Bungie API settings
	BungieAPIKey     string
	BungieAPIBaseURL string

	// Auth service settings (for fetching Bungie tokens)
	AuthServiceURL string
	InternalAPIKey string
	JWTSecret      string

	// Manifest settings
	ManifestDBPath        string
	ManifestCheckInterval time.Duration

	// Rate limiting
	RateLimitRPS   int
	RateLimitBurst int

	// Cache settings
	CacheEnabled        bool
	CacheTTLCollections time.Duration
	CacheTTLManifest    time.Duration

	// CORS
	CORSAllowedOrigins []string
}

// Global config instance
var cfg *Config

// Load initializes and returns the configuration
func Load() *Config {
	if cfg != nil {
		return cfg
	}

	cfg = &Config{
		// Server
		Port:  getEnv("PORT", "8082"),
		GoEnv: getEnv("GO_ENV", "development"),

		// Bungie API
		BungieAPIKey:     os.Getenv("BUNGIE_API_KEY"),
		BungieAPIBaseURL: getEnv("BUNGIE_API_BASE_URL", "https://www.bungie.net/Platform"),

		// Auth service (for fetching Bungie tokens)
		AuthServiceURL: getEnv("AUTH_SERVICE_URL", "http://localhost:8081"),
		InternalAPIKey: getEnv("INTERNAL_API_KEY", "dev-internal-key"),
		JWTSecret:      os.Getenv("JWT_SECRET"),

		// Manifest
		ManifestDBPath:        getEnv("MANIFEST_DB_PATH", "./data/manifest.sqlite"),
		ManifestCheckInterval: getDurationEnv("MANIFEST_CHECK_INTERVAL", 1*time.Hour),

		// Rate limiting
		RateLimitRPS:   getIntEnv("BUNGIE_API_RPS", 10),
		RateLimitBurst: getIntEnv("BUNGIE_API_BURST", 20),

		// Cache
		CacheEnabled:        getBoolEnv("CACHE_ENABLED", true),
		CacheTTLCollections: getDurationEnv("CACHE_TTL_COLLECTIONS", 5*time.Minute),
		CacheTTLManifest:    getDurationEnv("CACHE_TTL_MANIFEST", 24*time.Hour),

		// CORS
		CORSAllowedOrigins: strings.Split(
			getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:4000"),
			",",
		),
	}

	return cfg
}

// Get returns the current configuration (must call Load first)
func Get() *Config {
	if cfg == nil {
		return Load()
	}
	return cfg
}

// Validate checks that required configuration is present
func (c *Config) Validate() error {
	if c.BungieAPIKey == "" {
		log.Println("WARNING: BUNGIE_API_KEY is not set")
		if c.GoEnv == "production" {
			log.Fatal("Cannot start in production without BUNGIE_API_KEY")
		}
	}
	return nil
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.GoEnv == "production"
}

// Helper functions for environment variable parsing

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getIntEnv(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}

func getBoolEnv(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return fallback
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		// Try parsing as seconds first
		if seconds, err := strconv.Atoi(value); err == nil {
			return time.Duration(seconds) * time.Second
		}
		// Try parsing as duration string (e.g., "5m", "1h")
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return fallback
}
