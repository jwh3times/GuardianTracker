// Package api builds the HTTP route table.
//
// It exists so the route table is reachable from a test. Everything here used
// to live in package main, which meant the two rules that make an endpoint safe
// were unassertable: that every /api route sits behind JWT verification, and
// that each flag-gated route names the flag it actually intends. Both fail
// silently when wrong — a missing JWT middleware makes an endpoint public, and
// RequireFlag deliberately fails open on a key it cannot resolve, so a typo'd
// key gates nothing.
//
// The other half of the fix is structural: JWT verification is applied once per
// group rather than repeated per route, so "forgot the middleware" is no longer
// a thing an individual route can do.
package api

import (
	"log/slog"

	"guardian-tracker/api-service/api/handlers"
	"guardian-tracker/api-service/api/middleware"
	"guardian-tracker/api-service/auth"
	"guardian-tracker/api-service/config"
	"guardian-tracker/api-service/observability"

	"github.com/gin-gonic/gin"
)

// Handlers is the full set of request handlers the route table mounts. Grouped
// into their own struct so Deps stays about policy (who may call what) rather
// than being a flat bag of twelve.
type Handlers struct {
	Health      *handlers.HealthHandler
	Auth        *handlers.AuthHandler
	Wishlist    *handlers.WishlistHandler
	Account     *handlers.AccountHandler
	Admin       *handlers.AdminHandler
	Audit       *handlers.AuditHandler
	Characters  *handlers.CharactersHandler
	Collections *handlers.CollectionsHandler
	Items       *handlers.ItemsHandler
	Weekly      *handlers.WeeklyHandler
	Records     *handlers.RecordsHandler
	Search      *handlers.SearchHandler
}

// Deps is everything NewRouter needs. It is a composition root's parameter
// list, so it is wide by nature; the value is that constructing it is now
// something a test can do.
type Deps struct {
	Config *config.Config
	Logger *slog.Logger

	// JWT and Revoker together are the authentication gate. Revoker checks the
	// session and role claims against the database and fails open when there
	// isn't one.
	JWT     *auth.JWT
	Revoker *auth.RevocationChecker

	// Authz gates by role tier and by feature flag. Flags resolves flag state;
	// a nil Flags makes every flag gate fail open (see auth.RequireFlag).
	Authz *auth.Authz
	Flags auth.FlagResolver

	Handlers Handlers
}

