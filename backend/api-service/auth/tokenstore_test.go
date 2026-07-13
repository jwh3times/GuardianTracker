package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeTokenRepo is an in-memory TokenRepo with the same CAS semantics as
// db.BungieTokenStore.Upsert.
type fakeTokenRepo struct {
	mu        sync.Mutex
	rec       *EncryptedTokenRecord // nil = no row
	updatedAt time.Time
	seq       int64 // strictly-increasing fake clock (real now() is distinct per statement)
	upserts   int
	failAll   bool // simulate DB outage
	noUserRow bool // simulate missing users row (INSERT ... SELECT matches nothing)
}

func (f *fakeTokenRepo) Get(ctx context.Context, membershipID string) (*EncryptedTokenRecord, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failAll {
		return nil, fmt.Errorf("db down")
	}
	if f.rec == nil {
		return nil, ErrTokensNotFound
	}
	r := *f.rec
	r.UpdatedAt = f.updatedAt
	return &r, nil
}

func (f *fakeTokenRepo) Upsert(ctx context.Context, membershipID string, t *EncryptedTokenRecord, prev time.Time) (time.Time, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failAll {
		return time.Time{}, false, fmt.Errorf("db down")
	}
	f.upserts++
	if f.noUserRow {
		return time.Time{}, false, ErrNoUserRow
	}
	// CAS only applies when a row already exists (matches ON CONFLICT ... WHERE).
	if !prev.IsZero() && f.rec != nil && !f.updatedAt.Equal(prev) {
		return time.Time{}, false, nil
	}
	cp := *t
	f.rec = &cp
	f.seq++
	f.updatedAt = time.Unix(1700000000, 0).Add(time.Duration(f.seq) * time.Second)
	return f.updatedAt, true, nil
}

func (f *fakeTokenRepo) Delete(ctx context.Context, membershipID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rec = nil
	return nil
}

func (f *fakeTokenRepo) upsertCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.upserts
}

