package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"guardian-tracker/api-service/api/handlers"
	"guardian-tracker/api-service/auth"
	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/config"
	"guardian-tracker/api-service/db"
	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/characters"
	"guardian-tracker/api-service/services/collections"
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

func (a *tokenRepoAdapter) Upsert(ctx context.Context, membershipID string, t *auth.EncryptedTokenRecord, prev time.Time) (bool, error) {
	return a.s.Upsert(ctx, membershipID, &db.EncryptedTokens{
		AccessTokenEnc:   t.AccessTokenEnc,
		RefreshTokenEnc:  t.RefreshTokenEnc,
		AccessExpiresAt:  t.AccessExpiresAt,
		RefreshExpiresAt: t.RefreshExpiresAt,
		KeyVersion:       t.KeyVersion,
	}, prev)
}

func (a *tokenRepoAdapter) Delete(ctx context.Context, membershipID string) error {
	return a.s.Delete(ctx, membershipID)
}

// lazyManifest satisfies handlers.manifestLookupIface and defers opening the SQLite
// file until the first call, retrying until the manifest has been downloaded.
type lazyManifest struct {
	path string
	mu   sync.RWMutex
	repo *manifestrepo.Repository
}

func (l *lazyManifest) GetItemsByHashes(hashes []uint32) (map[uint32]*bungie.InventoryItemDefinition, error) {
	l.mu.RLock()
	r := l.repo
	l.mu.RUnlock()
	if r == nil {
		l.mu.Lock()
		if l.repo == nil {
			if repo, err := manifestrepo.NewRepository(l.path); err == nil {
				l.repo = repo
			}
		}
		r = l.repo
		l.mu.Unlock()
	}
	if r == nil {
		return nil, fmt.Errorf("manifest not ready")
	}
	return r.GetItemsByHashes(hashes)
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
	cfg.Validate()

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
	tokenStore := auth.NewTokenStore(ctx, cfg.BungieClientID, cfg.BungieClientSecret, tokenRepo, tokenCipher)

	// Bungie API client
	bungieClient := bungie.NewClient(cfg.BungieAPIKey, cfg.BungieAPIBaseURL, cfg.RateLimitRPS, cfg.RateLimitBurst)

	// Manifest — EnsureReady runs in a goroutine so the HTTP server starts immediately.
	// The collections endpoint returns 503 until the manifest database is available.
	manifestService := bungie.NewManifestService(bungieClient, cfg.ManifestDBPath, cfg.ManifestCheckInterval)
	go func() {
		log.Println("Checking manifest status...")
		if err := manifestService.EnsureReady(ctx); err != nil {
			log.Printf("Warning: could not initialize manifest: %v", err)
			log.Println("Collections endpoint will return 503 until manifest is available")
		}
	}()
	manifestService.StartBackgroundUpdater(ctx)

	// Cache
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
	}

	// Services
	charactersService := characters.NewService(bungieClient, appCache, cfg.CacheTTLCollections)
	collectionsService := collections.NewService(bungieClient, manifestService, cfg.ManifestDBPath, appCache, cfg.CacheTTLCollections)

	// Shared manifest repository for weekly/records services (nil on first run until manifest downloads).
	// The wishlist uses a lazy variant that auto-connects once the manifest file appears.
	var sharedManifestRepo *manifestrepo.Repository
	if _, statErr := os.Stat(cfg.ManifestDBPath); statErr == nil {
		if repo, repoErr := manifestrepo.NewRepository(cfg.ManifestDBPath); repoErr == nil {
			sharedManifestRepo = repo
		} else {
			log.Printf("Warning: could not open manifest repository: %v", repoErr)
		}
	}
	weeklyManifestRepo := sharedManifestRepo

	// Weekly service
	var weeklyWishlist weekly.WishlistReader
	if stores.Wishlist != nil {
		weeklyWishlist = &weeklyWishlistAdapter{s: stores.Wishlist}
	}
	weeklyService := weekly.NewService(bungieClient, weeklyManifestRepo, collectionsService, weeklyWishlist, appCache)
	weeklyHandler := handlers.NewWeeklyHandler(weeklyService, tokenStore)

	// Search service — builds index asynchronously after manifest is available
	searchService := search.NewService(manifestService, cfg.ManifestDBPath)
	go searchService.BuildIndex()
	searchHandler := handlers.NewSearchHandler(searchService)

	// Records service (catalysts, crafting, seals) — shares manifest repo with weekly service
	recordsService := records.NewService(bungieClient, sharedManifestRepo, appCache, cfg.CacheTTLRecords)
	recordsHandler := handlers.NewRecordsHandler(recordsService, tokenStore)

	// Handlers
	authHandler := handlers.NewAuthHandler(ctx, jwtHelper, tokenStore, cfg, stores.Users, appCache, revoker)
	// Wishlist uses a lazy manifest that auto-connects once the manifest file is available,
	// so items are enriched even when the manifest downloads after the service starts.
	wishlistHandler := handlers.NewWishlistHandler(stores.Wishlist, &lazyManifest{path: cfg.ManifestDBPath}, stores.Prefs)
	healthHandler := handlers.NewHealthHandler(manifestService)
	charactersHandler := handlers.NewCharactersHandler(charactersService, tokenStore)
	collectionsHandler := handlers.NewCollectionsHandler(collectionsService, tokenStore, appCache)

	// Router
	router := gin.Default()
	router.Use(corsMiddleware(cfg.CORSAllowedOrigins))

	router.GET("/health", healthHandler.Health)
	router.GET("/ready", healthHandler.Ready)

	api := router.Group("/api")
	{
		api.GET("/manifest/status", healthHandler.ManifestStatus)

		// Auth
		api.GET("/auth/bungie", authHandler.GetBungieAuthURL)
		api.POST("/auth/bungie/callback", authHandler.BungieCallback)
		api.POST("/auth/refresh", authHandler.RefreshToken)
		api.GET("/auth/validate", jwtHelper.Middleware(revoker), authHandler.ValidateToken)
		api.GET("/auth/profile", jwtHelper.Middleware(revoker), authHandler.GetProfile)
		api.POST("/auth/logout", jwtHelper.Middleware(revoker), authHandler.Logout)

		// Wishlist
		api.GET("/wishlist", jwtHelper.Middleware(revoker), wishlistHandler.GetWishlist)
		api.POST("/wishlist", jwtHelper.Middleware(revoker), wishlistHandler.AddToWishlist)
		api.PUT("/wishlist/:id", jwtHelper.Middleware(revoker), wishlistHandler.UpdateWishlistItem)
		api.DELETE("/wishlist/:id", jwtHelper.Middleware(revoker), wishlistHandler.RemoveFromWishlist)

		// Preferences
		api.GET("/preferences", jwtHelper.Middleware(revoker), wishlistHandler.GetPreferences)
		api.PUT("/preferences", jwtHelper.Middleware(revoker), wishlistHandler.UpdatePreferences)

		// Characters
		api.GET("/characters/:membershipType/:membershipId", jwtHelper.Middleware(revoker), charactersHandler.GetCharacters)

		// Collections
		api.GET("/collections/:membershipType/:membershipId", jwtHelper.Middleware(revoker), collectionsHandler.GetCollections)
		api.POST("/collections/:membershipType/:membershipId/refresh", jwtHelper.Middleware(revoker), collectionsHandler.RefreshCollections)

		// Weekly recommendations
		api.GET("/weekly/recommendations", jwtHelper.Middleware(revoker), weeklyHandler.GetWeekly)

		// Item search (in-memory index from manifest)
		api.GET("/items/search", jwtHelper.Middleware(revoker), searchHandler.Search)

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
	collectionsService.Close()
	log.Println("Server exited gracefully")
}

func corsMiddleware(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		for _, allowed := range allowedOrigins {
			if strings.TrimSpace(allowed) == origin || allowed == "*" {
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