// NewRouter builds the complete route table.
//
// Route ordering within the function follows the group structure, not the URL
// alphabet: public routes first, then the single authenticated group, then the
// admin subgroup inside it. Read the group a route is registered on to know
// what protects it — that is the whole point of the arrangement.
func NewRouter(d Deps) *gin.Engine {
	cfg := d.Config

	router := gin.New()
	router.Use(observability.HTTPMiddleware(d.Logger))
	// Trust only configured proxies for X-Forwarded-For so the audit-logged client
	// IP can't be spoofed by clients. Empty list (local dev) trusts none.
	if err := router.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		slog.Warn("trusted proxy configuration rejected", observability.Err(err))
	}
	router.Use(middleware.APISecurityHeaders())
	router.Use(middleware.CORS(cfg.CORSAllowedOrigins))
	router.Use(middleware.MaxBodyBytes(cfg.MaxBodyBytes))

	// One shared limiter instance across the public auth endpoints (per-IP
	// buckets are inside it) — constructed once so the sweeper isn't duplicated.
	authLimiter := middleware.PerIPRateLimit(float64(cfg.AuthRateLimitRPS), cfg.AuthRateLimitBurst)

	// Liveness and readiness sit outside /api: no auth, no CORS expectations.
	router.GET("/health", d.Handlers.Health.Health)
	router.GET("/ready", d.Handlers.Health.Ready)

	api := router.Group("/api")

	// --- Public: no bearer token required. ---
	//
	// Every route registered directly on `api` (rather than on `authed` below)
	// is deliberately public. There are four, and router_test.go enumerates
	// them: adding a fifth without adding it there fails the suite.
	api.GET("/manifest/status", d.Handlers.Health.ManifestStatus)

	// Auth responses are never cacheable. Callback and refresh both establish
	// or rotate the refresh cookie, so they also require an exact browser Origin.
	authRoutes := api.Group("/auth", middleware.NoStore())
	authRoutes.GET("/bungie", authLimiter, d.Handlers.Auth.GetBungieAuthURL)
	authRoutes.POST("/bungie/callback", middleware.RequireAllowedOrigin(cfg.CORSAllowedOrigins), authLimiter, d.Handlers.Auth.BungieCallback)
	authRoutes.POST("/refresh", middleware.RequireAllowedOrigin(cfg.CORSAllowedOrigins), authLimiter, d.Handlers.Auth.RefreshToken)

	// --- Authenticated: the JWT gate is applied once, here. ---
	//
	// It used to be repeated on all 24 authenticated routes, where omitting one
	// published that endpoint with no error and no log. Registering on the
	// wrong group is now the only way to make that mistake, and it is visible
	// in the diff.
	jwtGate := d.JWT.Middleware(d.Revoker)

	authedAuthRoutes := authRoutes.Group("", jwtGate)
	authedAuthRoutes.GET("/validate", d.Handlers.Auth.ValidateToken)
	authedAuthRoutes.GET("/profile", d.Handlers.Auth.GetProfile)
	authedAuthRoutes.POST("/logout", d.Handlers.Auth.Logout)
	authedAuthRoutes.POST("/logout/all", d.Handlers.Auth.LogoutAll)

	authed := api.Group("", jwtGate)

	// Wishlist
	authed.GET("/wishlist", d.Handlers.Wishlist.GetWishlist)
	authed.POST("/wishlist", d.Handlers.Wishlist.AddToWishlist)
	authed.PUT("/wishlist/:id", d.Handlers.Wishlist.UpdateWishlistItem)
	authed.DELETE("/wishlist/:id", d.Handlers.Wishlist.RemoveFromWishlist)
	authed.POST("/wishlist/bulk", d.Handlers.Wishlist.BulkUpdate)

	// Preferences
	authed.GET("/preferences", d.Handlers.Wishlist.GetPreferences)
	authed.PUT("/preferences", d.Handlers.Wishlist.UpdatePreferences)

	// Account self-service role opt-in + resolved feature-flag state
	authed.PUT("/account/role", d.Handlers.Account.SetRole)
	authed.GET("/flags", d.Handlers.Account.GetFlags)

	// Admin console. RequireAdmin 403s a non-admin and 503s in degraded mode,
	// where role claims are not authoritative.
	admin := authed.Group("/admin", d.Authz.RequireAdmin())
	admin.GET("/users", d.Handlers.Admin.ListUsers)
	admin.PUT("/users/:id/role", d.Handlers.Admin.SetUserRole)
	admin.GET("/flags", d.Handlers.Admin.ListFlags)
	admin.PUT("/flags/:key", d.Handlers.Admin.UpdateFlag)
	admin.GET("/audit", d.Handlers.Audit.ListAudit)

	// Characters
	authed.GET("/characters/:membershipType/:membershipId", d.Handlers.Characters.GetCharacters)

	// Collections
	authed.GET("/collections/:membershipType/:membershipId", d.Handlers.Collections.GetCollections)
	authed.POST("/collections/:membershipType/:membershipId/refresh", d.Handlers.Collections.RefreshCollections)

	// Flag-gated routes. RequireFlag must compose after the JWT gate, which sets
	// the role in the context — the group supplies that ordering. It fails open
	// on an unresolvable key, so a wrong key here gates nothing; router_test.go
	// asserts each route against the flag it claims.
	authed.GET("/weekly/recommendations", d.Authz.RequireFlag(d.Flags, handlers.FlagWeeklyPlanner), d.Handlers.Weekly.GetWeekly)

	// Item search (in-memory index from manifest)
	authed.GET("/items/search", d.Authz.RequireFlag(d.Flags, handlers.FlagGlobalSearch), d.Handlers.Search.Search)
	authed.GET("/items/:itemHash", d.Handlers.Items.GetItem)
	authed.GET("/items/:itemHash/perks", d.Handlers.Items.GetPerks)

	// Catalysts, crafting patterns, and seals
	authed.GET("/catalysts/:membershipType/:membershipId", d.Authz.RequireFlag(d.Flags, handlers.FlagCatalystsCrafting), d.Handlers.Records.GetCatalysts)
	authed.GET("/crafting/:membershipType/:membershipId", d.Authz.RequireFlag(d.Flags, handlers.FlagCatalystsCrafting), d.Handlers.Records.GetCrafting)
	authed.GET("/seals/:membershipType/:membershipId", d.Authz.RequireFlag(d.Flags, handlers.FlagTriumphsSeals), d.Handlers.Records.GetSeals)

	return router
}
