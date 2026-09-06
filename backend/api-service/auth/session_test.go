package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"guardian-tracker/api-service/cache"
)

// These tests reach the composition that used to be unreachable: the ordering
// of exchange → profile → upsert → token store → mint → session write, and the
// rules that only exist in how those calls are wired together. Bungie is a
// httptest fixture; everything else is real.

const (
	testSecret     = "session-test-secret-at-least-32-chars!!"
	testMembership = "4611686018467260757"
)

// stubSessionStore records what the issuer asked of the store and returns what
// the test told it to.
type stubSessionStore struct {
	upsertID      int64
	upsertVersion int
	upsertRole    int16
	upsertErr     error
	upsertAdmin   bool // captured forceAdmin argument
	upsertCalls   int

	created     map[string]string // sid -> jti
	createdUA   string
	createErr   error
	createCalls int

	rotated      bool
	reused       bool
	rotateErr    error
	rotateCalls  int
	rotatedOldID string

	deleted     []string
	deleteErr   error
	deletedAll  []string
	bumped      []string
	bumpErr     error
	deleteAllEr error

	version int
	role    int16
	found   bool
}

func newStubStore() *stubSessionStore {
	return &stubSessionStore{
		upsertID: 42, upsertVersion: 7, upsertRole: RoleBeta,
		created: map[string]string{},
		rotated: true,
		version: 7, found: true,
	}
}

func (s *stubSessionStore) Upsert(_ context.Context, _ string, _ int16, _ string, forceAdmin bool) (int64, int, int16, error) {
	s.upsertCalls++
	s.upsertAdmin = forceAdmin
	if s.upsertErr != nil {
		return 0, 0, 0, s.upsertErr
	}
	return s.upsertID, s.upsertVersion, s.upsertRole, nil
}

func (s *stubSessionStore) CreateSession(_ context.Context, id, _, jti, userAgent string, _ time.Time) error {
	s.createCalls++
	if s.createErr != nil {
		return s.createErr
	}
	s.created[id] = jti
	s.createdUA = userAgent
	return nil
}

func (s *stubSessionStore) RotateSession(_ context.Context, _, _, oldJTI, _ string, _ time.Time) (bool, bool, error) {
	s.rotateCalls++
	s.rotatedOldID = oldJTI
	return s.rotated, s.reused, s.rotateErr
}

func (s *stubSessionStore) DeleteSession(_ context.Context, id string) error {
	s.deleted = append(s.deleted, id)
	return s.deleteErr
}

func (s *stubSessionStore) DeleteUserSessions(_ context.Context, membershipID string) error {
	s.deletedAll = append(s.deletedAll, membershipID)
	return s.deleteAllEr
}

func (s *stubSessionStore) BumpTokenVersion(_ context.Context, membershipID string) error {
	s.bumped = append(s.bumped, membershipID)
	return s.bumpErr
}

func (s *stubSessionStore) GetAuthInfo(_ context.Context, _ string) (int, int16, bool, error) {
	return s.version, s.role, s.found, nil
}

func (s *stubSessionStore) SessionExists(_ context.Context, _ string) (bool, error) {
	return true, nil
}

// fakeBungie stands in for Bungie's OAuth token and membership endpoints.
type fakeBungie struct {
	*httptest.Server
	tokenBody   string
	profileBody string
	tokenStatus int
	tokenHits   int
	lastGrant   string
}

func newFakeBungie(t *testing.T) *fakeBungie {
	t.Helper()
	f := &fakeBungie{
		tokenBody:   `{"access_token":"bungie-access","expires_in":3600}`,
		profileBody: `{"Response":{"destinyMemberships":[{"membershipType":3,"membershipId":"` + testMembership + `","displayName":"TestGuardian"}],"primaryMembershipId":"` + testMembership + `"},"ErrorCode":1}`,
	}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "GetMembershipsForCurrentUser") {
			fmt.Fprint(w, f.profileBody)
			return
		}
		f.tokenHits++
		body := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(body)
		f.lastGrant = string(body)
		if f.tokenStatus != 0 {
			w.WriteHeader(f.tokenStatus)
		}
		fmt.Fprint(w, f.tokenBody)
	}))
	t.Cleanup(f.Close)
	return f
}

