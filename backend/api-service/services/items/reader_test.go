package items

import (
	"context"
	"errors"
	"testing"

	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/manifest"
)

// factsRepo is an itemRepo whose acquisition reads are scriptable.
type factsRepo struct {
	items       map[uint32]*bungie.InventoryItemDefinition
	cols        map[uint32][]bungie.CollectibleDefinition
	all         []manifest.CollectibleWithItem
	err         error
	rowCalls    int
	catalogCall int
	onRows      func()
}

func (f *factsRepo) GetWeaponPerks(uint32) ([]manifest.PerkColumn, error) { return nil, nil }
func (f *factsRepo) GetWeaponCatalysts(uint32) ([]manifest.WeaponCatalyst, error) {
	return nil, nil
}

func (f *factsRepo) GetAcquisitionRows(hashes []uint32) (*manifest.AcquisitionRows, error) {
	f.rowCalls++
	if f.onRows != nil {
		f.onRows()
	}
	if f.err != nil {
		return nil, f.err
	}
	items := map[uint32]*bungie.InventoryItemDefinition{}
	cols := map[uint32][]bungie.CollectibleDefinition{}
	for _, h := range hashes {
		if it, ok := f.items[h]; ok {
			items[h] = it
		}
		if c, ok := f.cols[h]; ok {
			cols[h] = c
		}
	}
	return &manifest.AcquisitionRows{Items: items, Collectibles: cols}, nil
}

func (f *factsRepo) GetAllCollectiblesWithItems() ([]manifest.CollectibleWithItem, error) {
	f.catalogCall++
	if f.err != nil {
		return nil, f.err
	}
	return f.all, nil
}

func weapon(hash uint32, name string) *bungie.InventoryItemDefinition {
	d := &bungie.InventoryItemDefinition{Hash: hash, ItemType: bungie.ItemTypeWeapon}
	d.DisplayProperties.Name = name
	d.Inventory.TierType = bungie.TierTypeExotic
	return d
}

func armor(hash uint32, name string, slot uint32) *bungie.InventoryItemDefinition {
	d := &bungie.InventoryItemDefinition{Hash: hash, ItemType: bungie.ItemTypeArmor}
	d.DisplayProperties.Name = name
	d.EquippingBlock.EquipmentSlotTypeHash = slot
	return d
}

func col(hash, itemHash uint32, source string) bungie.CollectibleDefinition {
	c := bungie.CollectibleDefinition{Hash: hash, ItemHash: itemHash, SourceString: source}
	c.DisplayProperties.Name = "c"
	return c
}

func TestLookup_EmptyInputTouchesNoManifest(t *testing.T) {
	repo := &factsRepo{}
	svc := NewService(repo)

	got, err := svc.Lookup(context.Background(), nil)
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Errorf("result = %v, want an allocated empty map", got)
	}
	if repo.rowCalls != 0 {
		t.Errorf("repo calls = %d, want 0", repo.rowCalls)
	}
}

func TestLookup_UnknownHashIsAbsentNotAnError(t *testing.T) {
	repo := &factsRepo{items: map[uint32]*bungie.InventoryItemDefinition{1: weapon(1, "Gjallarhorn")}}
	svc := NewService(repo)

	got, err := svc.Lookup(context.Background(), []uint32{1, 999})
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if _, ok := got[1]; !ok {
		t.Error("known hash missing from result")
	}
	if _, ok := got[999]; ok {
		t.Error("unknown hash present in result, want absent")
	}
}

func TestLookup_ErrorYieldsNoPartialResult(t *testing.T) {
	want := errors.New("manifest unavailable")
	svc := NewService(&factsRepo{err: want})

	got, err := svc.Lookup(context.Background(), []uint32{1})
	if !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
	if got != nil {
		t.Errorf("result = %v, want nil — unavailability must not look like 'no sources'", got)
	}
}

