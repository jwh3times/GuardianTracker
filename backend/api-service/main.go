package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"guardian-tracker/api-service/api/handlers"
	"guardian-tracker/api-service/api/middleware"
	"guardian-tracker/api-service/auth"
	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/config"
	"guardian-tracker/api-service/db"
	"guardian-tracker/api-service/observability"
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
		slog.Debug("environment file not loaded; using process environment")
	}

	cfg := config.Load()
	logger, err := observability.NewLogger(cfg.LogLevel, cfg.LogFormat, os.Stdout)
	if err != nil {
		slog.Error("logging configuration invalid", observability.Err(err))
		os.Exit(1)
	}
	slog.SetDefault(logger)
	if err := cfg.Validate(); err != nil {
		slog.Error("configuration invalid", observability.Err(err))
		os.Exit(1)
	}

	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Database — returns nil pool (not error) when DATABASE_URL is empty (degraded mode)
	pool, err := db.Connect(ctx, cfg)
	if err != nil {
		slog.Error("database connection failed", observability.Err(err))
		os.Exit(1)
	}
	if pool != nil {
		if err := db.Migrate(ctx, pool); err != nil {
			slog.Error("database migration failed", observability.Err(err))
			os.Exit(1)
		}
		slog.Info("database migrations applied")
	}
	stores := db.NewStores(pool)

	// Token cipher — nil when TOKEN_ENCRYPTION_KEY is empty (degraded mode)
	tokenCipher, err := auth.NewTokenCipher(
		cfg.TokenEncryptionKey,
		cfg.TokenEncryptionKeyVersion,
		cfg.TokenEncryptionKeyPrev,
		cfg.TokenEncryptionKeyPrevVersion,
	)
	if err != nil {
		slog.Error("token cipher initialization failed", observability.Err(err))
		os.Exit(1)
	}

	// Auth — pass DB repo + cipher when both are available
	jwtHelper := auth.NewJWTWithTTL(cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshExpiryDays)
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

	// Search service — restores a matching on-disk snapshot synchronously when
	// available, then builds asynchronously after the manifest is available.
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

	// Weekly service
	var weeklyWishlist weekly.WishlistReader
	if stores.Wishlist != nil {
		weeklyWishlist = &weeklyWishlistAdapter{s: stores.Wishlist}
	}
	weeklyService := weekly.NewService(bungieClient, manifestProvider, collectionsService, weeklyWishlist, appCache, efficiencyEngine, manifestService)
	if cfg.E2EFixedTime != nil {
		fixedTime := *cfg.E2EFixedTime
		weeklyService = weekly.NewServiceWithClock(
			bungieClient, manifestProvider, collectionsService, weeklyWishlist, appCache, efficiencyEngine, manifestService,
			func() time.Time { return fixedTime },
		)
	}
	weeklyHandler := handlers.NewWeeklyHandler(weeklyService, tokenStore)

	// Records service (catalysts, crafting, seals)
	recordsService := records.NewService(bungieClient, manifestProvider, appCache, cfg.CacheTTLRecords)
	recordsHandler := handlers.NewRecordsHandler(recordsService, tokenStore)
	searchHandler := handlers.NewSearchHandler(searchService)

	// Manifest swap enrolment. Must happen before EnsureReady can trigger a
	// download, and therefore after every participant and observer exists.
	//
	// Participants hold an OS handle on the manifest file and are closed before
	// the rename, then reopened after it (including on the rollback path).
	// manifestProvider goes first: observers may query the manifest, and every
	// lazily-rebuilding consumer reaches it through the provider.
	//
	// searchService is a participant in its own right because it opens its own
	// SQLite connection instead of sharing the provider — the omission that let a
	// full index scan straddle os.Rename.
	manifestService.RegisterParticipant(manifestProvider)
	manifestService.RegisterParticipant(searchService)

	// Observers hold manifest-derived state and are told only when a new version
	// actually landed. Each owns what that means for it; main.go deliberately
	// knows none of it.
	manifestService.RegisterObserver(recordsService)
	manifestService.RegisterObserver(weeklyService)
	manifestService.RegisterObserver(collectionsService)
	manifestService.RegisterObserver(itemsService)
	manifestService.RegisterObserver(searchService)
	manifestService.RegisterObserver(efficiencyEngine)

	go func() {
		slog.Info("checking manifest status")
		if err := manifestService.EnsureReady(ctx); err != nil {
			slog.Warn("manifest initialization failed; dependent endpoints remain unavailable", observability.Err(err))
		}
	}()
	manifestService.StartBackgroundUpdater(ctx)

	// Initial builds for the already-installed manifest. A swap triggered by
	// EnsureReady above notifies these through OnVersionChanged instead.
	go searchService.BuildIndex()
	go efficiencyEngine.BuildIndex()

	// Handlers
	// Pass audit as a true-nil interface in degraded mode so handlers' nil-guards
	// engage (a typed-nil *db.AuditStore would make `!= nil` true and panic).
	var auditLogger handlers.AuditLogger
	if stores.Audit != nil {
		auditLogger = stores.Audit
	}
	authHandler := handlers.NewAuthHandler(jwtHelper, tokenStore, cfg, stores.Users, appCache, revoker, auditLogger)
	wishlistHandler := handlers.NewWishlistHandler(stores.Wishlist, manifestProvider, stores.Prefs, weeklyService, tokenStore)
	// Pass the DB pool as a true-nil interface in degraded mode so /ready's
	// nil-guard engages (a typed-nil *pgxpool.Pool would make `!= nil` true
	// and panic on Ping) — same trap as the audit logger above.
	var dbPinger handlers.DBPinger
	if pool != nil {
		dbPinger = pool
	}
	healthHandler := handlers.NewHealthHandler(manifestService, dbPinger)
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

	// Shared flag resolver for server-side enforcement. Uses the same appCache and
	// flags:all key as AccountHandler's resolver, so both read one cache entry
	// (evicted together by AdminHandler.UpdateFlag). True-nil store in degraded mode.
	var enforceFlags *handlers.FlagResolver
	if stores.Flags != nil {
		enforceFlags = handlers.NewFlagResolver(stores.Flags, appCache)
	} else {
		enforceFlags = handlers.NewFlagResolver(nil, appCache)
	}

	// Router
	router := gin.New()
	router.Use(observability.HTTPMiddleware(logger))
	// Trust only configured proxies for X-Forwarded-For so the audit-logged client
	// IP can't be spoofed by clients. Empty list (local dev) trusts none.
	if err := router.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		slog.Warn("trusted proxy configuration rejected", observability.Err(err))
	}
	router.Use(middleware.APISecurityHeaders())
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

		// Auth responses are never cacheable. Callback and refresh both establish
		// or rotate the refresh cookie, so they also require an exact browser Origin.
		authRoutes := api.Group("/auth", middleware.NoStore())
		authRoutes.GET("/bungie", authLimiter, authHandler.GetBungieAuthURL)
		authRoutes.POST("/bungie/callback", middleware.RequireAllowedOrigin(cfg.CORSAllowedOrigins), authLimiter, authHandler.BungieCallback)
		authRoutes.POST("/refresh", middleware.RequireAllowedOrigin(cfg.CORSAllowedOrigins), authLimiter, authHandler.RefreshToken)
		authRoutes.GET("/validate", jwtHelper.Middleware(revoker), authHandler.ValidateToken)
		authRoutes.GET("/profile", jwtHelper.Middleware(revoker), authHandler.GetProfile)
		authRoutes.POST("/logout", jwtHelper.Middleware(revoker), authHandler.Logout)
		authRoutes.POST("/logout/all", jwtHelper.Middleware(revoker), authHandler.LogoutAll)

		// Wishlist
		api.GET("/wishlist", jwtHelper.Middleware(revoker), wishlistHandler.GetWishlist)
		api.POST("/wishlist", jwtHelper.Middleware(revoker), wishlistHandler.AddToWishlist)
		api.PUT("/wishlist/:id", jwtHelper.Middleware(revoker), wishlistHandler.UpdateWishlistItem)
		api.DELETE("/wishlist/:id", jwtHelper.Middleware(revoker), wishlistHandler.RemoveFromWishlist)
		api.POST("/wishlist/bulk", jwtHelper.Middleware(revoker), wishlistHandler.BulkUpdate)

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
		api.GET("/weekly/recommendations", jwtHelper.Middleware(revoker), authz.RequireFlag(enforceFlags, handlers.FlagWeeklyPlanner), weeklyHandler.GetWeekly)

		// Item search (in-memory index from manifest)
		api.GET("/items/search", jwtHelper.Middleware(revoker), authz.RequireFlag(enforceFlags, handlers.FlagGlobalSearch), searchHandler.Search)
		api.GET("/items/:itemHash", jwtHelper.Middleware(revoker), itemsHandler.GetItem)
		api.GET("/items/:itemHash/perks", jwtHelper.Middleware(revoker), itemsHandler.GetPerks)

		// Catalysts, crafting patterns, and seals
		api.GET("/catalysts/:membershipType/:membershipId", jwtHelper.Middleware(revoker), authz.RequireFlag(enforceFlags, handlers.FlagCatalystsCrafting), recordsHandler.GetCatalysts)
		api.GET("/crafting/:membershipType/:membershipId", jwtHelper.Middleware(revoker), authz.RequireFlag(enforceFlags, handlers.FlagCatalystsCrafting), recordsHandler.GetCrafting)
		api.GET("/seals/:membershipType/:membershipId", jwtHelper.Middleware(revoker), authz.RequireFlag(enforceFlags, handlers.FlagTriumphsSeals), recordsHandler.GetSeals)
	}

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		slog.Info("API service starting", "port", cfg.Port, "environment", cfg.GoEnv, "manifest_ready", manifestService.IsReady())
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("API server stopped unexpectedly", observability.Err(err))
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down API service")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown deadline exceeded", observability.Err(err))
	}
	if err := manifestProvider.Close(); err != nil {
		slog.Warn("manifest provider close failed", observability.Err(err))
	}
	slog.Info("API service stopped")
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
					slog.Error("session pruning failed", observability.Err(err))
				} else if n > 0 {
					slog.Info("expired sessions pruned", "count", n)
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
					slog.Error("audit pruning failed", observability.Err(err))
				} else if n > 0 {
					slog.Info("expired audit rows pruned", "count", n)
				}
			}
		}
	}()
}

func corsMiddleware(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		c.Header("Vary", "Origin")
		if middleware.OriginAllowed(origin, allowedOrigins) {
			c.Header("Access-Control-Allow-Origin", origin)
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID")
		c.Header("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
