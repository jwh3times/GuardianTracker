// Command api-service is the Guardian Tracker backend.
//
// This file is a composition root and nothing else: load configuration,
// construct every service and handler, declare who participates in the manifest
// swap, hand the result to api.NewRouter, serve, shut down. Logic that used to
// live here — the route table, the CORS middleware, the db-to-consumer adapters,
// the background pruners — moved into packages where it can be tested.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"guardian-tracker/api-service/api"
	"guardian-tracker/api-service/api/handlers"
	"guardian-tracker/api-service/auth"
	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/config"
	"guardian-tracker/api-service/db"
	"guardian-tracker/api-service/db/adapters"
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

	// Auth — the token repo is always wired; without a cipher there is nothing
	// safe to persist, so that stays a real condition.
	jwtHelper := auth.NewJWTWithTTL(cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshExpiryDays)
	var tokenRepo auth.TokenRepo
	if tokenCipher != nil {
		tokenRepo = adapters.NewTokenRepo(stores.Tokens)
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
	// reconnects across the hourly manifest swap as a registered participant.
	manifestProvider := manifestrepo.NewProvider(cfg.ManifestDBPath)
	itemsService := items.NewService(manifestProvider)

	// Search service — restores a matching on-disk snapshot synchronously when
	// available, then builds asynchronously after the manifest is available.
	searchService := search.NewService(manifestService, cfg.ManifestDBPath)

	// Efficiency engine — scores items by acquisition difficulty; index built async
	efficiencyEngine := efficiency.NewEngine(manifestProvider, manifestService)

	// Cache — created before the swap registrations below so observers can evict
	// manifest-derived cache entries.
	var appCache cache.Cache
	if cfg.CacheEnabled {
		appCache = cache.NewMemoryCache(cfg.CacheTTLCollections, 10*time.Minute)
	} else {
		appCache = cache.NewNoOpCache()
	}

	// The revocation checker is always wired: it fails open on a store error, so
	// in degraded mode it simply skips the version check. The pruners are gated
	// on Available() because a background loop that can only ever fail is noise,
	// not resilience.
	revoker := auth.NewRevocationChecker(stores.Users, appCache)
	if stores.Available() {
		db.StartSessionPruner(ctx, stores.Users, db.DefaultPruneInterval)
		db.StartAuditPruner(ctx, stores.Audit, cfg.AuditRetentionDays, db.DefaultPruneInterval)
	}

	// Tier gating. Not the same question as "would a query fail": RequireTier
	// reads the role claim from the JWT and never touches a store. What it needs
	// to know is whether those claims are authoritative, which they are only when
	// roles can be read back from the database.
	authz := auth.NewAuthz(stores.Available())

	// Services — all manifest consumers share manifestProvider (lazy open,
	// reconnects across manifest swaps).
	charactersService := characters.NewService(bungieClient, appCache, cfg.CacheTTLCollections)
	collectionsService := collections.NewService(bungieClient, manifestService, manifestProvider, appCache, cfg.CacheTTLCollections)

	// Weekly service
	weeklyWishlist := adapters.NewWeeklyWishlist(stores.Wishlist)
	weeklyService := weekly.NewService(bungieClient, manifestProvider, collectionsService, weeklyWishlist, appCache, efficiencyEngine, manifestService)
	if cfg.E2EFixedTime != nil {
		fixedTime := *cfg.E2EFixedTime
		weeklyService = weekly.NewServiceWithClock(
			bungieClient, manifestProvider, collectionsService, weeklyWishlist, appCache, efficiencyEngine, manifestService,
			func() time.Time { return fixedTime },
		)
	}

	// Records service (catalysts, crafting, seals)
	recordsService := records.NewService(bungieClient, manifestProvider, appCache, cfg.CacheTTLRecords)

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
	auditLogger := handlers.AuditLogger(stores.Audit)

	// Shared flag resolver for server-side enforcement. Uses the same appCache and
	// flags:all key as AccountHandler's resolver, so both read one cache entry
	// (evicted together by AdminHandler.UpdateFlag).
	enforceFlags := handlers.NewFlagResolver(stores.Flags, appCache)

	router := api.NewRouter(api.Deps{
		Config:  cfg,
		Logger:  logger,
		JWT:     jwtHelper,
		Revoker: revoker,
		Authz:   authz,
		Flags:   enforceFlags,
		Handlers: api.Handlers{
			Health:      handlers.NewHealthHandler(manifestService, stores.Pinger),
			Auth:        handlers.NewAuthHandler(jwtHelper, tokenStore, cfg, stores.Users, appCache, revoker, auditLogger),
			Wishlist:    handlers.NewWishlistHandler(stores.Wishlist, manifestProvider, stores.Prefs, weeklyService, tokenStore),
			Account:     handlers.NewAccountHandler(stores.Users, stores.Flags, appCache, auditLogger),
			Admin:       handlers.NewAdminHandler(stores.Users, stores.Flags, appCache),
			Audit:       handlers.NewAuditHandler(stores.Audit),
			Characters:  handlers.NewCharactersHandler(charactersService, tokenStore),
			Collections: handlers.NewCollectionsHandler(collectionsService, charactersService, recordsService, tokenStore, weeklyService),
			Items:       handlers.NewItemsHandler(itemsService),
			Weekly:      handlers.NewWeeklyHandler(weeklyService, tokenStore),
			Records:     handlers.NewRecordsHandler(recordsService, tokenStore),
			Search:      handlers.NewSearchHandler(searchService),
		},
	})

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