func newIssuer(t *testing.T, store *stubSessionStore, bungie *fakeBungie, admins ...string) *SessionIssuer {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	tokenURL, apiBase := "", ""
	if bungie != nil {
		tokenURL, apiBase = bungie.URL, bungie.URL
	}
	return NewSessionIssuer(SessionDeps{
		JWT:     NewJWTWithTTL(testSecret, time.Hour, 30),
		Tokens:  NewTokenStore(ctx, "cid", tokenURL, nil, nil),
		Users:   store,
		Revoker: NewRevocationChecker(store, cache.NewNoOpCache()),
		Cache:   cache.NewNoOpCache(),
		State:   NewStateSigner(testSecret),
		OAuth: OAuthConfig{
			ClientID:        "cid",
			TokenURL:        tokenURL,
			APIBaseURL:      apiBase,
			AuthorizeURL:    "https://www.bungie.net/en/OAuth/Authorize",
			RedirectURI:     "http://localhost:5273/auth/callback",
			BootstrapAdmins: admins,
		},
	})
}

// login runs the real state round-trip: the state the issuer hands out through
// AuthorizeURL is the one the callback presents.
func login(t *testing.T, s *SessionIssuer, code string) (*Session, error) {
	t.Helper()
	_, state, nonce, err := s.AuthorizeURL()
	if err != nil {
		t.Fatalf("AuthorizeURL: %v", err)
	}
	return s.Login(context.Background(), code, state, nonce, "test-agent")
}

func reasonOf(t *testing.T, err error) Reason {
	t.Helper()
	se, ok := AsSessionError(err)
	if !ok {
		t.Fatalf("error %v is not a SessionError", err)
	}
	return se.Reason
}

// TestLogin_Success walks the whole composition and asserts what each step
// contributed to the result: the token version and role from the upsert, the
// session row bound to the refresh token's jti, and the Bungie tokens stored
// under the membership the profile resolved to.
func TestLogin_Success(t *testing.T) {
	store := newStubStore()
	bungie := newFakeBungie(t)
	issuer := newIssuer(t, store, bungie)

	session, err := login(t, issuer, "auth-code")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	if session.Membership.MembershipID != testMembership || session.Membership.DisplayName != "TestGuardian" {
		t.Errorf("membership = %+v", session.Membership)
	}
	if session.Role != RoleBeta {
		t.Errorf("role = %d, want the role the upsert returned (%d)", session.Role, RoleBeta)
	}
	if session.AppUserID == nil || *session.AppUserID != 42 {
		t.Errorf("AppUserID = %v, want the users-row id for the audit trail", session.AppUserID)
	}

	grant, err := url.ParseQuery(bungie.lastGrant)
	if err != nil {
		t.Fatalf("parse authorization-code grant: %v", err)
	}
	if got := grant.Get("client_id"); got != "cid" {
		t.Errorf("authorization-code client_id = %q, want cid", got)
	}
	if _, present := grant["client_secret"]; present {
		t.Error("authorization-code grant unexpectedly included client_secret")
	}

	// Both JWTs carry the session id, and the access token carries the token
	// version the upsert reported — not the fallback.
	claims, err := issuer.jwt.ValidateToken(session.AccessToken)
	if err != nil {
		t.Fatalf("access token invalid: %v", err)
	}
	if claims.SessionID != session.SessionID {
		t.Errorf("access token sid = %q, want %q", claims.SessionID, session.SessionID)
	}
	if claims.TokenVersion != 7 {
		t.Errorf("token version = %d, want 7 from the upsert", claims.TokenVersion)
	}

	// The session row is what the refresh token rotates against, so it must
	// hold that token's jti, not some other one.
	refreshClaims, err := issuer.jwt.ValidateToken(session.RefreshToken)
	if err != nil {
		t.Fatalf("refresh token invalid: %v", err)
	}
	if got := store.created[session.SessionID]; got != refreshClaims.ID {
		t.Errorf("session row jti = %q, want the issued refresh token's jti %q", got, refreshClaims.ID)
	}
	if store.createdUA != "test-agent" {
		t.Errorf("session user agent = %q, want the request's", store.createdUA)
	}

	// The Bungie tokens the exchange returned are stored under the resolved
	// membership, which is what every later Bungie call depends on.
	if tok, err := issuer.tokens.GetValidToken(testMembership); err != nil || tok != "bungie-access" {
		t.Errorf("stored Bungie token = %q, %v; want bungie-access", tok, err)
	}
	if tokens := issuer.tokens.tokens[testMembership]; tokens.RefreshToken != "" || !tokens.RefreshTokenExpiresAt.IsZero() {
		t.Errorf("public-client token unexpectedly has refresh capability: %+v", tokens)
	}
}

