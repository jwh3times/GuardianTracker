package config

import (
	"fmt"
	"log"
	"os"
	"slices"
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

	// AdminMembershipIDs are Bungie membership IDs pinned to the admin role on
	// every login upsert. Bootstraps admin to the owner without manual SQL and
	// survives DB resets; additional admins are granted via the console.
	AdminMembershipIDs []string

	// AuditRetentionDays bounds how long audit_log rows (and the IPs they carry)
	// are retained; an hourly pruner deletes older rows.
	AuditRetentionDays int
	// TrustedProxies are CIDRs/IPs gin trusts for X-Forwarded-For when resolving
	// the client IP recorded in the audit log. Empty in local dev.
	TrustedProxies []string
}

func Load() *Config {
	cfg := &Config{
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
		DBMaxConns:             getInt32Env("DB_MAX_CONNS", 4),
		TokenEncryptionKey:     os.Getenv("TOKEN_ENCRYPTION_KEY"),
		TokenEncryptionKeyPrev: os.Getenv("TOKEN_ENCRYPTION_KEY_PREVIOUS"),

		AdminMembershipIDs: parseCSV(os.Getenv("ADMIN_MEMBERSHIP_IDS")),

		AuditRetentionDays: getIntEnv("AUDIT_RETENTION_DAYS", 180),
		TrustedProxies:     parseCSV(os.Getenv("TRUSTED_PROXIES")),
	}
	// Clamp retention floor: 0 (or negative) would make the hourly pruner compute
	// a cutoff of now() and delete the entire audit_log table on every tick.
	if cfg.AuditRetentionDays < 1 {
		cfg.AuditRetentionDays = 180
	}
	return cfg
}

// IsBootstrapAdmin reports whether a membership ID is pinned to admin via
// ADMIN_MEMBERSHIP_IDS. Called on every login upsert.
func (c *Config) IsBootstrapAdmin(membershipID string) bool {
	return slices.Contains(c.AdminMembershipIDs, membershipID)
}

// parseCSV splits a comma-separated env value into trimmed, non-empty entries.
func parseCSV(v string) []string {
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// Validate checks the configuration. In production, missing required settings
// are returned as errors (the caller fatals); in development they only warn,
// and the service runs in degraded mode.
func (c *Config) Validate() error {
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
			return fmt.Errorf("cannot start in production without required environment variables: %v", missing)
		}
	}
	if c.IsProduction() && len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters in production")
	}
	if c.IsProduction() && c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required in production")
	}
	if c.IsProduction() && c.TokenEncryptionKey == "" {
		return fmt.Errorf("TOKEN_ENCRYPTION_KEY is required in production")
	}
	if !c.IsProduction() {
		if c.DatabaseURL == "" {
			log.Println("WARNING: DATABASE_URL is not set — running in degraded mode (memory-only token store)")
		}
		if c.TokenEncryptionKey == "" {
			log.Println("WARNING: TOKEN_ENCRYPTION_KEY is not set — Bungie tokens will not be encrypted at rest")
		}
	}
	return nil
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

// getInt32Env parses an int32 env var. ParseInt with bitSize 32 rejects values
// outside the int32 range (returning an error), so the conversion can't silently
// truncate — out-of-range or unparseable values fall back to the default.
func getInt32Env(key string, fallback int32) int32 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 32); err == nil {
			return int32(n)
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
