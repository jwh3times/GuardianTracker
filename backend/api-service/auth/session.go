package auth

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"time"

	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/observability"

	"github.com/google/uuid"
)

// ErrUnavailable reports that persistence is absent, not that a write failed.
// The db adapter translates db.ErrUnavailable into it (see db/adapters).
//
// The distinction is load-bearing at exactly one place: a login whose session
// row cannot be written must fail, because the access token is rejected on its
// next request when no live session exists — unless there is no database at
// all, in which case no session row was ever going to exist and nothing checks
// for one. Conflating the two is what made login return 500 in degraded mode.
var ErrUnavailable = errors.New("auth: persistence not configured")

// maxAuthCodeLen bounds the authorization code accepted from the callback form.
const maxAuthCodeLen = 500

// stateTTL is how long an issued OAuth state parameter stays valid.
const stateTTL = 10 * time.Minute

// Reason names why session issuance failed. It is also the audit reason string,
// so the two cannot drift: an added failure mode has to be given a name here
// before a handler can report it.
type Reason string

const (
	ReasonInvalidCode  Reason = "invalid_code"
	ReasonInvalidState Reason = "invalid_state"
	ReasonCodeExchange Reason = "code_exchange"
	// ReasonMembershipFetch retains the historical audit reason value while
	// naming what the Bungie call actually resolves.
	ReasonMembershipFetch Reason = "profile_fetch"
	ReasonTokenMint       Reason = "token_mint"
	ReasonSessionWrite    Reason = "session_write"
	ReasonMissingCookie   Reason = "missing_cookie"
	ReasonInvalidToken    Reason = "invalid_token"
	ReasonRevoked         Reason = "revoked"
	ReasonReuse           Reason = "reuse"
	ReasonExpired         Reason = "expired"
)

// SessionError is every failure Login and Refresh report. It carries the
// identity the caller can no longer recover for itself: once the issuer owns
// refresh-token validation, the handler has no claims to read a membership or
// session id from when it writes the audit event.
type SessionError struct {
	Reason       Reason
	MembershipID string
	SessionID    string
	Err          error
}

func (e *SessionError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("session %s: %v", e.Reason, e.Err)
	}
	return fmt.Sprintf("session %s", e.Reason)
}

func (e *SessionError) Unwrap() error { return e.Err }

// AsSessionError unwraps err to the session failure it carries, reporting false
// when it carries none. Callers need the reason and the identity together — the
// reason picks the response, the identity fills the audit event.
func AsSessionError(err error) (*SessionError, bool) {
	if se, ok := errors.AsType[*SessionError](err); ok {
		return se, true
	}
	return nil, false
}

// Session is one issued browser session: the pair of JWTs, who they belong to,
// and the server-side session id they are bound to.
type Session struct {
	AccessToken      string
	RefreshToken     string
	RefreshExpiresAt time.Time
	Membership       DestinyMembership
	Role             int
	SessionID        string
	// AppUserID is the Guardian Tracker users-row id for the audit trail. Nil when the row could
	// not be read, which is not fatal to the login.
	AppUserID *int64
}

// SessionStore is the slice of the user store that session issuance needs.
// Satisfied by db.UserStore through db/adapters.NewSessionStore, which is also
// what translates db.ErrUnavailable into ErrUnavailable.
type SessionStore interface {
	Upsert(ctx context.Context, membershipID string, membershipType int16, displayName string, forceAdmin bool) (id int64, tokenVersion int, role int16, err error)
	CreateSession(ctx context.Context, id, membershipID, jti, userAgent string, expiresAt time.Time) error
	RotateSession(ctx context.Context, id, membershipID, oldJTI, newJTI string, newExpiresAt time.Time) (rotated bool, reused bool, err error)
	DeleteSession(ctx context.Context, id string) error
	DeleteUserSessions(ctx context.Context, membershipID string) error
	BumpTokenVersion(ctx context.Context, membershipID string) error
}