// newBungieTokenServer fakes Bungie's OAuth token endpoint. Each call mints a
// numbered access/refresh pair so tests can assert which refresh produced a token.
func newBungieTokenServer(t *testing.T, calls *atomic.Int32) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil || r.PostForm.Get("grant_type") != "refresh_token" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		n := calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"access_token":"new-access-%d","refresh_token":"new-refresh-%d","expires_in":3600,"refresh_expires_in":7776000}`, n, n)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newTestStore(t *testing.T, repo TokenRepo, tokenURL string) (*TokenStore, *TokenCipher) {
	t.Helper()
	var cipher *TokenCipher
	if repo != nil {
		var err error
		cipher, err = NewTokenCipher(validKey(t, 7), 1, "", 0)
		if err != nil {
			t.Fatalf("NewTokenCipher: %v", err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	s := NewTokenStore(ctx, "client-id", "client-secret", "", repo, cipher)
	if tokenURL != "" {
		s.tokenURL = tokenURL
	}
	return s, cipher
}

func validTokens(membershipID string) *BungieTokens {
	return &BungieTokens{
		AccessToken:           "access-1",
		RefreshToken:          "refresh-1",
		AccessTokenExpiresAt:  time.Now().Add(1 * time.Hour),
		RefreshTokenExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		MembershipID:          membershipID,
	}
}

func expiredAccessTokens(membershipID string) *BungieTokens {
	t := validTokens(membershipID)
	t.AccessTokenExpiresAt = time.Now().Add(-1 * time.Minute)
	return t
}

func TestTokenStore_StoreAndGet(t *testing.T) {
	repo := &fakeTokenRepo{}
	s, _ := newTestStore(t, repo, "")

	s.Store("user-1", validTokens("user-1"))

	got, err := s.GetValidToken("user-1")
	if err != nil {
		t.Fatalf("GetValidToken: %v", err)
	}
	if got != "access-1" {
		t.Errorf("got token %q, want access-1", got)
	}
	if repo.upsertCount() != 1 {
		t.Errorf("upserts = %d, want 1", repo.upsertCount())
	}
}

func TestTokenStore_MemoryOnlyDegradedMode(t *testing.T) {
	s, _ := newTestStore(t, nil, "")
	s.Store("user-1", validTokens("user-1"))
	got, err := s.GetValidToken("user-1")
	if err != nil || got != "access-1" {
		t.Fatalf("GetValidToken = %q, %v; want access-1, nil", got, err)
	}
}

func TestTokenStore_DBLoadOnMemoryMiss(t *testing.T) {
	repo := &fakeTokenRepo{}
	s1, cipher := newTestStore(t, repo, "")
	s1.Store("user-1", validTokens("user-1"))

	// New store sharing the repo + cipher simulates a restart: memory is empty,
	// tokens must be loaded and decrypted from the DB.
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	s2 := NewTokenStore(ctx, "client-id", "client-secret", "", repo, cipher)

	got, err := s2.GetValidToken("user-1")
	if err != nil {
		t.Fatalf("GetValidToken after restart: %v", err)
	}
	if got != "access-1" {
		t.Errorf("got token %q, want access-1", got)
	}
}

// TestTokenStore_RefreshPersistsNewTokens is the B1 regression test: a token
// refresh must write the new tokens through to the DB so they survive a restart.
func TestTokenStore_RefreshPersistsNewTokens(t *testing.T) {
	var calls atomic.Int32
	srv := newBungieTokenServer(t, &calls)
	repo := &fakeTokenRepo{}
	s, cipher := newTestStore(t, repo, srv.URL)

	s.Store("user-1", expiredAccessTokens("user-1"))

	got, err := s.GetValidToken("user-1")
	if err != nil {
		t.Fatalf("GetValidToken: %v", err)
	}
	if got != "new-access-1" {
		t.Errorf("got token %q, want new-access-1", got)
	}
	if repo.upsertCount() != 2 {
		t.Errorf("upserts = %d, want 2 (initial store + refresh write)", repo.upsertCount())
	}

	// The DB row must now decrypt to the *new* tokens (this is what B1 broke).
	rec, err := repo.Get(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("repo.Get: %v", err)
	}
	plain, err := cipher.Decrypt(rec.AccessTokenEnc, []byte("user-1"), rec.KeyVersion)
	if err != nil {
		t.Fatalf("decrypt persisted access token: %v", err)
	}
	if string(plain) != "new-access-1" {
		t.Errorf("persisted access token = %q, want new-access-1", plain)
	}
	refPlain, err := cipher.Decrypt(rec.RefreshTokenEnc, []byte("user-1"), rec.KeyVersion)
	if err != nil {
		t.Fatalf("decrypt persisted refresh token: %v", err)
	}
	if string(refPlain) != "new-refresh-1" {
		t.Errorf("persisted refresh token = %q, want new-refresh-1", refPlain)
	}
}

// TestTokenStore_RefreshRaceAdoptsWinner: when the CAS write loses (another
// replica refreshed first), the store must adopt the winner's tokens from the
// DB instead of overwriting them.
func TestTokenStore_RefreshRaceAdoptsWinner(t *testing.T) {
	var calls atomic.Int32
	srv := newBungieTokenServer(t, &calls)
	repo := &fakeTokenRepo{}
	s, cipher := newTestStore(t, repo, srv.URL)

	s.Store("user-1", expiredAccessTokens("user-1"))

	// Simulate another replica winning the refresh race: write a newer, valid
	// row directly to the repo so our in-memory dbUpdatedAt baseline is stale.
	aad := []byte("user-1")
	winAcc, kv, _ := cipher.Encrypt([]byte("winner-access"), aad)
	winRef, _, _ := cipher.Encrypt([]byte("winner-refresh"), aad)
	if _, ok, err := repo.Upsert(context.Background(), "user-1", &EncryptedTokenRecord{
		AccessTokenEnc:   winAcc,
		RefreshTokenEnc:  winRef,
		AccessExpiresAt:  time.Now().Add(1 * time.Hour),
		RefreshExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		KeyVersion:       kv,
	}, time.Time{}); err != nil || !ok {
		t.Fatalf("seed winner row: ok=%v err=%v", ok, err)
	}

	got, err := s.GetValidToken("user-1")
	if err != nil {
		t.Fatalf("GetValidToken: %v", err)
	}
	if got != "winner-access" {
		t.Errorf("got token %q, want winner-access (adopted from DB)", got)
	}
	// Subsequent calls serve the adopted token from memory without refreshing again.
	if calls.Load() != 1 {
		t.Errorf("bungie refresh calls = %d, want 1", calls.Load())
	}
	got2, err := s.GetValidToken("user-1")
	if err != nil || got2 != "winner-access" {
		t.Errorf("second GetValidToken = %q, %v; want winner-access, nil", got2, err)
	}
}

func TestTokenStore_ExpiredRefreshTokenRequiresReauth(t *testing.T) {
	repo := &fakeTokenRepo{}
	s, _ := newTestStore(t, repo, "")

	tok := validTokens("user-1")
	tok.AccessTokenExpiresAt = time.Now().Add(-1 * time.Hour)
	tok.RefreshTokenExpiresAt = time.Now().Add(-1 * time.Minute)
	s.Store("user-1", tok)

	_, err := s.GetValidToken("user-1")
	if err == nil || !strings.Contains(err.Error(), "re-authentication required") {
		t.Fatalf("err = %v, want re-authentication required", err)
	}
}

func TestTokenStore_UnknownUser(t *testing.T) {
	s, _ := newTestStore(t, &fakeTokenRepo{}, "")
	if _, err := s.GetValidToken("nobody"); err == nil {
		t.Fatal("expected error for unknown user")
	}
}

func TestTokenStore_ConcurrentRefreshSingleFlight(t *testing.T) {
	var calls atomic.Int32
	srv := newBungieTokenServer(t, &calls)
	repo := &fakeTokenRepo{}
	s, _ := newTestStore(t, repo, srv.URL)

	s.Store("user-1", expiredAccessTokens("user-1"))

	const n = 10
	var wg sync.WaitGroup
	results := make([]string, n)
	errs := make([]error, n)
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = s.GetValidToken("user-1")
		}(i)
	}
	wg.Wait()

	if calls.Load() != 1 {
		t.Errorf("bungie refresh calls = %d, want 1 (per-user mutex must single-flight)", calls.Load())
	}
	for i := range n {
		if errs[i] != nil {
			t.Fatalf("goroutine %d: %v", i, errs[i])
		}
		if results[i] != "new-access-1" {
			t.Errorf("goroutine %d got %q, want new-access-1", i, results[i])
		}
	}
}

func TestTokenStore_DeleteClearsMemoryAndDB(t *testing.T) {
	repo := &fakeTokenRepo{}
	s, _ := newTestStore(t, repo, "")
	s.Store("user-1", validTokens("user-1"))

	s.Delete("user-1")

	if _, err := s.GetValidToken("user-1"); err == nil {
		t.Fatal("expected error after delete")
	}
	if repo.rec != nil {
		t.Error("DB row not deleted")
	}
}

func TestTokenStore_StoreSurvivesDBOutage(t *testing.T) {
	repo := &fakeTokenRepo{failAll: true}
	s, _ := newTestStore(t, repo, "")
	s.Store("user-1", validTokens("user-1"))
	got, err := s.GetValidToken("user-1")
	if err != nil || got != "access-1" {
		t.Fatalf("GetValidToken = %q, %v; want access-1 from memory despite DB outage", got, err)
	}
}