func TestLookup_KnownItemWithNoCollectibleIsComplete(t *testing.T) {
	svc := NewService(&factsRepo{items: map[uint32]*bungie.InventoryItemDefinition{1: weapon(1, "Gjallarhorn")}})

	got, err := svc.Lookup(context.Background(), []uint32{1})
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	f := got[1]
	if f.CollectibleHashes == nil || len(f.CollectibleHashes) != 0 {
		t.Errorf("CollectibleHashes = %v, want allocated empty", f.CollectibleHashes)
	}
	if f.AcquisitionSources == nil || len(f.AcquisitionSources) != 0 {
		t.Errorf("AcquisitionSources = %v, want allocated empty", f.AcquisitionSources)
	}
}

func TestLookup_UnionsSourcesAndOrdersCollectibleHashes(t *testing.T) {
	svc := NewService(&factsRepo{
		items: map[uint32]*bungie.InventoryItemDefinition{1: weapon(1, "Superblack")},
		cols: map[uint32][]bungie.CollectibleDefinition{1: {
			col(90, 1, "Season of the Wish"),
			col(20, 1, "Into the Light"),
			col(50, 1, "Into the Light"), // duplicate text collapses
		}},
	})

	got, _ := svc.Lookup(context.Background(), []uint32{1})
	f := got[1]

	wantHashes := []uint32{20, 50, 90}
	if len(f.CollectibleHashes) != len(wantHashes) {
		t.Fatalf("CollectibleHashes = %v, want %v", f.CollectibleHashes, wantHashes)
	}
	for i, h := range wantHashes {
		if f.CollectibleHashes[i] != h {
			t.Fatalf("CollectibleHashes = %v, want ascending %v", f.CollectibleHashes, wantHashes)
		}
	}
	if len(f.AcquisitionSources) != 2 {
		t.Fatalf("sources = %+v, want 2 after dedup", f.AcquisitionSources)
	}
	if f.AcquisitionSources[0].Text != "Into the Light" {
		t.Errorf("sources not ordered by text: %+v", f.AcquisitionSources)
	}
}

func TestLookup_ArmorIsSlotSpecific(t *testing.T) {
	svc := NewService(&factsRepo{
		items: map[uint32]*bungie.InventoryItemDefinition{1: armor(1, "Helm of Saint-14", bungie.SlotHelmet)},
	})

	got, _ := svc.Lookup(context.Background(), []uint32{1})
	if got[1].ItemType != "Helmet" {
		t.Errorf("ItemType = %q, want %q — canonical facts are slot-specific", got[1].ItemType, "Helmet")
	}
}

func TestLookup_ServesWholeBatchFromCacheOrReloadsIt(t *testing.T) {
	repo := &factsRepo{items: map[uint32]*bungie.InventoryItemDefinition{
		1: weapon(1, "One"), 2: weapon(2, "Two"),
	}}
	svc := NewService(repo)

	if _, err := svc.Lookup(context.Background(), []uint32{1, 2}); err != nil {
		t.Fatalf("first: %v", err)
	}
	if _, err := svc.Lookup(context.Background(), []uint32{1, 2}); err != nil {
		t.Fatalf("second: %v", err)
	}
	if repo.rowCalls != 1 {
		t.Errorf("repo calls = %d, want 1 (full batch hit)", repo.rowCalls)
	}

	// A partial hit reloads the whole batch rather than mixing generations.
	repo.items[3] = weapon(3, "Three")
	if _, err := svc.Lookup(context.Background(), []uint32{1, 2, 3}); err != nil {
		t.Fatalf("third: %v", err)
	}
	if repo.rowCalls != 2 {
		t.Errorf("repo calls = %d, want 2 (partial hit reloads)", repo.rowCalls)
	}
}

func TestLookup_ReturnsDefensiveCopies(t *testing.T) {
	svc := NewService(&factsRepo{
		items: map[uint32]*bungie.InventoryItemDefinition{1: weapon(1, "One")},
		cols:  map[uint32][]bungie.CollectibleDefinition{1: {col(10, 1, "Raid")}},
	})

	first, _ := svc.Lookup(context.Background(), []uint32{1})
	first[1].CollectibleHashes[0] = 999
	first[1].AcquisitionSources[0].Text = "tampered"

	second, _ := svc.Lookup(context.Background(), []uint32{1})
	if second[1].CollectibleHashes[0] != 10 {
		t.Error("caller mutation reached cached collectible hashes")
	}
	if second[1].AcquisitionSources[0].Text != "Raid" {
		t.Error("caller mutation reached cached sources")
	}
}