// OAuthConfig is the Bungie OAuth wiring the issuer needs. main.go builds it
// from config.Config so this package stays free of configuration loading.
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	// TokenURL and APIBaseURL are overridable for the fake-Bungie fixture;
	// an empty TokenURL uses Bungie's real endpoint.
	TokenURL     string
	APIBaseURL   string
	APIKey       string
	AuthorizeURL string
	RedirectURI  string
	// BootstrapAdmins are membership IDs pinned to the admin role on every
	// login (ADMIN_MEMBERSHIP_IDS).
	BootstrapAdmins []string
}

// SessionDeps are the collaborators SessionIssuer composes. None may be nil:
// a session issuer that silently skips revocation or persistence would issue
// credentials that look valid and are not.
type SessionDeps struct {
	JWT     *JWT
	Tokens  *TokenStore
	Users   SessionStore
	Revoker *RevocationChecker
	Cache   cache.Cache
	State   *StateSigner
	OAuth   OAuthConfig
}

// SessionIssuer owns the browser session lifecycle: start the OAuth flow, turn
// a callback into a session, rotate one, and end one.
//
// It exists because that lifecycle was two Gin handlers reaching nine
// collaborators through *gin.Context, so none of the composed rules — the
// token-version fallback, admin pinning, "a failed session write must fail the
// login", reuse detection — could be reached by a test without an HTTP request.
// Everything HTTP-shaped (status codes, cookies, audit events, response bodies)
// stays in the handler; everything about what a session *is* lives here.
type SessionIssuer struct {
	jwt     *JWT
	tokens  *TokenStore
	users   SessionStore
	revoker *RevocationChecker
	cache   cache.Cache
	state   *StateSigner
	oauth   OAuthConfig
	bungie  *bungieOAuth
}

func NewSessionIssuer(d SessionDeps) *SessionIssuer {
	return &SessionIssuer{
		jwt:     d.JWT,
		tokens:  d.Tokens,
		users:   d.Users,
		revoker: d.Revoker,
		cache:   d.Cache,
		state:   d.State,
		oauth:   d.OAuth,
		bungie:  newBungieOAuth(d.OAuth.ClientID, d.OAuth.ClientSecret, d.OAuth.TokenURL),
	}
}

// AuthorizeURL returns the Bungie authorization URL to send the browser to,
// along with the signed state it embeds. Generating and verifying that state
// are one CSRF rule, so they have one owner.
func (s *SessionIssuer) AuthorizeURL() (authURL, state string, err error) {
	state, err = s.state.Generate()
	if err != nil {
		return "", "", err
	}
	return fmt.Sprintf(
		"%s?client_id=%s&response_type=code&redirect_uri=%s&state=%s",
		s.oauth.AuthorizeURL,
		s.oauth.ClientID,
		url.QueryEscape(s.oauth.RedirectURI),
		url.QueryEscape(state),
	), state, nil
}

// Login turns an OAuth callback into a session: verify state, exchange the
// code, resolve the primary Destiny membership, upsert the user, store the
// Bungie tokens, mint the JWT pair, and record the session row they rotate against.
func (s *SessionIssuer) Login(ctx context.Context, code, state, userAgent string) (*Session, error) {
	if code == "" || len(code) > maxAuthCodeLen {
		return nil, &SessionError{Reason: ReasonInvalidCode}
	}
	if state == "" || !s.state.Verify(state, time.Now(), stateTTL) {
		return nil, &SessionError{Reason: ReasonInvalidState}
	}

	bungieTokens, err := s.bungie.exchangeCode(ctx, code)
	if err != nil {
		return nil, &SessionError{Reason: ReasonCodeExchange, Err: err}
	}
	membership, err := s.bungie.primaryDestinyMembership(ctx, s.oauth.APIBaseURL, s.oauth.APIKey, bungieTokens.AccessToken)
	if err != nil {
		return nil, &SessionError{Reason: ReasonMembershipFetch, Err: err}
	}

	// The Guardian Tracker user row is best-effort: a login that cannot be recorded still works,
	// with the token version that revokes nothing and the tier that grants
	// nothing. Membership IDs in ADMIN_MEMBERSHIP_IDS are pinned on every login.
	tokenVersion, role := 1, RoleStandard
	var appUserID *int64
	forceAdmin := slices.Contains(s.oauth.BootstrapAdmins, membership.MembershipID)
	id, tv, r, err := s.users.Upsert(ctx, membership.MembershipID, int16(membership.MembershipType), membership.DisplayName, forceAdmin)
	if err != nil {
		observability.Logger(ctx).WarnContext(ctx, "login user persistence failed",
			observability.ID("membership", membership.MembershipID), observability.Err(err))
	} else {
		tokenVersion, role, appUserID = tv, int(r), &id
	}

	s.tokens.Store(membership.MembershipID, bungieTokens)

	// Each login opens a new per-device session. Both JWTs carry its id (sid).
	sessionID := uuid.NewString()
	session, jti, err := s.mint(membership, tokenVersion, role, sessionID)
	if err != nil {
		return nil, &SessionError{Reason: ReasonTokenMint, MembershipID: membership.MembershipID, SessionID: sessionID, Err: err}
	}
	session.AppUserID = appUserID

	if err := s.recordSession(ctx, sessionID, membership.MembershipID, jti, userAgent, session.RefreshExpiresAt); err != nil {
		return nil, err
	}
	return session, nil
}

