package efficiency

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"guardian-tracker/api-service/observability"
	"guardian-tracker/api-service/services/manifest"
)

// collectibleSource provides the joined collectible+item rows for index building.
// Satisfied by *manifest.Provider.
type collectibleSource interface {
	GetAllCollectiblesWithItems() ([]manifest.CollectibleWithItem, error)
}

// versioner reports the current manifest version. Satisfied by *bungie.ManifestService.
type versioner interface {
	Version() string
}

// Engine holds the per-manifest-version source-bucket index and ranks actions.
type Engine struct {
	source  collectibleSource
	version versioner

	mu           sync.RWMutex
	buckets      map[uint32]*Bucket
	builtVersion string
	building     bool
}

// NewEngine constructs an Engine. The index is empty until BuildIndex runs.
func NewEngine(src collectibleSource, ver versioner) *Engine {
	return &Engine{source: src, version: ver}
}

// IsReady reports whether the index has been built at least once.
func (e *Engine) IsReady() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.buckets) > 0
}

// ensureIndex kicks an async rebuild if the manifest version changed.
func (e *Engine) ensureIndex() {
	if e.version == nil {
		return
	}
	e.mu.RLock()
	current := e.version.Version()
	needsRebuild := current != "" && e.builtVersion != current && !e.building
	e.mu.RUnlock()
	if needsRebuild {
		go e.BuildIndex()
	}
}

// OnVersionChanged rebuilds the source-bucket index against the new manifest.
// Implements bungie.ManifestObserver.
//
// The build is asynchronous: it reads every collectible row and must not block
// the swap. Rank and MissingForMilestone serve the previous index until it
// lands, and ensureIndex re-kicks the build if this one loses a race.
func (e *Engine) OnVersionChanged(version string) error {
	go e.BuildIndex()
	return nil
}

// BuildIndex (re)builds the source-bucket index. Safe to call concurrently; a
// concurrent call is a no-op while a build is already running. Mirrors search.BuildIndex.
func (e *Engine) BuildIndex() {
	e.mu.Lock()
	if e.building {
		e.mu.Unlock()
		return
	}
	e.building = true
	e.mu.Unlock()
	defer func() {
		e.mu.Lock()
		e.building = false
		e.mu.Unlock()
	}()

	if e.source == nil || e.version == nil {
		return
	}
	version := e.version.Version()
	if version == "" {
		return
	}
	rows, err := e.source.GetAllCollectiblesWithItems()
	if err != nil {
		// A build racing a manifest swap gets ErrNotReady from the provider.
		// That is expected and self-healing — builtVersion is left untouched, so
		// ensureIndex re-kicks on the next Rank/MissingForMilestone call — so it
		// logs at debug. Anything else is a real fault.
		level := slog.LevelError
		if errors.Is(err, manifest.ErrNotReady) {
			level = slog.LevelDebug
		}
		slog.LogAttrs(context.Background(), level, "efficiency index collectible lookup failed",
			slog.String("manifest_version", version),
			observability.Err(err),
		)
		return
	}
	buckets := buildBuckets(rows)

	e.mu.Lock()
	e.buckets = buckets
	e.builtVersion = version
	e.mu.Unlock()
	slog.Info("efficiency index built",
		slog.Int("source_bucket_count", len(buckets)),
		slog.String("manifest_version", version),
	)
}