// TestLogin_UpsertFailureFallsBack pins the documented best-effort rule: a login
// whose user row cannot be written still succeeds, with token version 1 and the
// standard tier, and no actor id for the audit trail.
func TestLogin_UpsertFailureFallsBack(t *testing.T) {
	store := newStubStore()
	store.upsertErr = errors.New("database on fire")
	issuer := newIssuer(t, store, newFakeBungie(t))

	session, err := login(t, issuer, "auth-code")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if session.Role != RoleStandard {
		t.Errorf("role = %d, want standard when the row could not be read", session.Role)
	}
	if session.AppUserID != nil {
		t.Errorf("AppUserID = %v, want nil when the upsert failed", *session.AppUserID)
	}
	claims, err := issuer.jwt.ValidateToken(session.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	if claims.TokenVersion != 1 {
		t.Errorf("token version = %d, want the fallback 1", claims.TokenVersion)
	}
}

// TestLogin_SessionWriteFailureFailsLogin is the load-bearing one: the access
// token is rejected on its next request when no live session row exists, so a
// failed write must fail the login rather than hand back a dead session.
func TestLogin_SessionWriteFailureFailsLogin(t *testing.T) {
	store := newStubStore()
	store.createErr = errors.New("write failed")
	issuer := newIssuer(t, store, newFakeBungie(t))

	session, err := login(t, issuer, "auth-code")
	if session != nil {
		t.Fatalf("a session was returned despite the failed write: %+v", session)
	}
	if got := reasonOf(t, err); got != ReasonSessionWrite {
		t.Errorf("reason = %q, want %q", got, ReasonSessionWrite)
	}
}

// TestLogin_WithoutPersistenceIssuesSessionlessLogin covers the deployment with
// no database at all (ADR 0004's Minikube path). There is no session row to
// write and nothing that reads one, so login must succeed — it returned 500
// before this seam existed, because "no database" and "the write failed" were
// the same error.
func TestLogin_WithoutPersistenceIssuesSessionlessLogin(t *testing.T) {
	store := newStubStore()
	store.upsertErr = ErrUnavailable
	store.createErr = ErrUnavailable
	issuer := newIssuer(t, store, newFakeBungie(t))

	session, err := login(t, issuer, "auth-code")
	if err != nil {
		t.Fatalf("degraded login failed: %v", err)
	}
	if session.AccessToken == "" || session.RefreshToken == "" {
		t.Error("degraded login returned no usable tokens")
	}
	if store.createCalls != 1 {
		t.Errorf("CreateSession called %d times, want 1 — the attempt is what distinguishes the two cases", store.createCalls)
	}
}

// TestLogin_WrappedStoreFailureStillFails guards the above: only the sentinel
// is tolerated. A real failure that merely happens alongside one must not be
// mistaken for an absent database.
func TestLogin_WrappedStoreFailureStillFails(t *testing.T) {
	store := newStubStore()
	store.createErr = fmt.Errorf("connection reset: %w", errors.New("net"))
	issuer := newIssuer(t, store, newFakeBungie(t))

	if _, err := login(t, issuer, "auth-code"); reasonOf(t, err) != ReasonSessionWrite {
		t.Errorf("reason = %q, want %q", reasonOf(t, err), ReasonSessionWrite)
	}
}

// TestLogin_PinsBootstrapAdmin pins ADMIN_MEMBERSHIP_IDS being applied on every
// login, and only to the listed membership.
func TestLogin_PinsBootstrapAdmin(t *testing.T) {
	t.Run("listed", func(t *testing.T) {
		store := newStubStore()
		issuer := newIssuer(t, store, newFakeBungie(t), "someone-else", testMembership)
		if _, err := login(t, issuer, "code"); err != nil {
			t.Fatal(err)
		}
		if !store.upsertAdmin {
			t.Error("a listed membership was not pinned to admin")
		}
	})
	t.Run("not listed", func(t *testing.T) {
		store := newStubStore()
		issuer := newIssuer(t, store, newFakeBungie(t), "someone-else")
		if _, err := login(t, issuer, "code"); err != nil {
			t.Fatal(err)
		}
		if store.upsertAdmin {
			t.Error("an unlisted membership was pinned to admin")
		}
	})
}

// TestLogin_RejectsBadInputBeforeReachingBungie pins that the CSRF state and the
// code bound are checked before any credential is exchanged: a forged state must
// not cost a Bungie round-trip.
func TestLogin_RejectsBadInputBeforeReachingBungie(t *testing.T) {
	bungie := newFakeBungie(t)
	issuer := newIssuer(t, newStubStore(), bungie)
	ctx := context.Background()

	_, state, nonce, err := issuer.AuthorizeURL()
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name  string
		code  string
		state string
		want  Reason
	}{
		{"empty code", "", state, ReasonInvalidCode},
		{"oversized code", strings.Repeat("a", maxAuthCodeLen+1), state, ReasonInvalidCode},
		{"empty state", "code", "", ReasonInvalidState},
		{"forged state", "code", "v1.1.aaaa.bbbb", ReasonInvalidState},
		{"state from another signer", "code", otherSignerState(t), ReasonInvalidState},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := issuer.Login(ctx, tc.code, tc.state, nonce, "ua")
			if got := reasonOf(t, err); got != tc.want {
				t.Errorf("reason = %q, want %q", got, tc.want)
			}
		})
	}
	if bungie.tokenHits != 0 {
		t.Errorf("rejected callbacks reached Bungie %d times, want 0", bungie.tokenHits)
	}
}

