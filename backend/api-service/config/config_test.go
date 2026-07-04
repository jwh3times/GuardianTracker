package config

import (
	"strings"
	"testing"
	"time"
)

func validProdConfig() *Config {
	return &Config{
		GoEnv:              "production",
		BungieAPIKey:       "key",
		BungieClientID:     "client",
		JWTSecret:          strings.Repeat("s", 32),
		DatabaseURL:        "postgres://localhost/db",
		TokenEncryptionKey: "base64key",
	}
}

func TestValidate_ProductionBranches(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*Config)
		wantErr string // substring; "" = no error
	}{
		{"complete config passes", func(c *Config) {}, ""},
		{"missing bungie key", func(c *Config) { c.BungieAPIKey = "" }, "BUNGIE_API_KEY"},
		{"missing client id", func(c *Config) { c.BungieClientID = "" }, "BUNGIE_CLIENT_ID"},
		{"missing jwt secret", func(c *Config) { c.JWTSecret = "" }, "JWT_SECRET"},
		{"short jwt secret", func(c *Config) { c.JWTSecret = "short" }, "32 characters"},
		{"missing database url", func(c *Config) { c.DatabaseURL = "" }, "DATABASE_URL"},
		{"missing encryption key", func(c *Config) { c.TokenEncryptionKey = "" }, "TOKEN_ENCRYPTION_KEY"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := validProdConfig()
			tc.mutate(cfg)
			err := cfg.Validate()
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Validate() = %v, want error containing %q", err, tc.wantErr)
			}
		})
	}
}

func TestValidate_DevelopmentNeverErrors(t *testing.T) {
	// A completely empty dev config only warns — degraded mode is supported.
	cfg := &Config{GoEnv: "development"}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() in development = %v, want nil", err)
	}
}

func TestLoad_EnvParsing(t *testing.T) {
	t.Setenv("PORT", "9999")
	t.Setenv("GO_ENV", "production")
	t.Setenv("JWT_EXPIRY_HOURS", "12")
	t.Setenv("CACHE_ENABLED", "false")
	t.Setenv("CACHE_TTL_COLLECTIONS", "120")  // bare seconds
	t.Setenv("MANIFEST_CHECK_INTERVAL", "2h") // duration string
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://a.example,http://b.example")
	t.Setenv("DB_MAX_CONNS", "8")

	cfg := Load()

	if cfg.Port != "9999" {
		t.Errorf("Port = %q", cfg.Port)
	}
	if !cfg.IsProduction() {
		t.Error("IsProduction() = false")
	}
	if cfg.JWTExpiryHours != 12 {
		t.Errorf("JWTExpiryHours = %d", cfg.JWTExpiryHours)
	}
	if cfg.CacheEnabled {
		t.Error("CacheEnabled = true, want false")
	}
	if cfg.CacheTTLCollections != 120*time.Second {
		t.Errorf("CacheTTLCollections = %v", cfg.CacheTTLCollections)
	}
	if cfg.ManifestCheckInterval != 2*time.Hour {
		t.Errorf("ManifestCheckInterval = %v", cfg.ManifestCheckInterval)
	}
	if len(cfg.CORSAllowedOrigins) != 2 || cfg.CORSAllowedOrigins[1] != "http://b.example" {
		t.Errorf("CORSAllowedOrigins = %v", cfg.CORSAllowedOrigins)
	}
	if cfg.DBMaxConns != 8 {
		t.Errorf("DBMaxConns = %d", cfg.DBMaxConns)
	}
}

func TestLoad_FallbacksOnInvalidValues(t *testing.T) {
	t.Setenv("JWT_EXPIRY_HOURS", "not-a-number")
	t.Setenv("CACHE_ENABLED", "not-a-bool")
	t.Setenv("MANIFEST_CHECK_INTERVAL", "not-a-duration")

	cfg := Load()

	if cfg.JWTExpiryHours != 24 {
		t.Errorf("JWTExpiryHours = %d, want fallback 24", cfg.JWTExpiryHours)
	}
	if !cfg.CacheEnabled {
		t.Error("CacheEnabled = false, want fallback true")
	}
	if cfg.ManifestCheckInterval != time.Hour {
		t.Errorf("ManifestCheckInterval = %v, want fallback 1h", cfg.ManifestCheckInterval)
	}
}

