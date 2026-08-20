package items

import (
	"errors"
	"testing"

	"guardian-tracker/api-service/services/manifest"
)

type fakeRepo struct {
	calls int
	cols  []manifest.PerkColumn
	err   error
}

func (f *fakeRepo) GetWeaponPerks(uint32) ([]manifest.PerkColumn, error) {
	f.calls++
	return f.cols, f.err
}

func (f *fakeRepo) GetAcquisitionRows([]uint32) (*manifest.AcquisitionRows, error) {
	return &manifest.AcquisitionRows{}, nil
}
func (f *fakeRepo) GetAllCollectiblesWithItems() ([]manifest.CollectibleWithItem, error) {
	return nil, nil
}

func (f *fakeRepo) GetWeaponCatalysts(uint32) ([]manifest.WeaponCatalyst, error) { return nil, nil }

func TestService_CachesByHash(t *testing.T) {
	repo := &fakeRepo{cols: []manifest.PerkColumn{{Role: "barrel", Label: "Barrel", Perks: []string{"Full Bore"}}}}
	svc := NewService(repo)

	for i := 0; i < 3; i++ {
		got, err := svc.GetWeaponPerks(1000)
		if err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
		if len(got) != 1 || got[0].Label != "Barrel" {
			t.Fatalf("call %d: got %+v", i, got)
		}
	}
	if repo.calls != 1 {
		t.Errorf("repo calls = %d, want 1 (cached)", repo.calls)
	}
}

func TestService_CachesNonWeaponNil(t *testing.T) {
	repo := &fakeRepo{cols: nil}
	svc := NewService(repo)
	for i := 0; i < 2; i++ {
		if _, err := svc.GetWeaponPerks(3000); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	if repo.calls != 1 {
		t.Errorf("nil result not cached: repo calls = %d, want 1", repo.calls)
	}
}

func TestService_DoesNotCacheErrors(t *testing.T) {
	repo := &fakeRepo{err: errors.New("boom")}
	svc := NewService(repo)
	_, _ = svc.GetWeaponPerks(1000)
	_, _ = svc.GetWeaponPerks(1000)
	if repo.calls != 2 {
		t.Errorf("error was cached: repo calls = %d, want 2", repo.calls)
	}
}

func TestService_InvalidateCache(t *testing.T) {
	repo := &fakeRepo{cols: []manifest.PerkColumn{{Label: "Barrel"}}}
	svc := NewService(repo)
	_, _ = svc.GetWeaponPerks(1000)
	svc.InvalidateCache()
	_, _ = svc.GetWeaponPerks(1000)
	if repo.calls != 2 {
		t.Errorf("invalidate did not clear cache: repo calls = %d, want 2", repo.calls)
	}
}

func TestService_BoundsCacheSize(t *testing.T) {
	repo := &fakeRepo{cols: []manifest.PerkColumn{{Label: "Barrel"}}}
	svc := NewService(repo)
	for i := uint32(0); i < maxCacheEntries+50; i++ {
		if _, err := svc.GetWeaponPerks(i); err != nil {
			t.Fatalf("hash %d: %v", i, err)
		}
	}
	if got := svc.perks.size(); got > maxCacheEntries {
		t.Errorf("cache size = %d, want <= %d", got, maxCacheEntries)
	}
}

// fakeCatalystRepo implements itemRepo for GetCatalysts caching tests.
type fakeCatalystRepo struct {
	cats  []manifest.WeaponCatalyst
	err   error
	calls int
}

func (f *fakeCatalystRepo) GetWeaponPerks(uint32) ([]manifest.PerkColumn, error) { return nil, nil }
func (f *fakeCatalystRepo) GetAcquisitionRows([]uint32) (*manifest.AcquisitionRows, error) {
	return &manifest.AcquisitionRows{}, nil
}
func (f *fakeCatalystRepo) GetAllCollectiblesWithItems() ([]manifest.CollectibleWithItem, error) {
	return nil, nil
}
func (f *fakeCatalystRepo) GetWeaponCatalysts(uint32) ([]manifest.WeaponCatalyst, error) {
	f.calls++
	return f.cats, f.err
}

func TestService_GetCatalysts_CachesByHash(t *testing.T) {
	repo := &fakeCatalystRepo{cats: []manifest.WeaponCatalyst{{Name: "Loose Change", Description: "text"}}}
	svc := NewService(repo)

	for i := 0; i < 3; i++ {
		got, err := svc.GetCatalysts(2907129557)
		if err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
		if len(got) != 1 || got[0].Name != "Loose Change" {
			t.Fatalf("call %d: got %+v", i, got)
		}
	}
	if repo.calls != 1 {
		t.Errorf("repo calls = %d, want 1 (cached)", repo.calls)
	}
}

func TestService_GetCatalysts_CachesNonExoticNil(t *testing.T) {
	repo := &fakeCatalystRepo{cats: nil}
	svc := NewService(repo)
	for i := 0; i < 2; i++ {
		if _, err := svc.GetCatalysts(100); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	if repo.calls != 1 {
		t.Errorf("nil result not cached: repo calls = %d, want 1", repo.calls)
	}
}

func TestService_GetCatalysts_DoesNotCacheErrors(t *testing.T) {
	repo := &fakeCatalystRepo{err: errors.New("boom")}
	svc := NewService(repo)
	_, _ = svc.GetCatalysts(2907129557)
	_, _ = svc.GetCatalysts(2907129557)
	if repo.calls != 2 {
		t.Errorf("error was cached: repo calls = %d, want 2", repo.calls)
	}
}

func TestService_GetCatalysts_InvalidateCache(t *testing.T) {
	repo := &fakeCatalystRepo{cats: []manifest.WeaponCatalyst{{Name: "Loose Change"}}}
	svc := NewService(repo)
	_, _ = svc.GetCatalysts(2907129557)
	svc.InvalidateCache()
	_, _ = svc.GetCatalysts(2907129557)
	if repo.calls != 2 {
		t.Errorf("invalidate did not clear catalyst cache: repo calls = %d, want 2", repo.calls)
	}
}