func otherSignerState(t *testing.T) string {
	t.Helper()
	state, _, err := NewStateSigner("a-completely-different-secret-value!!!").Generate()
	if err != nil {
		t.Fatal(err)
	}
	return state
}

// TestLogin_BungieFailuresAreDistinguishable keeps the two Bungie steps apart:
// they produce different audit reasons, and conflating them would lose the only
// signal that says which call broke.
func TestLogin_BungieFailuresAreDistinguishable(t *testing.T) {
	t.Run("code exchange", func(t *testing.T) {
		bungie := newFakeBungie(t)
		bungie.tokenStatus = http.StatusBadRequest
		issuer := newIssuer(t, newStubStore(), bungie)
		if _, err := login(t, issuer, "code"); reasonOf(t, err) != ReasonCodeExchange {
			t.Errorf("reason = %q, want %q", reasonOf(t, err), ReasonCodeExchange)
		}
	})
	t.Run("profile fetch", func(t *testing.T) {
		bungie := newFakeBungie(t)
		bungie.profileBody = `{"Response":{"destinyMemberships":[]}}`
		issuer := newIssuer(t, newStubStore(), bungie)
		if _, err := login(t, issuer, "code"); reasonOf(t, err) != ReasonMembershipFetch {
			t.Errorf("reason = %q, want %q", reasonOf(t, err), ReasonMembershipFetch)
		}
	})
}

func TestLogin_RejectsUnusableAccessTokenResponse(t *testing.T) {
	for _, body := range []string{
		`{"expires_in":3600}`,
		`{"access_token":"access","expires_in":0}`,
	} {
		bungie := newFakeBungie(t)
		bungie.tokenBody = body
		issuer := newIssuer(t, newStubStore(), bungie)
		if _, err := login(t, issuer, "code"); reasonOf(t, err) != ReasonCodeExchange {
			t.Errorf("body %s reason = %q, want %q", body, reasonOf(t, err), ReasonCodeExchange)
		}
	}
}