func TestLoad_AuditDefaults(t *testing.T) {
	t.Setenv("AUDIT_RETENTION_DAYS", "")
	t.Setenv("TRUSTED_PROXIES", "")
	c := Load()
	if c.AuditRetentionDays != 180 {
		t.Errorf("AuditRetentionDays = %d, want 180", c.AuditRetentionDays)
	}
	if len(c.TrustedProxies) != 0 {
		t.Errorf("TrustedProxies = %v, want empty", c.TrustedProxies)
	}
}

func TestLoad_AuditOverrides(t *testing.T) {
	t.Setenv("AUDIT_RETENTION_DAYS", "30")
	t.Setenv("TRUSTED_PROXIES", "10.0.0.0/8, 127.0.0.1")
	c := Load()
	if c.AuditRetentionDays != 30 {
		t.Errorf("AuditRetentionDays = %d, want 30", c.AuditRetentionDays)
	}
	if len(c.TrustedProxies) != 2 || c.TrustedProxies[0] != "10.0.0.0/8" || c.TrustedProxies[1] != "127.0.0.1" {
		t.Errorf("TrustedProxies = %v, want [10.0.0.0/8 127.0.0.1]", c.TrustedProxies)
	}
}

func TestLoad_AuditRetentionClamp(t *testing.T) {
	// AUDIT_RETENTION_DAYS=0 must be clamped to 180; otherwise the hourly pruner
	// computes a cutoff of now() and deletes the entire audit_log table each tick.
	t.Setenv("AUDIT_RETENTION_DAYS", "0")
	c := Load()
	if c.AuditRetentionDays != 180 {
		t.Errorf("AuditRetentionDays = %d with input 0, want clamp to 180", c.AuditRetentionDays)
	}

	// A valid positive override must pass through unchanged.
	t.Setenv("AUDIT_RETENTION_DAYS", "30")
	c = Load()
	if c.AuditRetentionDays != 30 {
		t.Errorf("AuditRetentionDays = %d with input 30, want 30", c.AuditRetentionDays)
	}
}

func TestLoadBungieURLDefaults(t *testing.T) {
	cfg := Load()
	if cfg.BungieAuthorizeURL != "https://www.bungie.net/en/OAuth/Authorize" {
		t.Errorf("BungieAuthorizeURL = %q", cfg.BungieAuthorizeURL)
	}
	if cfg.BungieTokenURL != "https://www.bungie.net/platform/app/oauth/token/" {
		t.Errorf("BungieTokenURL = %q", cfg.BungieTokenURL)
	}
	if cfg.BungieCDNBaseURL != "https://www.bungie.net" {
		t.Errorf("BungieCDNBaseURL = %q", cfg.BungieCDNBaseURL)
	}
	if cfg.AuthRateLimitRPS != 5 || cfg.AuthRateLimitBurst != 10 {
		t.Errorf("auth rate limit = %d/%d, want 5/10", cfg.AuthRateLimitRPS, cfg.AuthRateLimitBurst)
	}
	if cfg.MaxBodyBytes != 65536 {
		t.Errorf("MaxBodyBytes = %d, want 65536", cfg.MaxBodyBytes)
	}
}

func TestLoadBungieURLOverrides(t *testing.T) {
	t.Setenv("BUNGIE_AUTHORIZE_URL", "http://localhost:8090/en/OAuth/Authorize")
	t.Setenv("BUNGIE_TOKEN_URL", "http://fake:8090/platform/app/oauth/token/")
	t.Setenv("BUNGIE_CDN_BASE_URL", "http://fake:8090")
	cfg := Load()
	if cfg.BungieAuthorizeURL != "http://localhost:8090/en/OAuth/Authorize" ||
		cfg.BungieTokenURL != "http://fake:8090/platform/app/oauth/token/" ||
		cfg.BungieCDNBaseURL != "http://fake:8090" {
		t.Errorf("overrides not honored: %+v", cfg)
	}
}

func TestValidateRejectsWildcardCORSInProduction(t *testing.T) {
	t.Setenv("GO_ENV", "production")
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("BUNGIE_API_KEY", "k")
	t.Setenv("BUNGIE_CLIENT_ID", "c")
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("TOKEN_ENCRYPTION_KEY", "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=")
	t.Setenv("CORS_ALLOWED_ORIGINS", "*")
	if err := Load().Validate(); err == nil {
		t.Fatal("expected Validate() error for CORS_ALLOWED_ORIGINS=* in production")
	}
}