// Refresh rotates a session: validate the presented refresh JWT, check it has
// not been revoked user-wide, mint a new pair, and advance the session row
// with reuse detection.
func (s *SessionIssuer) Refresh(ctx context.Context, refreshJWT, userAgent string) (*Session, error) {
	if refreshJWT == "" {
		return nil, &SessionError{Reason: ReasonMissingCookie}
	}
	claims, err := s.jwt.ValidateToken(refreshJWT)
	if err != nil {
		return nil, &SessionError{Reason: ReasonInvalidToken, Err: err}
	}
	if claims.TokenType != "refresh" {
		return nil, &SessionError{Reason: ReasonInvalidToken, Err: fmt.Errorf("token type %q is not a refresh token", claims.TokenType)}
	}

	// Reject refresh attempts after a sign-out-everywhere (token_version bump).
	// Resolve rather than Check, because the role it reads is the authoritative
	// one and the response carries it; Check throws it away.
	role, err := s.revoker.Resolve(ctx, claims.MembershipID, claims.TokenVersion, "")
	if err != nil {
		return nil, &SessionError{Reason: ReasonRevoked, MembershipID: claims.MembershipID, SessionID: claims.SessionID, Err: err}
	}

	membership := &DestinyMembership{
		MembershipID:   claims.MembershipID,
		DisplayName:    claims.DisplayName,
		MembershipType: claims.MembershipType,
	}

	// A token issued before sessions existed carries no sid; adopt it into a
	// fresh session on its first refresh so later refreshes are protected.
	sessionID := claims.SessionID
	adopt := sessionID == ""
	if adopt {
		sessionID = uuid.NewString()
	}

	session, newJTI, err := s.mint(membership, claims.TokenVersion, role, sessionID)
	if err != nil {
		return nil, &SessionError{Reason: ReasonTokenMint, MembershipID: claims.MembershipID, SessionID: sessionID, Err: err}
	}

	if adopt {
		// The adopted session is load-bearing for the new access token, so this
		// write is as required as the one at login.
		if err := s.recordSession(ctx, sessionID, claims.MembershipID, newJTI, userAgent, session.RefreshExpiresAt); err != nil {
			return nil, err
		}
		return session, nil
	}

	// Rotate the session's refresh jti with reuse detection. A replayed (already
	// rotated) token revokes the session; an unknown or expired session forces a
	// re-login. expires_at slides forward to match the freshly issued token.
	rotated, reused, err := s.users.RotateSession(ctx, sessionID, claims.MembershipID, claims.ID, newJTI, session.RefreshExpiresAt)
	switch {
	case reused:
		observability.Logger(ctx).WarnContext(ctx, "refresh token reuse detected; session revoked",
			observability.ID("membership", claims.MembershipID), observability.ID("session", sessionID))
		return nil, &SessionError{Reason: ReasonReuse, MembershipID: claims.MembershipID, SessionID: sessionID}
	case err != nil:
		// Genuine store failure (not reuse): fail open, consistent with the
		// revocation check above — the new jti goes unrecorded, so the next
		// refresh may force a re-login.
		observability.Logger(ctx).WarnContext(ctx, "refresh session rotation failed open",
			observability.ID("membership", claims.MembershipID), observability.ID("session", sessionID), observability.Err(err))
	case !rotated:
		return nil, &SessionError{Reason: ReasonExpired, MembershipID: claims.MembershipID, SessionID: sessionID}
	}
	return session, nil
}