// TestLogin_BungieRefreshExpiryFallback pins the 90-day default for a token
// response that omits refresh_expires_in. The rule used to be written twice —
// here and in TokenStore — and now has one home.
func TestLogin_BungieRefreshExpiryFallback(t *testing.T) {
	bungie := newFakeBungie(t)
	bungie.tokenBody = `{"access_token":"a","refresh_token":"r","expires_in":3600}`
	issuer := newIssuer(t, newStubStore(), bungie)

	before := time.Now()
	if _, err := login(t, issuer, "code"); err != nil {
		t.Fatal(err)
	}
	tokens := issuer.tokens.tokens[testMembership]
	if tokens == nil {
		t.Fatal("no Bungie tokens stored")
	}
	// Ninety days spelled out, not bungieDefaultRefreshTTL: a test that reads
	// the same constant the code does moves with it and pins nothing.
	want := before.Add(90 * 24 * time.Hour)
	if delta := tokens.RefreshTokenExpiresAt.Sub(want); delta < -time.Minute || delta > time.Minute {
		t.Errorf("refresh expiry = %v, want ~%v (the 90-day fallback)", tokens.RefreshTokenExpiresAt, want)
	}
}

func TestReconnect_ReplacesOnlyBungieAuthorization(t *testing.T) {
	store := newStubStore()
	bungie := newFakeBungie(t)
	issuer := newIssuer(t, store, bungie)
	issuer.tokens.Store(testMembership, publicClientTokens(testMembership, time.Now().Add(-time.Hour)))
	_, state, nonce, err := issuer.AuthorizeURL()
	if err != nil {
		t.Fatal(err)
	}

	if err := issuer.Reconnect(context.Background(), testMembership, "new-code", state, nonce); err != nil {
		t.Fatalf("Reconnect: %v", err)
	}
	if store.upsertCalls != 0 || store.createCalls != 0 || store.rotateCalls != 0 {
		t.Fatalf("reconnect touched Guardian session persistence: upsert=%d create=%d rotate=%d",
			store.upsertCalls, store.createCalls, store.rotateCalls)
	}
	if got, err := issuer.tokens.GetValidToken(testMembership); err != nil || got != "bungie-access" {
		t.Fatalf("reconnected token = %q, %v; want bungie-access, nil", got, err)
	}
}

func TestReconnect_RejectsDifferentBungieMembership(t *testing.T) {
	store := newStubStore()
	bungie := newFakeBungie(t)
	bungie.profileBody = `{"Response":{"destinyMemberships":[{"membershipType":3,"membershipId":"4611686018467260999","displayName":"OtherGuardian"}],"primaryMembershipId":"4611686018467260999"},"ErrorCode":1}`
	issuer := newIssuer(t, store, bungie)
	_, state, nonce, err := issuer.AuthorizeURL()
	if err != nil {
		t.Fatal(err)
	}

	err = issuer.Reconnect(context.Background(), testMembership, "new-code", state, nonce)
	if got := reasonOf(t, err); got != ReasonMembershipMismatch {
		t.Fatalf("reason = %q, want %q", got, ReasonMembershipMismatch)
	}
	if store.upsertCalls != 0 || store.createCalls != 0 {
		t.Fatal("membership mismatch touched Guardian session persistence")
	}
	if _, exists := issuer.tokens.tokens[testMembership]; exists {
		t.Fatal("membership mismatch replaced the authenticated user's Bungie token")
	}
}

func TestReconnect_PersistenceFailureDoesNotReplaceAuthorization(t *testing.T) {
	store := newStubStore()
	bungie := newFakeBungie(t)
	issuer := newIssuer(t, store, bungie)
	failingTokens, _ := newTestStore(t, &fakeTokenRepo{failAll: true}, "")
	issuer.tokens = failingTokens
	_, state, nonce, err := issuer.AuthorizeURL()
	if err != nil {
		t.Fatal(err)
	}

	err = issuer.Reconnect(context.Background(), testMembership, "new-code", state, nonce)
	if got := reasonOf(t, err); got != ReasonAuthorizationWrite {
		t.Fatalf("reason = %q, want %q", got, ReasonAuthorizationWrite)
	}
	if _, exists := issuer.tokens.tokens[testMembership]; exists {
		t.Fatal("failed reconnect left a false-success authorization in memory")
	}
	if store.upsertCalls != 0 || store.createCalls != 0 || store.rotateCalls != 0 {
		t.Fatal("failed reconnect touched Guardian session persistence")
	}
}

