package efficiency

import (
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
		slog.Error("efficiency index collectible lookup failed",
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
