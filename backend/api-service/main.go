package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"guardian-tracker/api-service/api/handlers"
	"guardian-tracker/api-service/api/middleware"
	"guardian-tracker/api-service/auth"
	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/config"
	"guardian-tracker/api-service/db"
	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/characters"
	"guardian-tracker/api-service/services/collections"
	"guardian-tracker/api-service/services/efficiency"
	"guardian-tracker/api-service/services/items"
	manifestrepo "guardian-tracker/api-service/services/manifest"
	"guardian-tracker/api-service/services/records"
	"guardian-tracker/api-service/services/search"
	"guardian-tracker/api-service/services/weekly"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// tokenRepoAdapter wraps db.BungieTokenStore to satisfy auth.TokenRepo.
// This adapter lives in main.go to avoid an import cycle between auth and db packages.
type tokenRepoAdapter struct{ s *db.BungieTokenStore }

func (a *tokenRepoAdapter) Get(ctx context.Context, membershipID string) (*auth.EncryptedTokenRecord, error) {
	t, err := a.s.Get(ctx, membershipID)
	if err != nil {
		if errors.Is(err, db.ErrTokensNotFound) {
			return nil, auth.ErrTokensNotFound
		}
		return nil, err
	}
	return &auth.EncryptedTokenRecord{
		AccessTokenEnc:   t.AccessTokenEnc,
		RefreshTokenEnc:  t.RefreshTokenEnc,
		AccessExpiresAt:  t.AccessExpiresAt,
		RefreshExpiresAt: t.RefreshExpiresAt,
		KeyVersion:       t.KeyVersion,
		UpdatedAt:        t.UpdatedAt,
	}, nil
}

func (a *tokenRepoAdapter) Upsert(ctx context.Context, membershipID string, t *auth.EncryptedTokenRecord, prev time.Time) (time.Time, bool, error) {
	at, ok, err := a.s.Upsert(ctx, membershipID, &db.EncryptedTokens{
		AccessTokenEnc:   t.AccessTokenEnc,
		RefreshTokenEnc:  t.RefreshTokenEnc,
		AccessExpiresAt:  t.AccessExpiresAt,
		RefreshExpiresAt: t.RefreshExpiresAt,
		KeyVersion:       t.KeyVersion,
	}, prev)
	if errors.Is(err, db.ErrNoUserRow) {
		return at, ok, auth.ErrNoUserRow
	}
	return at, ok, err
}

func (a *tokenRepoAdapter) Delete(ctx context.Context, membershipID string) error {
	return a.s.Delete(ctx, membershipID)
}

// weeklyWishlistAdapter wraps db.WishlistStore to satisfy weekly.WishlistReader.
// This adapter lives in main.go to avoid an import cycle between weekly and db packages.
type weeklyWishlistAdapter struct{ s *db.WishlistStore }

func (a *weeklyWishlistAdapter) GetUserID(ctx context.Context, membershipID string) (int64, error) {
	return a.s.GetUserID(ctx, membershipID)
}

