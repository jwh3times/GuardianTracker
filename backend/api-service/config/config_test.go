package config

import (
	"strings"
	"testing"
	"time"
)

func validProdConfig() *Config {
	return &Config{
		GoEnv:                     "production",
		LogLevel:                  "info",
		LogFormat:                 "json",
		BungieAPIKey:              "key",
		BungieClientID:            "client",
		JWTSecret:                 strings.Repeat("s", 32),
		DatabaseURL:               "postgres://localhost/db",
		TokenEncryptionKey:        "base64key",
		TokenEncryptionKeyVersion: 1,
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
	cfg := &Config{GoEnv: "development", LogLevel: "info", LogFormat: "text"}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() in development = %v, want nil", err)
	}
}

func TestValidate_RequiresExplicitKnownGoEnv(t *testing.T) {
	for _, tc := range []struct {
		name string
		env  string
	}{
		{name: "unset", env: ""},
		{name: "unknown", env: "staging"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{GoEnv: tc.env}
			if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "GO_ENV") {
				t.Fatalf("Validate() = %v, want GO_ENV error", err)
			}
		})
	}
}

func TestLoad_EnvParsing(t *testing.T) {
	t.Setenv("PORT", "9999")
	t.Setenv("GO_ENV", "production")
	t.Setenv("JWT_ACCESS_TTL", "45m")
	t.Setenv("JWT_EXPIRY_HOURS", "12") // legacy value must lose to the duration setting
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
	if cfg.JWTAccessTTL != 45*time.Minute {
		t.Errorf("JWTAccessTTL = %v", cfg.JWTAccessTTL)
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

func TestLoad_TokenEncryptionKeyVersions(t *testing.T) {
	t.Setenv("TOKEN_ENCRYPTION_KEY_VERSION", "")
	t.Setenv("TOKEN_ENCRYPTION_KEY_PREVIOUS_VERSION", "")
	cfg := Load()
	if cfg.TokenEncryptionKeyVersion != 1 || cfg.TokenEncryptionKeyPrevVersion != 0 {
		t.Fatalf("default versions = %d/%d, want 1/0", cfg.TokenEncryptionKeyVersion, cfg.TokenEncryptionKeyPrevVersion)
	}

	t.Setenv("TOKEN_ENCRYPTION_KEY_VERSION", "2")
	t.Setenv("TOKEN_ENCRYPTION_KEY_PREVIOUS_VERSION", "1")
	cfg = Load()
	if cfg.TokenEncryptionKeyVersion != 2 || cfg.TokenEncryptionKeyPrevVersion != 1 {
		t.Fatalf("configured versions = %d/%d, want 2/1", cfg.TokenEncryptionKeyVersion, cfg.TokenEncryptionKeyPrevVersion)
	}
}

func TestLoad_RejectsInvalidTokenEncryptionKeyVersions(t *testing.T) {
	t.Setenv("GO_ENV", "development")
	t.Setenv("TOKEN_ENCRYPTION_KEY", "configured")
	for _, value := range []string{"0", "-1", "32768", "not-a-number"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("TOKEN_ENCRYPTION_KEY_VERSION", value)
			if err := Load().Validate(); err == nil || !strings.Contains(err.Error(), "TOKEN_ENCRYPTION_KEY_VERSION") {
				t.Fatalf("Validate() = %v, want key-version error", err)
			}
		})
	}
}

