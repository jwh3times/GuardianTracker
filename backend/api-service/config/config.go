package config

import (
	"errors"
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

	// BungieAuthorizeURL / BungieTokenURL / BungieCDNBaseURL let dev & E2E point
	// the OAuth pages, token endpoint, and manifest-zip host at a fake Bungie.
	// BungieAuthorizeURL is browser-facing; the other two are server-side.
	BungieAuthorizeURL string
	BungieTokenURL     string
	BungieCDNBaseURL   string

	JWTSecret            string
	JWTAccessTTL         time.Duration
	JWTRefreshExpiryDays int

	ManifestDBPath        string
	ManifestCheckInterval time.Duration

	RateLimitRPS   int
	RateLimitBurst int

	// AuthRateLimitRPS/Burst throttle the unauthenticated auth endpoints per
	// client IP. MaxBodyBytes caps every request body (http.MaxBytesReader).
	AuthRateLimitRPS   int
	AuthRateLimitBurst int
	MaxBodyBytes       int64

	CacheEnabled        bool
	CacheTTLCollections time.Duration
	CacheTTLRecords     time.Duration

	CORSAllowedOrigins []string

	DatabaseURL                   string
	DBMaxConns                    int32
	TokenEncryptionKey            string
	TokenEncryptionKeyVersion     int16
	TokenEncryptionKeyPrev        string
	TokenEncryptionKeyPrevVersion int16

	loadErr error

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
	keyVersion, keyVersionErr := getPositiveSmallIntEnv("TOKEN_ENCRYPTION_KEY_VERSION", 1)
	previousKeyVersion, previousKeyVersionErr := getPositiveSmallIntEnv("TOKEN_ENCRYPTION_KEY_PREVIOUS_VERSION", 0)

	cfg := &Config{
		Port:  getEnv("PORT", "8081"),
		GoEnv: strings.TrimSpace(os.Getenv("GO_ENV")),

		BungieAPIKey:       os.Getenv("BUNGIE_API_KEY"),
		BungieAPIBaseURL:   getEnv("BUNGIE_API_BASE_URL", "https://www.bungie.net/Platform"),
		BungieClientID:     os.Getenv("BUNGIE_CLIENT_ID"),
		BungieClientSecret: os.Getenv("BUNGIE_CLIENT_SECRET"),
		AuthRedirectURI:    getEnv("AUTH_REDIRECT_URI", "http://localhost:3000/auth/callback"),

		BungieAuthorizeURL: getEnv("BUNGIE_AUTHORIZE_URL", "https://www.bungie.net/en/OAuth/Authorize"),
		BungieTokenURL:     getEnv("BUNGIE_TOKEN_URL", "https://www.bungie.net/platform/app/oauth/token/"),
		BungieCDNBaseURL:   getEnv("BUNGIE_CDN_BASE_URL", "https://www.bungie.net"),

		JWTSecret:            os.Getenv("JWT_SECRET"),
		JWTAccessTTL:         getJWTAccessTTL(),
		JWTRefreshExpiryDays: getIntEnv("JWT_REFRESH_EXPIRY_DAYS", 30),

		ManifestDBPath:        getEnv("MANIFEST_DB_PATH", "./data/manifest.sqlite"),
		ManifestCheckInterval: getDurationEnv("MANIFEST_CHECK_INTERVAL", 1*time.Hour),

		RateLimitRPS:   getIntEnv("BUNGIE_API_RPS", 10),
		RateLimitBurst: getIntEnv("BUNGIE_API_BURST", 20),

		AuthRateLimitRPS:   getIntEnv("AUTH_RATE_LIMIT_RPS", 5),
		AuthRateLimitBurst: getIntEnv("AUTH_RATE_LIMIT_BURST", 10),
		MaxBodyBytes:       int64(getIntEnv("MAX_BODY_BYTES", 65536)),

		CacheEnabled:        getBoolEnv("CACHE_ENABLED", true),
		CacheTTLCollections: getDurationEnv("CACHE_TTL_COLLECTIONS", 5*time.Minute),
		CacheTTLRecords:     getDurationEnv("CACHE_TTL_RECORDS", 10*time.Minute),

		CORSAllowedOrigins: strings.Split(
			getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000"),
			",",
		),

		DatabaseURL:                   os.Getenv("DATABASE_URL"),
		DBMaxConns:                    getInt32Env("DB_MAX_CONNS", 4),
		TokenEncryptionKey:            os.Getenv("TOKEN_ENCRYPTION_KEY"),
		TokenEncryptionKeyVersion:     keyVersion,
		TokenEncryptionKeyPrev:        os.Getenv("TOKEN_ENCRYPTION_KEY_PREVIOUS"),
		TokenEncryptionKeyPrevVersion: previousKeyVersion,
		loadErr:                       errors.Join(keyVersionErr, previousKeyVersionErr),

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
	switch c.GoEnv {
	case "development", "production":
		// Explicitly supported environments.
	case "":
		return fmt.Errorf("GO_ENV is required (must be development or production)")
	default:
		return fmt.Errorf("GO_ENV must be development or production (got %q)", c.GoEnv)
	}
	if c.loadErr != nil {
		return c.loadErr
	}
	if c.TokenEncryptionKey != "" && c.TokenEncryptionKeyVersion <= 0 {
		return fmt.Errorf("TOKEN_ENCRYPTION_KEY_VERSION must be a positive SMALLINT")
	}
	if c.TokenEncryptionKeyPrev != "" {
		if c.TokenEncryptionKey == "" {
			return fmt.Errorf("TOKEN_ENCRYPTION_KEY_PREVIOUS requires TOKEN_ENCRYPTION_KEY")
		}
		if c.TokenEncryptionKeyPrevVersion <= 0 {
			return fmt.Errorf("TOKEN_ENCRYPTION_KEY_PREVIOUS_VERSION must be a positive SMALLINT when TOKEN_ENCRYPTION_KEY_PREVIOUS is set")
		}
		if c.TokenEncryptionKeyPrevVersion == c.TokenEncryptionKeyVersion {
			return fmt.Errorf("current and previous token encryption key versions must differ")
		}
	} else if c.TokenEncryptionKeyPrevVersion != 0 {
		return fmt.Errorf("TOKEN_ENCRYPTION_KEY_PREVIOUS_VERSION requires TOKEN_ENCRYPTION_KEY_PREVIOUS")
	}

	developmentWarnings := []string{}
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
		if c.IsProduction() {
			return fmt.Errorf("cannot start in production without required environment variables: %v", missing)
		}
		developmentWarnings = append(developmentWarnings, "missing application configuration: "+strings.Join(missing, ", "))
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
			developmentWarnings = append(developmentWarnings, "PostgreSQL-backed persistence and revocation")
		}
		if c.TokenEncryptionKey == "" {
			developmentWarnings = append(developmentWarnings, "Bungie token encryption at rest")
		}
	}
	// A "*" entry combined with the credentialed CORS middleware would reflect
	// any Origin with Allow-Credentials: true. Never allow it in production.
	if slices.Contains(c.CORSAllowedOrigins, "*") {
		if c.IsProduction() {
			return fmt.Errorf("CORS_ALLOWED_ORIGINS must not contain \"*\" (credentialed CORS)")
		}
		developmentWarnings = append(developmentWarnings, "browser CORS access (wildcard origins are not reflected)")
	}
	if len(developmentWarnings) > 0 {
		log.Printf("SECURITY WARNING: development mode is degraded; disabled protections/features: %s", strings.Join(developmentWarnings, "; "))
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

// getPositiveSmallIntEnv parses a PostgreSQL SMALLINT key version. An unset
// value returns fallback; any explicitly configured value must be in 1..32767.
// A zero fallback is used for the optional previous-key version.
func getPositiveSmallIntEnv(key string, fallback int16) (int16, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.ParseInt(v, 10, 16)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("%s must be a positive SMALLINT", key)
	}
	return int16(n), nil
}

func getBoolEnv(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

const defaultJWTAccessTTL = 30 * time.Minute

// getJWTAccessTTL prefers the duration-based setting so access tokens can be
// configured in minutes. JWT_EXPIRY_HOURS remains a compatibility fallback
// for existing deployments that have not migrated their environment yet.
func getJWTAccessTTL() time.Duration {
	if ttl := getDurationEnv("JWT_ACCESS_TTL", 0); ttl > 0 {
		return ttl
	}
	if hours := getIntEnv("JWT_EXPIRY_HOURS", 0); hours > 0 {
		return time.Duration(hours) * time.Hour
	}
	return defaultJWTAccessTTL
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
