package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port  string
	GoEnv string

	BungieAPIKey       string
	BungieAPIBaseURL   string
	BungieClientID     string
	BungieClientSecret string
	AuthRedirectURI    string

	JWTSecret            string
	JWTExpiryHours       int
	JWTRefreshExpiryDays int

	ManifestDBPath        string
	ManifestCheckInterval time.Duration

	RateLimitRPS   int
	RateLimitBurst int

	CacheEnabled        bool
	CacheTTLCollections time.Duration
	CacheTTLRecords     time.Duration

	CORSAllowedOrigins []string

	DatabaseURL            string
	DBMaxConns             int32
	TokenEncryptionKey     string
	TokenEncryptionKeyPrev string
}

func Load() *Config {
	return &Config{
		Port:  getEnv("PORT", "8081"),
		GoEnv: getEnv("GO_ENV", "development"),

		BungieAPIKey:       os.Getenv("BUNGIE_API_KEY"),
		BungieAPIBaseURL:   getEnv("BUNGIE_API_BASE_URL", "https://www.bungie.net/Platform"),
		BungieClientID:     os.Getenv("BUNGIE_CLIENT_ID"),
		BungieClientSecret: os.Getenv("BUNGIE_CLIENT_SECRET"),
		AuthRedirectURI:    getEnv("AUTH_REDIRECT_URI", "http://localhost:3000/auth/callback"),

		JWTSecret:            os.Getenv("JWT_SECRET"),
		JWTExpiryHours:       getIntEnv("JWT_EXPIRY_HOURS", 24),
		JWTRefreshExpiryDays: getIntEnv("JWT_REFRESH_EXPIRY_DAYS", 30),

		ManifestDBPath:        getEnv("MANIFEST_DB_PATH", "./data/manifest.sqlite"),
		ManifestCheckInterval: getDurationEnv("MANIFEST_CHECK_INTERVAL", 1*time.Hour),

		RateLimitRPS:   getIntEnv("BUNGIE_API_RPS", 10),
		RateLimitBurst: getIntEnv("BUNGIE_API_BURST", 20),

		CacheEnabled:        getBoolEnv("CACHE_ENABLED", true),
		CacheTTLCollections: getDurationEnv("CACHE_TTL_COLLECTIONS", 5*time.Minute),
		CacheTTLRecords:     getDurationEnv("CACHE_TTL_RECORDS", 10*time.Minute),

		CORSAllowedOrigins: strings.Split(
			getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000"),
			",",
		),

		DatabaseURL:            os.Getenv("DATABASE_URL"),
		DBMaxConns:             int32(getIntEnv("DB_MAX_CONNS", 4)),
		TokenEncryptionKey:     os.Getenv("TOKEN_ENCRYPTION_KEY"),
		TokenEncryptionKeyPrev: os.Getenv("TOKEN_ENCRYPTION_KEY_PREVIOUS"),
	}
}

func (c *Config) Validate() {
	missing := []string{}
	if c.BungieAPIKey == "" {
		missing = append(missing, "BUNGIE_API_KEY")
	}
	if c.BungieClientID == "" {
		missing = append(missing, "BUNGIE_CLIENT_ID")
	}
	if c.JWTSecret == "" {
		missing = append(missing, "JWT_SECRET")
	}
	if len(missing) > 0 {
		log.Printf("WARNING: Missing required environment variables: %v", missing)
		if c.IsProduction() {
			log.Fatalf("Cannot start in production without required environment variables")
		}
	}
	if c.IsProduction() && len(c.JWTSecret) < 32 {
		log.Fatal("JWT_SECRET must be at least 32 characters in production")
	}
	if c.IsProduction() && c.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required in production")
	}
	if c.IsProduction() && c.TokenEncryptionKey == "" {
		log.Fatal("TOKEN_ENCRYPTION_KEY is required in production")
	}
	if !c.IsProduction() {
		if c.DatabaseURL == "" {
			log.Println("WARNING: DATABASE_URL is not set — running in degraded mode (memory-only token store)")
		}
		if c.TokenEncryptionKey == "" {
			log.Println("WARNING: TOKEN_ENCRYPTION_KEY is not set — Bungie tokens will not be encrypted at rest")
		}
	}
}

func (c *Config) IsProduction() bool {
	return c.GoEnv == "production"
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getIntEnv(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getBoolEnv(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if s, err := strconv.Atoi(v); err == nil {
			return time.Duration(s) * time.Second
		}
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