// ADR 0014: work that began under a retired generation answers its caller but
// installs nothing.
func TestLookup_ResultLoadedUnderRetiredGenerationIsNotCached(t *testing.T) {
	repo := &factsRepo{items: map[uint32]*bungie.InventoryItemDefinition{1: weapon(1, "One")}}
	svc := NewService(repo)
	repo.onRows = func() {
		// A manifest swap lands while this read is in flight.
		if err := svc.OnVersionChanged("v2"); err != nil {
			t.Errorf("OnVersionChanged: %v", err)
		}
	}

	got, err := svc.Lookup(context.Background(), []uint32{1})
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if got[1].Name != "One" {
		t.Error("initiating caller did not get its coherent result")
	}
	if _, cached := svc.facts.get(1); cached {
		t.Error("a result loaded under a retired generation was cached")
	}
}

func TestCatalog_GroupsByItemAndOrdersByHash(t *testing.T) {
	i1, i2 := weapon(30, "Thirty"), weapon(10, "Ten")
	svc := NewService(&factsRepo{all: []manifest.CollectibleWithItem{
		{Collectible: col(1, 30, "Raid"), Item: i1},
		{Collectible: col(2, 10, "Trials"), Item: i2},
		{Collectible: col(3, 30, "Dungeon"), Item: i1}, // second collectible for item 30
	}})

	got, err := svc.Catalog(context.Background())
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("catalog = %d entries, want 2 (grouped by item)", len(got))
	}
	if got[0].ItemHash != 10 || got[1].ItemHash != 30 {
		t.Errorf("order = %d,%d, want ascending 10,30", got[0].ItemHash, got[1].ItemHash)
	}
	if len(got[1].AcquisitionSources) != 2 {
		t.Errorf("item 30 sources = %+v, want both collectibles unioned", got[1].AcquisitionSources)
	}
}

func TestCatalog_SkipsNamelessItemsAndCachesUntilVersionChange(t *testing.T) {
	named := weapon(10, "Ten")
	nameless := weapon(20, "")
	repo := &factsRepo{all: []manifest.CollectibleWithItem{
		{Collectible: col(1, 10, "Raid"), Item: named},
		{Collectible: col(2, 20, "Raid"), Item: nameless},
		{Collectible: col(3, 0, "Raid"), Item: nil},
	}}
	svc := NewService(repo)

	got, _ := svc.Catalog(context.Background())
	if len(got) != 1 || got[0].ItemHash != 10 {
		t.Fatalf("catalog = %+v, want only the named item", got)
	}

	if _, err := svc.Catalog(context.Background()); err != nil {
		t.Fatalf("second Catalog: %v", err)
	}
	if repo.catalogCall != 1 {
		t.Errorf("repo calls = %d, want 1 (published catalog reused)", repo.catalogCall)
	}

	if err := svc.OnVersionChanged("v2"); err != nil {
		t.Fatalf("OnVersionChanged: %v", err)
	}
	if _, err := svc.Catalog(context.Background()); err != nil {
		t.Fatalf("third Catalog: %v", err)
	}
	if repo.catalogCall != 2 {
		t.Errorf("repo calls = %d, want 2 after a version change", repo.catalogCall)
	}
}

func TestCatalog_ReturnsDefensiveCopies(t *testing.T) {
	item := weapon(10, "Ten")
	svc := NewService(&factsRepo{all: []manifest.CollectibleWithItem{
		{Collectible: col(1, 10, "Raid"), Item: item},
	}})

	first, _ := svc.Catalog(context.Background())
	first[0].AcquisitionSources[0].Text = "tampered"
	first[0].CollectibleHashes[0] = 999

	second, _ := svc.Catalog(context.Background())
	if second[0].AcquisitionSources[0].Text != "Raid" {
		t.Error("caller mutation reached the published catalog's sources")
	}
	if second[0].CollectibleHashes[0] != 1 {
		t.Error("caller mutation reached the published catalog's collectible hashes")
	}
}