func TestValidate_TokenEncryptionPreviousVersionPairing(t *testing.T) {
	base := Config{
		GoEnv:                         "development",
		LogLevel:                      "info",
		LogFormat:                     "text",
		TokenEncryptionKey:            "current",
		TokenEncryptionKeyVersion:     2,
		TokenEncryptionKeyPrev:        "previous",
		TokenEncryptionKeyPrevVersion: 1,
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("valid rotation config rejected: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{name: "previous version missing", mutate: func(c *Config) { c.TokenEncryptionKeyPrevVersion = 0 }},
		{name: "versions equal", mutate: func(c *Config) { c.TokenEncryptionKeyPrevVersion = c.TokenEncryptionKeyVersion }},
		{name: "previous key missing", mutate: func(c *Config) { c.TokenEncryptionKeyPrev = "" }},
		{name: "current key missing", mutate: func(c *Config) { c.TokenEncryptionKey = "" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := base
			tc.mutate(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("Validate() unexpectedly succeeded")
			}
		})
	}
}

func TestLoad_LoggingDefaultsByEnvironment(t *testing.T) {
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("LOG_FORMAT", "")

	t.Setenv("GO_ENV", "development")
	dev := Load()
	if dev.LogLevel != "info" || dev.LogFormat != "text" {
		t.Fatalf("development logging = %s/%s, want info/text", dev.LogLevel, dev.LogFormat)
	}

	t.Setenv("GO_ENV", "production")
	prod := Load()
	if prod.LogLevel != "info" || prod.LogFormat != "json" {
		t.Fatalf("production logging = %s/%s, want info/json", prod.LogLevel, prod.LogFormat)
	}
}

func TestValidate_RejectsInvalidLoggingConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name   string
		level  string
		format string
	}{
		{name: "level", level: "trace", format: "text"},
		{name: "format", level: "info", format: "xml"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{GoEnv: "development", LogLevel: tc.level, LogFormat: tc.format}
			if err := cfg.Validate(); err == nil {
				t.Fatal("Validate() unexpectedly succeeded")
			}
		})
	}
}

func TestLoad_E2EFixedTime(t *testing.T) {
	t.Setenv("GO_ENV", "development")
	t.Setenv("E2E_FIXED_TIME", "2026-07-11T18:00:00-04:00")
	cfg := Load()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if cfg.E2EFixedTime == nil {
		t.Fatalf("E2EFixedTime = %v", cfg.E2EFixedTime)
	}
	if got := cfg.E2EFixedTime.Format(time.RFC3339); got != "2026-07-11T22:00:00Z" {
		t.Fatalf("E2EFixedTime = %s", got)
	}
}

func TestLoad_RejectsInvalidOrProductionE2EFixedTime(t *testing.T) {
	t.Run("invalid timestamp", func(t *testing.T) {
		t.Setenv("GO_ENV", "development")
		t.Setenv("E2E_FIXED_TIME", "Saturday afternoon")
		if err := Load().Validate(); err == nil || !strings.Contains(err.Error(), "E2E_FIXED_TIME") {
			t.Fatalf("Validate() = %v, want E2E_FIXED_TIME error", err)
		}
	})
	t.Run("production", func(t *testing.T) {
		t.Setenv("GO_ENV", "production")
		t.Setenv("E2E_FIXED_TIME", "2026-07-11T18:00:00Z")
		if err := Load().Validate(); err == nil || !strings.Contains(err.Error(), "development-only") {
			t.Fatalf("Validate() = %v, want development-only error", err)
		}
	})
}

func TestLoad_FallbacksOnInvalidValues(t *testing.T) {
	t.Setenv("JWT_ACCESS_TTL", "not-a-duration")
	t.Setenv("JWT_EXPIRY_HOURS", "not-a-number")
	t.Setenv("CACHE_ENABLED", "not-a-bool")
	t.Setenv("MANIFEST_CHECK_INTERVAL", "not-a-duration")

	cfg := Load()

	if cfg.JWTAccessTTL != defaultJWTAccessTTL {
		t.Errorf("JWTAccessTTL = %v, want fallback %v", cfg.JWTAccessTTL, defaultJWTAccessTTL)
	}
	if !cfg.CacheEnabled {
		t.Error("CacheEnabled = false, want fallback true")
	}
	if cfg.ManifestCheckInterval != time.Hour {
		t.Errorf("ManifestCheckInterval = %v, want fallback 1h", cfg.ManifestCheckInterval)
	}
}

func TestLoad_LegacyJWTExpiryHoursFallback(t *testing.T) {
	t.Setenv("JWT_ACCESS_TTL", "")
	t.Setenv("JWT_EXPIRY_HOURS", "12")

	cfg := Load()
	if cfg.JWTAccessTTL != 12*time.Hour {
		t.Errorf("JWTAccessTTL = %v, want legacy fallback 12h", cfg.JWTAccessTTL)
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