// EndSession ends one device's session: the row it rotates against, plus the
// cached "this session is live" answer so the access token stops working on its
// next request rather than lingering until the cache entry ages out.
func (s *SessionIssuer) EndSession(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return nil
	}
	err := s.users.DeleteSession(ctx, sessionID)
	s.cache.Delete(sessionCacheKey(sessionID))
	return ignoreUnavailable(err)
}

// EndAllSessions signs a user out everywhere: bump token_version (invalidating
// every access token), delete every session row, and drop the Bungie tokens
// stored for the user's tracked membership. The cached auth info is evicted so
// the bump takes effect at
// once; other sessions' cache entries age out, but their access tokens already
// fail the version check.
func (s *SessionIssuer) EndAllSessions(ctx context.Context, membershipID string) error {
	bumped := s.users.BumpTokenVersion(ctx, membershipID)
	deleted := s.users.DeleteUserSessions(ctx, membershipID)
	s.tokens.Delete(membershipID)
	s.cache.Delete(authInfoCacheKey(membershipID))
	return errors.Join(ignoreUnavailable(bumped), ignoreUnavailable(deleted))
}

// mint issues the JWT pair for a session and returns the refresh token's jti,
// which is what the session row stores for reuse detection.
func (s *SessionIssuer) mint(membership *DestinyMembership, tokenVersion, role int, sessionID string) (*Session, string, error) {
	access, err := s.jwt.GenerateAccessToken(membership, tokenVersion, sessionID)
	if err != nil {
		return nil, "", fmt.Errorf("access token: %w", err)
	}
	refresh, jti, err := s.jwt.GenerateRefreshToken(membership, tokenVersion, sessionID)
	if err != nil {
		return nil, "", fmt.Errorf("refresh token: %w", err)
	}
	return &Session{
		AccessToken:      access,
		RefreshToken:     refresh,
		RefreshExpiresAt: time.Now().Add(s.jwt.RefreshTokenTTL()),
		Membership:       *membership,
		Role:             role,
		SessionID:        sessionID,
	}, jti, nil
}

// recordSession writes the session row the freshly minted tokens rotate
// against. A failure fails the caller — the access token is rejected on its
// next request without a live row, so returning one would hand back a session
// that is dead on arrival. Running without a database is the one exception:
// there is no row to write and nothing that reads one.
func (s *SessionIssuer) recordSession(ctx context.Context, sessionID, membershipID, jti, userAgent string, expiresAt time.Time) error {
	err := s.users.CreateSession(ctx, sessionID, membershipID, jti, userAgent, expiresAt)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrUnavailable):
		observability.Logger(ctx).InfoContext(ctx, "session issued without persistence; no session row recorded",
			observability.ID("membership", membershipID), observability.ID("session", sessionID))
		return nil
	default:
		observability.Logger(ctx).ErrorContext(ctx, "refresh session creation failed",
			observability.ID("membership", membershipID), observability.ID("session", sessionID), observability.Err(err))
		return &SessionError{Reason: ReasonSessionWrite, MembershipID: membershipID, SessionID: sessionID, Err: err}
	}
}

// ignoreUnavailable drops "there is no database" and passes every real failure
// through. Applied per call, never to a joined error, so one absent store
// cannot swallow another store's genuine failure.
func ignoreUnavailable(err error) error {
	if errors.Is(err, ErrUnavailable) {
		return nil
	}
	return err
}
