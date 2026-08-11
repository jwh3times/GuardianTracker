package adapters

import (
	"context"
	"errors"
	"fmt"
	"reflect"
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

// TestSessionStore_TranslatesUnavailableOnEveryMethod drives every method of
// the session adapter by reflection against the real degraded store.
//
// Reflection rather than a hand-written list because the translation is only
// useful if it is total: a method added later that forgets it would report "the
// database is absent" as a write failure, and the login it belongs to would
// 500 on a deployment that never had a database — which is exactly the defect
// this seam was built to fix.
func TestSessionStore_TranslatesUnavailableOnEveryMethod(t *testing.T) {
	store := NewSessionStore(db.NewStores(nil).Users)
	v := reflect.ValueOf(store)
	typ := v.Type()

	if declared := reflect.TypeFor[auth.SessionStore]().NumMethod(); typ.NumMethod() != declared {
		t.Fatalf("adapter has %d methods, auth.SessionStore declares %d", typ.NumMethod(), declared)
	}

	for method := range typ.Methods() {
		t.Run(method.Name, func(t *testing.T) {
			args := make([]reflect.Value, method.Type.NumIn()-1)
			for j := range args {
				argType := method.Type.In(j + 1)
				if argType == reflect.TypeFor[context.Context]() {
					args[j] = reflect.ValueOf(context.Background())
					continue
				}
				args[j] = reflect.Zero(argType)
			}

			results := method.Func.Call(append([]reflect.Value{v}, args...))
			last := results[len(results)-1]
			err, _ := last.Interface().(error)
			if !errors.Is(err, auth.ErrUnavailable) {
				t.Fatalf("%s returned %v, want auth.ErrUnavailable", method.Name, err)
			}
			if errors.Is(err, db.ErrUnavailable) {
				t.Fatalf("%s leaked the db sentinel across the seam", method.Name)
			}
		})
	}
}

// A real failure must never be reported as an absent database: the session
// write that fails for a real reason has to fail the login.
func TestSessionStore_PassesRealFailuresThrough(t *testing.T) {
	boom := errors.New("connection reset")
	store := NewSessionStore(&failingUserRepo{err: boom})

	err := store.CreateSession(context.Background(), "sid", "member-1", "jti", "ua", time.Now())
	if !errors.Is(err, boom) {
		t.Fatalf("CreateSession error = %v, want the original failure", err)
	}
	if errors.Is(err, auth.ErrUnavailable) {
		t.Fatal("a real failure was reported as an absent database; the login would silently issue a dead session")
	}
}

// failingUserRepo fails every call with one error. It embeds db.UserRepo so the
// methods this test does not exercise panic rather than quietly succeed.
type failingUserRepo struct {
	db.UserRepo
	err error
}

func (f *failingUserRepo) CreateSession(context.Context, string, string, string, string, time.Time) error {
	return f.err
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