func (a *weeklyWishlistAdapter) List(ctx context.Context, userID int64) ([]weekly.WishlistItem, error) {
	items, err := a.s.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]weekly.WishlistItem, len(items))
	for i, it := range items {
		out[i] = weekly.WishlistItem{ItemHash: it.ItemHash}
	}
	return out, nil
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration invalid: %v", err)
	}

	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Database — returns nil pool (not error) when DATABASE_URL is empty (degraded mode)
	pool, err := db.Connect(ctx, cfg)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	if pool != nil {
		if err := db.Migrate(ctx, pool); err != nil {
			log.Fatalf("Database migration failed: %v", err)
		}
		log.Println("Database migrations applied successfully")
	}
	stores := db.NewStores(pool)

	// Token cipher — nil when TOKEN_ENCRYPTION_KEY is empty (degraded mode)
	tokenCipher, err := auth.NewTokenCipher(cfg.TokenEncryptionKey, cfg.TokenEncryptionKeyPrev)
	if err != nil {
		log.Fatalf("Token cipher initialization failed: %v", err)
	}

	// Auth — pass DB repo + cipher when both are available
	jwtHelper := auth.NewJWT(cfg.JWTSecret, cfg.JWTExpiryHours, cfg.JWTRefreshExpiryDays)
	var tokenRepo auth.TokenRepo
	if stores.Tokens != nil && tokenCipher != nil {
		tokenRepo = &tokenRepoAdapter{s: stores.Tokens}
	}
	tokenStore := auth.NewTokenStore(ctx, cfg.BungieClientID, cfg.BungieClientSecret, cfg.BungieTokenURL, tokenRepo, tokenCipher)

	// Bungie API client
	bungieClient := bungie.NewClient(cfg.BungieAPIKey, cfg.BungieAPIBaseURL, cfg.RateLimitRPS, cfg.RateLimitBurst)
	bungieClient.SetCDNBaseURL(cfg.BungieCDNBaseURL)

	// Manifest — EnsureReady runs in a goroutine so the HTTP server starts immediately.
	// The collections endpoint returns 503 until the manifest database is available.
	manifestService := bungie.NewManifestService(bungieClient, cfg.ManifestDBPath, cfg.ManifestCheckInterval)

	// Single manifest repository shared by all consumers: opens lazily once the
	// manifest file exists (no restart needed after a cold-start download) and
	// reconnects across the hourly manifest swap via the hooks below.
	manifestProvider := manifestrepo.NewProvider(cfg.ManifestDBPath)
	itemsService := items.NewService(manifestProvider)

	// Search service — builds its index asynchronously after the manifest is available
	searchService := search.NewService(manifestService, cfg.ManifestDBPath)

	// Efficiency engine — scores items by acquisition difficulty; index built async
	efficiencyEngine := efficiency.NewEngine(manifestProvider, manifestService)

	// Cache — created before the swap hooks so the after-swap hook can evict
	// manifest-derived cache entries.
	var appCache cache.Cache
	if cfg.CacheEnabled {
		appCache = cache.NewMemoryCache(cfg.CacheTTLCollections, 10*time.Minute)
	} else {
		appCache = cache.NewNoOpCache()
	}

	// Revocation checker — nil store = skip version check (degraded mode)
	var revoker *auth.RevocationChecker
	if stores.Users != nil {
		revoker = auth.NewRevocationChecker(stores.Users, appCache)
		startSessionPruner(ctx, stores.Users)
		if stores.Audit != nil {
			startAuditPruner(ctx, stores.Audit, cfg.AuditRetentionDays)
		}
	}

	// Tier gating — enabled only when a DB-backed user store is available; in
	// degraded mode role-gated endpoints return 503 (roles are unavailable).
	authz := auth.NewAuthz(stores.Users != nil)

	// Services — all manifest consumers share manifestProvider (lazy open,
	// reconnects across manifest swaps).
	charactersService := characters.NewService(bungieClient, appCache, cfg.CacheTTLCollections)
	collectionsService := collections.NewService(bungieClient, manifestService, manifestProvider, appCache, cfg.CacheTTLCollections)

	// Hooks must be registered before EnsureReady can trigger a download: close
	// SQLite handles before the file swap (the rename fails on Windows otherwise),
	// reopen + rebuild the search index after the new version is installed, and
	// evict manifest-derived caches (e.g. weapon types and the collections tree)
	// so they don't serve stale data until their TTL expires (B10).
	manifestService.RegisterSwapHooks(
		manifestProvider.CloseForSwap,
		func(version string) {
			if err := manifestProvider.Reopen(); err != nil {
				log.Printf("Warning: manifest provider reopen failed: %v", err)
			}
			appCache.Delete(records.WeaponTypesCacheKey)
			appCache.Delete(records.ExoticWeaponsCacheKey)
			collectionsService.InvalidateTreeCache()
			go searchService.BuildIndex()
			go efficiencyEngine.BuildIndex()
		},
	)
	manifestService.RegisterSwapHooks(nil, func(version string) {
		itemsService.InvalidateCache()
	})

	go func() {
		log.Println("Checking manifest status...")
		if err := manifestService.EnsureReady(ctx); err != nil {
			log.Printf("Warning: could not initialize manifest: %v", err)
			log.Println("Collections endpoint will return 503 until manifest is available")
		}
	}()
	manifestService.StartBackgroundUpdater(ctx)

	// Weekly service
	var weeklyWishlist weekly.WishlistReader
	if stores.Wishlist != nil {
		weeklyWishlist = &weeklyWishlistAdapter{s: stores.Wishlist}
	}
	weeklyService := weekly.NewService(bungieClient, manifestProvider, collectionsService, weeklyWishlist, appCache, efficiencyEngine)
	weeklyHandler := handlers.NewWeeklyHandler(weeklyService, tokenStore)

	go searchService.BuildIndex()
	go efficiencyEngine.BuildIndex()
	searchHandler := handlers.NewSearchHandler(searchService)

	// Records service (catalysts, crafting, seals)
	recordsService := records.NewService(bungieClient, manifestProvider, appCache, cfg.CacheTTLRecords)
	recordsHandler := handlers.NewRecordsHandler(recordsService, tokenStore)

	// Handlers
	// Pass audit as a true-nil interface in degraded mode so handlers' nil-guards
	// engage (a typed-nil *db.AuditStore would make `!= nil` true and panic).
	var auditLogger handlers.AuditLogger
	if stores.Audit != nil {
		auditLogger = stores.Audit
	}
	authHandler := handlers.NewAuthHandler(jwtHelper, tokenStore, cfg, stores.Users, appCache, revoker, auditLogger)
	wishlistHandler := handlers.NewWishlistHandler(stores.Wishlist, manifestProvider, stores.Prefs, weeklyService)
	healthHandler := handlers.NewHealthHandler(manifestService)
	charactersHandler := handlers.NewCharactersHandler(charactersService, tokenStore)
	collectionsHandler := handlers.NewCollectionsHandler(collectionsService, charactersService, recordsService, tokenStore, weeklyService)
	itemsHandler := handlers.NewItemsHandler(itemsService)

	// Roles, feature flags & admin console. In degraded mode pass true-nil stores
	// so the handlers' nil-guards engage (vs. a typed-nil interface).
	var accountHandler *handlers.AccountHandler
	var adminHandler *handlers.AdminHandler
	if stores.Users != nil {
		accountHandler = handlers.NewAccountHandler(stores.Users, stores.Flags, appCache, auditLogger)
		adminHandler = handlers.NewAdminHandler(stores.Users, stores.Flags, appCache)
	} else {
		accountHandler = handlers.NewAccountHandler(nil, nil, appCache, auditLogger)
		adminHandler = handlers.NewAdminHandler(nil, nil, appCache)
	}

	var auditHandler *handlers.AuditHandler
	if stores.Audit != nil {
		auditHandler = handlers.NewAuditHandler(stores.Audit)
	} else {
		auditHandler = handlers.NewAuditHandler(nil)
	}

	// Router
	router := gin.Default()
	// Trust only configured proxies for X-Forwarded-For so the audit-logged client
	// IP can't be spoofed by clients. Empty list (local dev) trusts none.
	if err := router.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		log.Printf("Warning: SetTrustedProxies(%v): %v", cfg.TrustedProxies, err)
	}
	router.Use(corsMiddleware(cfg.CORSAllowedOrigins))
	router.Use(middleware.MaxBodyBytes(cfg.MaxBodyBytes))
	// One shared limiter instance across the public auth endpoints (per-IP
	// buckets are inside it) — constructed once so the sweeper isn't duplicated.
	authLimiter := middleware.PerIPRateLimit(float64(cfg.AuthRateLimitRPS), cfg.AuthRateLimitBurst)

	router.GET("/health", healthHandler.Health)
	router.GET("/ready", healthHandler.Ready)

	api := router.Group("/api")
	{
		api.GET("/manifest/status", healthHandler.ManifestStatus)

		// Auth
		api.GET("/auth/bungie", authLimiter, authHandler.GetBungieAuthURL)
		api.POST("/auth/bungie/callback", authLimiter, authHandler.BungieCallback)
		api.POST("/auth/refresh", authLimiter, authHandler.RefreshToken)
		api.GET("/auth/validate", jwtHelper.Middleware(revoker), authHandler.ValidateToken)
		api.GET("/auth/profile", jwtHelper.Middleware(revoker), authHandler.GetProfile)
		api.POST("/auth/logout", jwtHelper.Middleware(revoker), authHandler.Logout)
		api.POST("/auth/logout/all", jwtHelper.Middleware(revoker), authHandler.LogoutAll)

		// Wishlist
		api.GET("/wishlist", jwtHelper.Middleware(revoker), wishlistHandler.GetWishlist)
		api.POST("/wishlist", jwtHelper.Middleware(revoker), wishlistHandler.AddToWishlist)
		api.PUT("/wishlist/:id", jwtHelper.Middleware(revoker), wishlistHandler.UpdateWishlistItem)
		api.DELETE("/wishlist/:id", jwtHelper.Middleware(revoker), wishlistHandler.RemoveFromWishlist)

		// Preferences
		api.GET("/preferences", jwtHelper.Middleware(revoker), wishlistHandler.GetPreferences)
		api.PUT("/preferences", jwtHelper.Middleware(revoker), wishlistHandler.UpdatePreferences)

		// Account self-service role opt-in + resolved feature-flag state
		api.PUT("/account/role", jwtHelper.Middleware(revoker), accountHandler.SetRole)
		api.GET("/flags", jwtHelper.Middleware(revoker), accountHandler.GetFlags)

		// Admin console — JWT + RequireAdmin (503 in degraded mode)
		admin := api.Group("/admin", jwtHelper.Middleware(revoker), authz.RequireAdmin())
		{
			admin.GET("/users", adminHandler.ListUsers)
			admin.PUT("/users/:id/role", adminHandler.SetUserRole)
			admin.GET("/flags", adminHandler.ListFlags)
			admin.PUT("/flags/:key", adminHandler.UpdateFlag)
			admin.GET("/audit", auditHandler.ListAudit)
		}

		// Characters
		api.GET("/characters/:membershipType/:membershipId", jwtHelper.Middleware(revoker), charactersHandler.GetCharacters)

		// Collections
		api.GET("/collections/:membershipType/:membershipId", jwtHelper.Middleware(revoker), collectionsHandler.GetCollections)
		api.POST("/collections/:membershipType/:membershipId/refresh", jwtHelper.Middleware(revoker), collectionsHandler.RefreshCollections)

		// Weekly recommendations
		api.GET("/weekly/recommendations", jwtHelper.Middleware(revoker), weeklyHandler.GetWeekly)

		// Item search (in-memory index from manifest)
		api.GET("/items/search", jwtHelper.Middleware(revoker), searchHandler.Search)
		api.GET("/items/:itemHash", jwtHelper.Middleware(revoker), itemsHandler.GetItem)
		api.GET("/items/:itemHash/perks", jwtHelper.Middleware(revoker), itemsHandler.GetPerks)

		// Catalysts, crafting patterns, and seals
		api.GET("/catalysts/:membershipType/:membershipId", jwtHelper.Middleware(revoker), recordsHandler.GetCatalysts)
		api.GET("/crafting/:membershipType/:membershipId", jwtHelper.Middleware(revoker), recordsHandler.GetCrafting)
		api.GET("/seals/:membershipType/:membershipId", jwtHelper.Middleware(revoker), recordsHandler.GetSeals)
	}

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("API service starting on port %s (%s)", cfg.Port, cfg.GoEnv)
		log.Printf("Manifest ready: %v", manifestService.IsReady())
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}
	if err := manifestProvider.Close(); err != nil {
		log.Printf("Warning: error closing manifest provider: %v", err)
	}
	log.Println("Server exited gracefully")
}

