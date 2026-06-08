package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"guardian-tracker/bungie-service/api/handlers"
	"guardian-tracker/bungie-service/cache"
	"guardian-tracker/bungie-service/config"
	"guardian-tracker/bungie-service/services/auth"
	"guardian-tracker/bungie-service/services/bungie"
	"guardian-tracker/bungie-service/services/characters"
	"guardian-tracker/bungie-service/services/collections"
	"guardian-tracker/bungie-service/services/manifest"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Load configuration
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Set Gin mode
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Bungie API client
	bungieClient := bungie.NewClient(
		cfg.BungieAPIKey,
		cfg.BungieAPIBaseURL,
		cfg.RateLimitRPS,
		cfg.RateLimitBurst,
	)

	// Initialize manifest service
	manifestService := bungie.NewManifestService(
		bungieClient,
		cfg.ManifestDBPath,
		cfg.ManifestCheckInterval,
	)

	// Download manifest if not present (blocking on startup)
	log.Println("Checking manifest status...")
	if err := manifestService.EnsureReady(ctx); err != nil {
		log.Printf("Warning: Could not initialize manifest: %v", err)
		log.Println("Service will start, but collections endpoint will fail until manifest is available")
	}

	// Start background manifest updater
	manifestService.StartBackgroundUpdater(ctx)

	// Initialize manifest repository (will fail if manifest not downloaded)
	var manifestRepo *manifest.Repository
	if manifestService.IsReady() {
		var err error
		manifestRepo, err = manifest.NewRepository(cfg.ManifestDBPath)
		if err != nil {
			log.Printf("Warning: Could not open manifest repository: %v", err)
		}
	}

	// Initialize cache
	var appCache cache.Cache
	if cfg.CacheEnabled {
		appCache = cache.NewMemoryCache(cfg.CacheTTLCollections, 10*time.Minute)
	} else {
		appCache = cache.NewNoOpCache()
	}

	// Initialize collections service
	var collectionsService *collections.Service
	if manifestRepo != nil {
		collectionsService = collections.NewService(
			bungieClient,
			manifestService,
			manifestRepo,
			appCache,
			cfg.CacheTTLCollections,
		)
	}

	// Initialize characters service (independent of the manifest)
	charactersService := characters.NewService(
		bungieClient,
		appCache,
		cfg.CacheTTLCollections,
	)

	// Initialize auth client for fetching Bungie tokens
	authClient := auth.NewClient(
		cfg.AuthServiceURL,
		cfg.InternalAPIKey,
		cfg.JWTSecret,
	)

	// Initialize handlers
	healthHandler := handlers.NewHealthHandler(manifestService)
	charactersHandler := handlers.NewCharactersHandler(charactersService, authClient)
	var collectionsHandler *handlers.CollectionsHandler
	if collectionsService != nil {
		collectionsHandler = handlers.NewCollectionsHandler(collectionsService, authClient)
	}

	// Setup router
	router := gin.Default()

	// Add CORS middleware
	router.Use(corsMiddleware(cfg.CORSAllowedOrigins))

	// Health endpoints
	router.GET("/health", healthHandler.Health)
	router.GET("/ready", healthHandler.Ready)

	// API routes
	api := router.Group("/api")
	{
		// Manifest status
		api.GET("/manifest/status", healthHandler.ManifestStatus)

		// Characters (does not depend on the manifest)
		api.GET("/characters/:membershipType/:membershipId", charactersHandler.GetCharacters)

		// Collections endpoints
		if collectionsHandler != nil {
			api.GET("/collections/:membershipType/:membershipId", collectionsHandler.GetCollections)
			api.POST("/collections/:membershipType/:membershipId/refresh", collectionsHandler.RefreshCollections)
		} else {
			// Fallback if manifest isn't ready
			api.GET("/collections/:membershipType/:membershipId", func(c *gin.Context) {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"error": "Manifest database not ready. Please try again later.",
					"code":  "MANIFEST_NOT_READY",
				})
			})
		}

		// Weekly recommendations (placeholder for now)
		api.GET("/weekly/recommendations", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"vendors":    []interface{}{},
				"activities": []interface{}{},
				"pursuits":   []interface{}{},
			})
		})

		// Item search (placeholder for now)
		api.GET("/items/search", func(c *gin.Context) {
			c.JSON(http.StatusOK, []interface{}{})
		})
	}

	// Create HTTP server with timeouts
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second, // Longer timeout for collection queries
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Bungie service starting on port %s", cfg.Port)
		log.Printf("Environment: %s", cfg.GoEnv)
		log.Printf("Manifest ready: %v", manifestService.IsReady())

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	cancel() // Cancel background operations

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	// Cleanup
	if manifestRepo != nil {
		manifestRepo.Close()
	}

	log.Println("Server exited gracefully")
}

// corsMiddleware handles CORS
func corsMiddleware(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if allowedOrigin == origin || allowedOrigin == "*" {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
