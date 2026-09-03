package efficiency

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"guardian-tracker/api-service/observability"
	"guardian-tracker/api-service/services/manifest"
	"guardian-tracker/api-service/services/manifeststate"
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

// Engine holds the per-manifest-version source-bucket index and ranks candidate facts.
type Engine struct {
	source  collectibleSource
	version versioner

	mu           sync.RWMutex
	buckets      map[uint32]*Bucket
	builtVersion string
	ready        bool
	publication  *manifeststate.Publication
	building     map[manifeststate.Attempt]struct{}
}

// NewEngine constructs an Engine. The index is empty until BuildIndex runs.
func NewEngine(src collectibleSource, ver versioner) *Engine {
	return &Engine{
		source:      src,
		version:     ver,
		publication: manifeststate.New(nil),
		building:    make(map[manifeststate.Attempt]struct{}),
	}
}

// IsReady reports whether the index has been built at least once.
func (e *Engine) IsReady() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.ready
}

// ensureIndex kicks an async rebuild if the manifest version changed.
func (e *Engine) ensureIndex() {
	if e.version == nil {
		return
	}
	current := e.version.Version()
	e.mu.RLock()
	needsRebuild := current != "" && e.builtVersion != current
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
	if err := e.publication.Advance(version); err != nil {
		return err
	}
	go e.BuildIndex()
	return nil
}

// BuildIndex (re)builds the source-bucket index. Safe to call concurrently; a
// concurrent call in the same generation is coalesced. A newer generation may
// build while obsolete work finishes; its distinct Attempt keeps the old
// cleanup from clearing the new generation's state.
func (e *Engine) BuildIndex() {
	if e.source == nil || e.version == nil || e.publication == nil {
		return
	}
	attempt := e.publication.Begin()
	version := e.version.Version()
	if version == "" {
		return
	}

	e.mu.Lock()
	if e.building == nil {
		e.building = make(map[manifeststate.Attempt]struct{})
	}
	if _, building := e.building[attempt]; building {
		e.mu.Unlock()
		return
	}
	e.building[attempt] = struct{}{}
	e.mu.Unlock()
	defer func() {
		e.mu.Lock()
		delete(e.building, attempt)
		e.mu.Unlock()
	}()
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

	if !attempt.Publish(func() {
		e.mu.Lock()
		e.buckets = buckets
		e.builtVersion = version
		e.ready = true
		e.mu.Unlock()
	}) {
		return
	}
	slog.Info("efficiency index built",
		slog.Int("source_bucket_count", len(buckets)),
		slog.String("manifest_version", version),
	)
}
