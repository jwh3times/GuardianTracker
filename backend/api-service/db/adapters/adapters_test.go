package adapters

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"guardian-tracker/api-service/auth"
	"guardian-tracker/api-service/db"
)

type fakeTokenStore struct {
	rec      *db.EncryptedTokens
	getErr   error
	upsertAt time.Time
	upsertOK bool
	upserErr error
	deleted  string
}

func (f *fakeTokenStore) Get(context.Context, string) (*db.EncryptedTokens, error) {
	return f.rec, f.getErr
}

func (f *fakeTokenStore) Upsert(context.Context, string, *db.EncryptedTokens, time.Time) (time.Time, bool, error) {
	return f.upsertAt, f.upsertOK, f.upserErr
}

func (f *fakeTokenStore) Delete(_ context.Context, membershipID string) error {
	f.deleted = membershipID
	return nil
}

// TestTokenRepo_GetTranslatesOnlyTheNotFoundSentinel is the point of this
// adapter. auth.TokenStore's CAS reconciliation branches on "definitively
// absent" versus "the read failed", so translating a transient error into
// ErrTokensNotFound would let it overwrite a row it never actually read.
func TestTokenRepo_GetTranslatesOnlyTheNotFoundSentinel(t *testing.T) {
	transient := errors.New("connection reset")

	tests := []struct {
		name    string
		getErr  error
		want    error
		wantIs  bool
		exactly error
	}{
		{
			name:   "db sentinel becomes the auth sentinel",
			getErr: db.ErrTokensNotFound,
			want:   auth.ErrTokensNotFound,
			wantIs: true,
		},
		{
			name:   "wrapped db sentinel is still recognised",
			getErr: fmt.Errorf("query users: %w", db.ErrTokensNotFound),
			want:   auth.ErrTokensNotFound,
			wantIs: true,
		},
		{
			name:    "any other error passes through untouched",
			getErr:  transient,
			exactly: transient,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := NewTokenRepo(&fakeTokenStore{getErr: tc.getErr})
			_, err := repo.Get(context.Background(), "member-1")

			if tc.wantIs {
				if !errors.Is(err, tc.want) {
					t.Fatalf("Get error = %v, want errors.Is(%v)", err, tc.want)
				}
				return
			}
			if !errors.Is(err, tc.exactly) {
				t.Fatalf("Get error = %v, want %v", err, tc.exactly)
			}
			if errors.Is(err, auth.ErrTokensNotFound) {
				t.Fatal("a transient error was reported as ErrTokensNotFound; the store may now clobber a row it never read")
			}
		})
	}
}

func TestTokenRepo_GetMapsEveryField(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	repo := NewTokenRepo(&fakeTokenStore{rec: &db.EncryptedTokens{
		AccessTokenEnc:   []byte("access"),
		RefreshTokenEnc:  []byte("refresh"),
		AccessExpiresAt:  now.Add(time.Hour),
		RefreshExpiresAt: now.Add(24 * time.Hour),
		KeyVersion:       2,
		UpdatedAt:        now,
	}})

	got, err := repo.Get(context.Background(), "member-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got.AccessTokenEnc) != "access" || string(got.RefreshTokenEnc) != "refresh" {
		t.Errorf("ciphertext not carried across: %+v", got)
	}
	if got.KeyVersion != 2 {
		t.Errorf("KeyVersion = %d, want 2 — the wrong key version cannot decrypt", got.KeyVersion)
	}
	if !got.AccessExpiresAt.Equal(now.Add(time.Hour)) || !got.RefreshExpiresAt.Equal(now.Add(24*time.Hour)) {
		t.Errorf("expiries not carried across: %+v", got)
	}
	if !got.UpdatedAt.Equal(now) {
		t.Errorf("UpdatedAt = %v, want %v — it is the CAS comparand", got.UpdatedAt, now)
	}
}

func TestTokenRepo_UpsertTranslatesNoUserRow(t *testing.T) {
	repo := NewTokenRepo(&fakeTokenStore{upserErr: db.ErrNoUserRow})
	_, _, err := repo.Upsert(context.Background(), "member-1", &auth.EncryptedTokenRecord{}, time.Time{})
	if !errors.Is(err, auth.ErrNoUserRow) {
		t.Fatalf("Upsert error = %v, want auth.ErrNoUserRow", err)
	}

	transient := errors.New("deadlock detected")
	repo = NewTokenRepo(&fakeTokenStore{upserErr: transient})
	if _, _, err := repo.Upsert(context.Background(), "member-1", &auth.EncryptedTokenRecord{}, time.Time{}); !errors.Is(err, transient) {
		t.Fatalf("Upsert error = %v, want the original error", err)
	}
}

func TestTokenRepo_DeletePassesThrough(t *testing.T) {
	store := &fakeTokenStore{}
	if err := NewTokenRepo(store).Delete(context.Background(), "member-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if store.deleted != "member-1" {
		t.Fatalf("deleted = %q, want member-1", store.deleted)
	}
}

func TestTokenRepo_UpsertCarriesTheCASResult(t *testing.T) {
	at := time.Now().UTC()
	repo := NewTokenRepo(&fakeTokenStore{upsertAt: at, upsertOK: true})

	gotAt, ok, err := repo.Upsert(context.Background(), "member-1", &auth.EncryptedTokenRecord{KeyVersion: 3}, at.Add(-time.Minute))
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if !ok || !gotAt.Equal(at) {
		t.Fatalf("Upsert = (%v, %v), want (%v, true)", gotAt, ok, at)
	}
}

type fakeWishlistStore struct {
	userID int64
	items  []db.WishlistItem
	err    error
}

func (f *fakeWishlistStore) GetUserID(context.Context, string) (int64, error) {
	return f.userID, f.err
}

func (f *fakeWishlistStore) List(context.Context, int64) ([]db.WishlistItem, error) {
	return f.items, f.err
}

func TestWeeklyWishlist_ProjectsItemHashes(t *testing.T) {
	repo := NewWeeklyWishlist(&fakeWishlistStore{
		userID: 42,
		items: []db.WishlistItem{
			{ID: 1, ItemHash: 100, Priority: 1, Notes: "Gjallarhorn"},
			{ID: 2, ItemHash: 200, Priority: 2, Notes: "Thorn"},
		},
	})

	id, err := repo.GetUserID(context.Background(), "member-1")
	if err != nil || id != 42 {
		t.Fatalf("GetUserID = (%d, %v), want (42, nil)", id, err)
	}

	items, err := repo.List(context.Background(), 42)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 2 || items[0].ItemHash != 100 || items[1].ItemHash != 200 {
		t.Fatalf("List = %+v, want the two item hashes in order", items)
	}
}

// A degraded store reports ErrUnavailable; the projection must not turn that
// into an empty wish list, which weekly would read as "wants nothing".
func TestWeeklyWishlist_ListPropagatesErrors(t *testing.T) {
	repo := NewWeeklyWishlist(&fakeWishlistStore{err: db.ErrUnavailable})
	items, err := repo.List(context.Background(), 1)
	if !errors.Is(err, db.ErrUnavailable) {
		t.Fatalf("List error = %v, want db.ErrUnavailable", err)
	}
	if items != nil {
		t.Fatalf("List items = %+v, want nil alongside an error", items)
	}
}
