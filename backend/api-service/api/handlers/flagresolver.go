package handlers

import (
	"context"
	"time"

	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/db"
)

// flagsCacheKey caches the full flag list so flag reads are one DB query per
// minute, not per request. Evicted on any admin flag change (see AdminHandler.UpdateFlag).
const flagsCacheKey = "flags:all"
const flagsCacheTTL = 60 * time.Second

// Feature-flag keys enforced server-side by auth.RequireFlag. These MUST stay in
// sync with the migration 0002_roles_flags.sql seed and the frontend NAV_FLAG /
// FlaggedRoute keys. The db test TestSeededFlagsIncludeEnforced guards the seed side.
const (
	FlagWeeklyPlanner     = "weekly-planner"
	FlagGlobalSearch      = "global-search"
	FlagCatalystsCrafting = "catalysts-crafting"
	FlagTriumphsSeals     = "triumphs-seals"
)

// FlagResolver caches the feature-flag catalog and answers per-key enabled/min-tier
// lookups. Shared by GetFlags (the UI hint) and auth.RequireFlag (enforcement) so
// there is exactly one implementation and one flags:all cache entry.
type FlagResolver struct {
	flags flagLister  // nil = degraded mode (no flag table)
	cache cache.Cache // may be a NoOpCache
}

// NewFlagResolver builds a resolver. Pass a true-nil flags interface (not a typed
// nil) in degraded mode so the nil-guards engage.
func NewFlagResolver(flags flagLister, c cache.Cache) *FlagResolver {
	return &FlagResolver{flags: flags, cache: c}
}

// List returns the full flag catalog from a 60s cache, reading the store on a miss.
// In degraded mode (nil store) it returns (nil, nil) — callers treat that as "no
// flags configured" (fail open / nothing hidden).
func (r *FlagResolver) List(ctx context.Context) ([]db.FeatureFlag, error) {
	if r.flags == nil {
		return nil, nil
	}
	if r.cache != nil {
		if v, ok := r.cache.Get(flagsCacheKey); ok {
			if flags, ok := v.([]db.FeatureFlag); ok {
				return flags, nil
			}
			// Wrong-typed value in the slot: treat as a miss and re-read below.
		}
	}
	flags, err := r.flags.List(ctx)
	if err != nil {
		return nil, err
	}
	if r.cache != nil {
		r.cache.Set(flagsCacheKey, flags, flagsCacheTTL)
	}
	return flags, nil
}

// Resolve reports whether a flag is enabled and its minimum tier. found is false
// for an unknown key or in degraded mode; callers fail open on !found or err.
func (r *FlagResolver) Resolve(ctx context.Context, key string) (enabled bool, minTier int, found bool, err error) {
	flags, err := r.List(ctx)
	if err != nil {
		return false, 0, false, err
	}
	for _, f := range flags {
		if f.Key == key {
			return f.Enabled, int(f.MinTier), true, nil
		}
	}
	return false, 0, false, nil
}