// startSessionPruner periodically deletes expired refresh sessions so abandoned
// sessions (never rotated, never logged out) don't accumulate. It runs until ctx is
// cancelled at shutdown.
func startSessionPruner(ctx context.Context, users *db.UserStore) {
	const interval = time.Hour
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				n, err := users.DeleteExpiredSessions(ctx)
				if err != nil {
					log.Printf("session pruner: %v", err)
				} else if n > 0 {
					log.Printf("session pruner: removed %d expired session(s)", n)
				}
			}
		}
	}()
}

// startAuditPruner periodically deletes audit_log rows older than retentionDays,
// bounding table growth and the retention window for stored client IPs. Runs until
// ctx is cancelled at shutdown.
func startAuditPruner(ctx context.Context, audit *db.AuditStore, retentionDays int) {
	const interval = time.Hour
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cutoff := time.Now().AddDate(0, 0, -retentionDays)
				n, err := audit.DeleteOlderThan(ctx, cutoff)
				if err != nil {
					log.Printf("audit pruner: %v", err)
				} else if n > 0 {
					log.Printf("audit pruner: removed %d expired audit row(s)", n)
				}
			}
		}
	}()
}

func corsMiddleware(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		for _, allowed := range allowedOrigins {
			if strings.TrimSpace(allowed) == origin {
				c.Header("Access-Control-Allow-Origin", origin)
				break
			}
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
