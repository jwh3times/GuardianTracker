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
