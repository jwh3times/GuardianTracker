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

func (f *fakeRepo) GetItemView(uint32) (*manifest.ItemView, error) { return nil, nil }

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
	if len(svc.cache) > maxCacheEntries {
		t.Errorf("cache size = %d, want <= %d", len(svc.cache), maxCacheEntries)
	}
}

type fakeItemRepo struct {
	view  *manifest.ItemView
	calls int
}

func (f *fakeItemRepo) GetWeaponPerks(uint32) ([]manifest.PerkColumn, error) { return nil, nil }
func (f *fakeItemRepo) GetItemView(uint32) (*manifest.ItemView, error) {
	f.calls++
	return f.view, nil
}

func TestService_GetItem_CachesAndInvalidates(t *testing.T) {
	repo := &fakeItemRepo{view: &manifest.ItemView{ItemHash: "100", Name: "Fatebringer"}}
	svc := NewService(repo)

	if v, _ := svc.GetItem(100); v == nil || v.Name != "Fatebringer" {
		t.Fatalf("GetItem = %+v", v)
	}
	if _, _ = svc.GetItem(100); repo.calls != 1 {
		t.Errorf("repo called %d times, want 1 (cached)", repo.calls)
	}
	svc.InvalidateCache()
	if _, _ = svc.GetItem(100); repo.calls != 2 {
		t.Errorf("after invalidate, repo called %d times, want 2", repo.calls)
	}
}