// --- Refresh ---

func refreshToken(t *testing.T, issuer *SessionIssuer, version int, sid string) (string, string) {
	t.Helper()
	membership := &DestinyMembership{MembershipID: testMembership, DisplayName: "TestGuardian", MembershipType: 3}
	token, jti, err := issuer.jwt.GenerateRefreshToken(membership, version, sid)
	if err != nil {
		t.Fatal(err)
	}
	return token, jti
}

// TestRefresh_RotatesAgainstThePresentedJTI covers the rotation composition: the
// old token's jti is what the store compares, and the response carries the
// authoritative role rather than the one baked into the presented token.
func TestRefresh_RotatesAgainstThePresentedJTI(t *testing.T) {
	store := newStubStore()
	store.role = RoleAdmin // the role changed since the token was minted
	issuer := newIssuer(t, store, nil)
	token, jti := refreshToken(t, issuer, 7, "sess-1")

	session, err := issuer.Refresh(context.Background(), token, "ua")
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if store.rotatedOldID != jti {
		t.Errorf("rotated against %q, want the presented token's jti %q", store.rotatedOldID, jti)
	}
	if session.SessionID != "sess-1" {
		t.Errorf("session id = %q, want the one the token carried", session.SessionID)
	}
	if session.Role != RoleAdmin {
		t.Errorf("role = %d, want the store's current role %d — refresh used to drop it entirely", session.Role, RoleAdmin)
	}
	if session.AccessToken == token || session.RefreshToken == token {
		t.Error("refresh returned the presented token instead of a fresh pair")
	}
}

