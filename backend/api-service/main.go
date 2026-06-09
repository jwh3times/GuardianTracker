package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"guardian-tracker/api-service/api/handlers"
	"guardian-tracker/api-service/auth"
	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/config"
	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/characters"
	"guardian-tracker/api-service/services/collections"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

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

	// Auth — C9: pass ctx so cleanupLoop exits on shutdown
	jwtHelper := auth.NewJWT(cfg.JWTSecret, cfg.JWTExpiryHours, cfg.JWTRefreshExpiryDays)
	tokenStore := auth.NewTokenStore(ctx, cfg.BungieClientID, cfg.BungieClientSecret)

	// Bungie API client
	bungieClient := bungie.NewClient(cfg.BungieAPIKey, cfg.BungieAPIBaseURL, cfg.RateLimitRPS, cfg.RateLimitBurst)

	// Manifest — C8: EnsureReady runs in a goroutine so the HTTP server starts immediately.
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

	// Services — C5: collectionsService opens the SQLite repo lazily on first request
	// after the manifest is ready, so no nil-conditional setup is needed here.
	charactersService := characters.NewService(bungieClient, appCache, cfg.CacheTTLCollections)
	collectionsService := collections.NewService(bungieClient, manifestService, cfg.ManifestDBPath, appCache, cfg.CacheTTLCollections)

	// Handlers — C9: pass ctx to NewAuthHandler for csrfStore cleanup goroutine
	authHandler := handlers.NewAuthHandler(ctx, jwtHelper, tokenStore, cfg)
	wishlistHandler := handlers.NewWishlistHandler()
	healthHandler := handlers.NewHealthHandler(manifestService)
	// C1: jwt field removed from characters/collections handlers — middleware handles JWT
	charactersHandler := handlers.NewCharactersHandler(charactersService, tokenStore)
	collectionsHandler := handlers.NewCollectionsHandler(collectionsService, tokenStore)

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
		api.GET("/auth/validate", jwtHelper.Middleware(), authHandler.ValidateToken)
		api.GET("/auth/profile", jwtHelper.Middleware(), authHandler.GetProfile)

		// Wishlist (stubs)
		api.GET("/wishlist", jwtHelper.Middleware(), wishlistHandler.GetWishlist)
		api.POST("/wishlist", jwtHelper.Middleware(), wishlistHandler.AddToWishlist)
		api.DELETE("/wishlist/:id", jwtHelper.Middleware(), wishlistHandler.RemoveFromWishlist)

		// Characters — C1: middleware validates JWT and enforces access-token type
		api.GET("/characters/:membershipType/:membershipId", jwtHelper.Middleware(), charactersHandler.GetCharacters)

		// Collections — C1+C2: middleware validates JWT and enforces access-token type;
		// handler returns 503 until manifest is ready (C5)
		api.GET("/collections/:membershipType/:membershipId", jwtHelper.Middleware(), collectionsHandler.GetCollections)
		api.POST("/collections/:membershipType/:membershipId/refresh", jwtHelper.Middleware(), collectionsHandler.RefreshCollections)

		// Placeholders
		api.GET("/weekly/recommendations", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"vendors":    []interface{}{},
				"activities": []interface{}{},
				"pursuits":   []interface{}{},
			})
		})
		api.GET("/items/search", func(c *gin.Context) {
			c.JSON(http.StatusOK, []interface{}{})
		})
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
