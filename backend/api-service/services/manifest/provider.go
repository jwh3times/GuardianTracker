package manifest

import (
	"errors"
	"fmt"
	"sync"

	"guardian-tracker/api-service/services/bungie"
)

// ErrNotReady marks any failure caused by the manifest database not being
// usable yet: still downloading, or temporarily closed during the hourly swap.
// Callers map it to 503 MANIFEST_NOT_READY; query errors against an open
// database (e.g. a corrupt file) do NOT wrap it and surface as real errors.
var ErrNotReady = errors.New("manifest not ready")

// Provider is the single manifest-repository lifecycle strategy for all consumers:
// it opens the SQLite database lazily (retrying until the file exists, so services
// work after a cold-start download without a restart — B4) and participates in the
// hourly manifest swap via CloseForSwap/Reopen (so the rename succeeds on Windows
// and repositories never serve a stale deleted inode on Linux — B5).
//
// Provider delegates every repository method consumers use, so a *Provider
// structurally satisfies each consumer-side manifest interface.
type Provider struct {
	path string
	mu   sync.RWMutex
	repo *Repository
	// swapping is true between CloseForSwap and Reopen. While set, get() returns
	// ErrNotReady instead of lazily reopening — reopening mid-swap would hold a
	// handle on the old file and make the rename fail on Windows (B5).
	swapping bool
}

func NewProvider(path string) *Provider {
	return &Provider{path: path}
}

// get returns the open repository, lazily opening it on first use.
// Returns ErrNotReady while the manifest file does not exist yet or is being swapped.
func (p *Provider) get() (*Repository, error) {
	p.mu.RLock()
	r, swapping := p.repo, p.swapping
	p.mu.RUnlock()
	if r != nil {
		return r, nil
	}
	if swapping {
		return nil, ErrNotReady
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.repo != nil {
		return p.repo, nil
	}
	if p.swapping {
		return nil, ErrNotReady
	}
	repo, err := NewRepository(p.path)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotReady, err)
	}
	p.repo = repo
	return p.repo, nil
}

// CloseForSwap closes the underlying SQLite handle ahead of the manifest file
// being replaced and marks the provider as swapping. Queries racing the swap get
// a clean ErrNotReady (mapped to 503) rather than a "database is closed" error,
// and get() will not reopen until Reopen clears the flag — so no handle is held
// on the old file and the rename succeeds on Windows.
func (p *Provider) CloseForSwap() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.repo != nil {
		_ = p.repo.Close()
		p.repo = nil
	}
	p.swapping = true
}

// Reopen re-establishes the SQLite connection after a manifest swap (or opens it
// for the first time after the initial download) and clears the swapping flag.
func (p *Provider) Reopen() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.swapping = false
	if p.repo != nil {
		// CloseForSwap nils repo, so this is only reached if a swap was not in
		// progress; reconnect to pick up any on-disk change.
		return p.repo.Reconnect()
	}
	repo, err := NewRepository(p.path)
	if err != nil {
		return err
	}
	p.repo = repo
	return nil
}

// Close releases the repository at shutdown.
func (p *Provider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.repo != nil {
		return p.repo.Close()
	}
	return nil
}

// --- delegated repository methods ---

func (p *Provider) GetItemsByHashes(hashes []uint32) (map[uint32]*bungie.InventoryItemDefinition, error) {
	r, err := p.get()
	if err != nil {
		return nil, err
	}
	return r.GetItemsByHashes(hashes)
}

func (p *Provider) GetMilestoneDefinitions(hashes []uint32) (map[uint32]*bungie.MilestoneDefinition, error) {
	r, err := p.get()
	if err != nil {
		return nil, err
	}
	return r.GetMilestoneDefinitions(hashes)
}

func (p *Provider) GetActivityDefinitions(hashes []uint32) (map[uint32]*bungie.ActivityDefinition, error) {
	r, err := p.get()
	if err != nil {
		return nil, err
	}
	return r.GetActivityDefinitions(hashes)
}

func (p *Provider) GetActivityModifierDefinitions(hashes []uint32) (map[uint32]*bungie.ActivityModifierDefinition, error) {
	r, err := p.get()
	if err != nil {
		return nil, err
	}
	return r.GetActivityModifierDefinitions(hashes)
}

func (p *Provider) GetPresentationNodeDefinitions(hashes []uint32) (map[uint32]*PresentationNodeDef, error) {
	r, err := p.get()
	if err != nil {
		return nil, err
	}
	return r.GetPresentationNodeDefinitions(hashes)
}

func (p *Provider) GetRecordDefinitions(hashes []uint32) (map[uint32]*RecordDef, error) {
	r, err := p.get()
	if err != nil {
		return nil, err
	}
	return r.GetRecordDefinitions(hashes)
}

func (p *Provider) GetAllCollectiblesWithItems() ([]CollectibleWithItem, error) {
	r, err := p.get()
	if err != nil {
		return nil, err
	}
	return r.GetAllCollectiblesWithItems()
}

func (p *Provider) GetAllPresentationNodes() (map[uint32]*PresentationNodeDef, error) {
	r, err := p.get()
	if err != nil {
		return nil, err
	}
	return r.GetAllPresentationNodes()
}

func (p *Provider) GetCollectiblesByItemHashes(hashes []uint32) (map[uint32]*bungie.CollectibleDefinition, error) {
	r, err := p.get()
	if err != nil {
		return nil, err
	}
	return r.GetCollectiblesByItemHashes(hashes)
}

func (p *Provider) GetWeaponTypesByName() (map[string]string, error) {
	r, err := p.get()
	if err != nil {
		return nil, err
	}
	return r.GetWeaponTypesByName()
}

func (p *Provider) GetExoticWeaponsByName() (map[string]ExoticWeapon, error) {
	r, err := p.get()
	if err != nil {
		return nil, err
	}
	return r.GetExoticWeaponsByName()
}