// TestRefresh_FailureModes covers the four ways a rotation is refused, each
// with its own reason because each drives different copy and a different audit
// event.
func TestRefresh_FailureModes(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*stubSessionStore)
		token func(*testing.T, *SessionIssuer) string
		want  Reason
	}{
		{
			name:  "missing cookie",
			token: func(*testing.T, *SessionIssuer) string { return "" },
			want:  ReasonMissingCookie,
		},
		{
			name:  "garbage token",
			token: func(*testing.T, *SessionIssuer) string { return "not-a-jwt" },
			want:  ReasonInvalidToken,
		},
		{
			name: "access token presented as refresh",
			token: func(t *testing.T, s *SessionIssuer) string {
				tok, err := s.jwt.GenerateAccessToken(&DestinyMembership{MembershipID: testMembership, MembershipType: 3}, 7, "sess-1")
				if err != nil {
					t.Fatal(err)
				}
				return tok
			},
			want: ReasonInvalidToken,
		},
		{
			name:  "revoked user-wide",
			setup: func(s *stubSessionStore) { s.version = 9 }, // token carries 7
			want:  ReasonRevoked,
		},
		{
			name:  "unknown user",
			setup: func(s *stubSessionStore) { s.found = false },
			want:  ReasonRevoked,
		},
		{
			name:  "replayed token",
			setup: func(s *stubSessionStore) { s.rotated, s.reused = false, true },
			want:  ReasonReuse,
		},
		{
			name:  "reuse wins over a commit error",
			setup: func(s *stubSessionStore) { s.rotated, s.reused, s.rotateErr = false, true, errors.New("commit failed") },
			want:  ReasonReuse,
		},
		{
			name:  "expired or unknown session",
			setup: func(s *stubSessionStore) { s.rotated, s.reused = false, false },
			want:  ReasonExpired,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := newStubStore()
			if tc.setup != nil {
				tc.setup(store)
			}
			issuer := newIssuer(t, store, nil)
			token := ""
			if tc.token != nil {
				token = tc.token(t, issuer)
			} else {
				token, _ = refreshToken(t, issuer, 7, "sess-1")
			}

			session, err := issuer.Refresh(context.Background(), token, "ua")
			if session != nil {
				t.Fatalf("a session was issued despite %s", tc.name)
			}
			if got := reasonOf(t, err); got != tc.want {
				t.Errorf("reason = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestRefresh_RotationErrorFailsOpen pins the deliberate exception: a genuine
// store failure that is *not* reuse lets the refresh through, consistent with
// the revocation check, so a database blip does not sign everyone out.
func TestRefresh_RotationErrorFailsOpen(t *testing.T) {
	store := newStubStore()
	store.rotated, store.reused, store.rotateErr = false, false, errors.New("connection reset")
	issuer := newIssuer(t, store, nil)
	token, _ := refreshToken(t, issuer, 7, "sess-1")

	session, err := issuer.Refresh(context.Background(), token, "ua")
	if err != nil {
		t.Fatalf("rotation error should fail open, got %v", err)
	}
	if session.SessionID != "sess-1" {
		t.Errorf("session id = %q, want sess-1", session.SessionID)
	}
}

// TestRefresh_AdoptsLegacyToken covers a token minted before sessions existed:
// it carries no sid, so the refresh opens a real session for it. The adopted
// row is as load-bearing as one written at login, so a failed write fails.
func TestRefresh_AdoptsLegacyToken(t *testing.T) {
	store := newStubStore()
	issuer := newIssuer(t, store, nil)
	token, _ := refreshToken(t, issuer, 7, "")

	session, err := issuer.Refresh(context.Background(), token, "ua")
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if session.SessionID == "" {
		t.Fatal("legacy refresh did not adopt a session id")
	}
	if len(store.created) != 1 {
		t.Errorf("created %d sessions, want 1 adopted", len(store.created))
	}
	// The adopted row must hold the jti of the token just issued — "a session
	// was created" is not the assertion. A row created against any other jti
	// makes the very next refresh look like a replay and revokes the session.
	refreshClaims, err := issuer.jwt.ValidateToken(session.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if got := store.created[session.SessionID]; got != refreshClaims.ID {
		t.Errorf("adopted session jti = %q, want the issued refresh token's jti %q", got, refreshClaims.ID)
	}
	if store.rotateCalls != 0 {
		t.Error("a token with no session must not rotate one")
	}

	t.Run("failed adoption fails the refresh", func(t *testing.T) {
		store := newStubStore()
		store.createErr = errors.New("write failed")
		issuer := newIssuer(t, store, nil)
		token, _ := refreshToken(t, issuer, 7, "")
		if _, err := issuer.Refresh(context.Background(), token, "ua"); reasonOf(t, err) != ReasonSessionWrite {
			t.Errorf("reason = %q, want %q", reasonOf(t, err), ReasonSessionWrite)
		}
	})
}

// --- Ending sessions ---

func TestEndSession(t *testing.T) {
	store := newStubStore()
	issuer := newIssuer(t, store, nil)

	if err := issuer.EndSession(context.Background(), "sess-1"); err != nil {
		t.Fatalf("EndSession: %v", err)
	}
	if len(store.deleted) != 1 || store.deleted[0] != "sess-1" {
		t.Errorf("deleted = %v, want [sess-1]", store.deleted)
	}

	// A token with no session id has nothing to end; it must not delete
	// something else by passing an empty id to the store.
	store.deleted = nil
	if err := issuer.EndSession(context.Background(), ""); err != nil {
		t.Fatalf("EndSession(\"\"): %v", err)
	}
	if len(store.deleted) != 0 {
		t.Errorf("empty session id reached the store: %v", store.deleted)
	}
}

func TestEndAllSessions(t *testing.T) {
	store := newStubStore()
	issuer := newIssuer(t, store, nil)
	issuer.tokens.Store(testMembership, &BungieTokens{
		AccessToken:           "a",
		AccessTokenExpiresAt:  time.Now().Add(time.Hour),
		RefreshTokenExpiresAt: time.Now().Add(time.Hour),
	})

	if err := issuer.EndAllSessions(context.Background(), testMembership); err != nil {
		t.Fatalf("EndAllSessions: %v", err)
	}
	if len(store.bumped) != 1 || len(store.deletedAll) != 1 {
		t.Errorf("bumped = %v, deletedAll = %v, want one each", store.bumped, store.deletedAll)
	}
	// The Bungie tokens stored for the user's tracked membership go too, so the
	// user cannot keep making Bungie calls from a warm cache.
	if _, err := issuer.tokens.GetValidToken(testMembership); err == nil {
		t.Error("Bungie tokens survived a sign-out-everywhere")
	}
}

// TestEndAllSessions_OneAbsentStoreDoesNotHideTheOther is why ErrUnavailable is
// filtered per call and never on a joined error: errors.Is on a join matches if
// *any* branch matches, so joining first would let a missing database swallow a
// real failure from the other write.
func TestEndAllSessions_OneAbsentStoreDoesNotHideTheOther(t *testing.T) {
	store := newStubStore()
	store.bumpErr = ErrUnavailable
	store.deleteAllEr = errors.New("delete blew up")
	issuer := newIssuer(t, store, nil)

	err := issuer.EndAllSessions(context.Background(), testMembership)
	if err == nil {
		t.Fatal("a real deletion failure was swallowed")
	}
	if errors.Is(err, ErrUnavailable) {
		t.Errorf("err = %v, want the real failure only", err)
	}
}

// TestEndSession_WithoutPersistenceSucceeds: logging out of a deployment with no
// database is not a failure, it is a no-op.
func TestEndSession_WithoutPersistenceSucceeds(t *testing.T) {
	store := newStubStore()
	store.deleteErr = ErrUnavailable
	issuer := newIssuer(t, store, nil)

	if err := issuer.EndSession(context.Background(), "sess-1"); err != nil {
		t.Errorf("EndSession without persistence = %v, want nil", err)
	}
}

func TestOAuth_RejectsBrowserMismatchBeforeExchange(t *testing.T) {
	bungie := newFakeBungie(t)
	issuer := newIssuer(t, newStubStore(), bungie)
	_, state, nonce, err := issuer.AuthorizeURL()
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(state, ".")
	parts[1] = fmt.Sprint(time.Now().Add(-OAuthTransactionTTL - time.Minute).Unix())
	payload := strings.Join(parts[:4], ".")
	expired := payload + "." + issuer.state.sign(payload)
	for _, tc := range []struct{ name, state, nonce string }{
		{"missing", state, ""}, {"different", state, "another-browser"}, {"expired", expired, nonce},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := issuer.Login(context.Background(), "fresh-code", tc.state, tc.nonce, "ua")
			if reasonOf(t, err) != ReasonInvalidState {
				t.Fatal(err)
			}
			if OAuthTransactionConsumed(err) {
				t.Fatal("invalid transaction consumed browser cookie")
			}
			err = issuer.Reconnect(context.Background(), testMembership, "fresh-code", tc.state, tc.nonce)
			if reasonOf(t, err) != ReasonInvalidState {
				t.Fatal(err)
			}
		})
	}
	if bungie.tokenHits != 0 {
		t.Fatal("invalid binding reached Bungie exchange")
	}
}

func TestLogin_BootstrapPromotionRetiresLocalRoleCache(t *testing.T) {
	for _, failed := range []bool{false, true} {
		t.Run(fmt.Sprintf("upsert_failed=%t", failed), func(t *testing.T) {
			store := newStubStore()
			store.role = RoleStandard
			issuer := newIssuer(t, store, newFakeBungie(t), testMembership)
			shared := cache.NewMemoryCache(time.Minute, 0)
			issuer.cache = shared
			revoker := NewRevocationChecker(store, shared)
			role, err := revoker.Resolve(context.Background(), testMembership, store.version, "")
			if err != nil || role != RoleStandard {
				t.Fatalf("prime standard role: %d %v", role, err)
			}
			store.upsertRole = RoleAdmin
			if failed {
				store.upsertErr = errors.New("upsert failed")
			} else {
				store.role = RoleAdmin
			}
			if _, err := login(t, issuer, "code"); err != nil {
				t.Fatal(err)
			}
			_, cached := shared.Get(authInfoCacheKey(testMembership))
			if cached != failed {
				t.Fatalf("post-bootstrap cached=%t, want %t", cached, failed)
			}
			role, err = revoker.Resolve(context.Background(), testMembership, store.version, "")
			want := RoleAdmin
			if failed {
				want = RoleStandard
			}
			if err != nil || role != want {
				t.Fatalf("post-bootstrap cached role=%d want=%d err=%v", role, want, err)
			}
		})
	}
}
