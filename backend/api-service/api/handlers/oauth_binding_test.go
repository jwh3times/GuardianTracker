package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"guardian-tracker/api-service/auth"
	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/config"
)

func TestOAuth_BrowserTransactions(t *testing.T) {
	var exchanges atomic.Int32
	bungie := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "GetMembershipsForCurrentUser") {
			fmt.Fprintf(w, `{"Response":{"destinyMemberships":[{"membershipType":3,"membershipId":"%s","displayName":"TestGuardian"}],"primaryMembershipId":"%s"},"ErrorCode":1}`, testUserID, testUserID)
			return
		}
		exchanges.Add(1)
		fmt.Fprint(w, `{"access_token":"access","expires_in":3600}`)
	}))
	defer bungie.Close()
	cfg := &config.Config{JWTSecret: testJWTSecret}
	store := newFakeUserStore()
	issuer := auth.NewSessionIssuer(auth.SessionDeps{
		JWT: auth.NewJWT(testJWTSecret, 24, 30), Tokens: newTokenStore(t), Users: store,
		Revoker: auth.NewRevocationChecker(store, cache.NewNoOpCache()), Cache: cache.NewNoOpCache(), State: auth.NewStateSigner(testJWTSecret),
		OAuth: auth.OAuthConfig{ClientID: "client", TokenURL: bungie.URL, APIBaseURL: bungie.URL},
	})
	h := NewAuthHandler(issuer, cfg, nil)
	router := gin.New()
	router.GET("/start", h.GetBungieAuthURL)
	router.POST("/login", h.BungieCallback)
	router.POST("/reconnect", func(c *gin.Context) { c.Set("membership_id", testUserID); h.ReconnectBungie(c) })
	server := httptest.NewServer(router)
	defer server.Close()
	newBrowser := func() *http.Client {
		jar, err := cookiejar.New(nil)
		if err != nil {
			t.Fatal(err)
		}
		return &http.Client{Jar: jar}
	}
	attacker, victim := newBrowser(), newBrowser()
	start := func(client *http.Client) string {
		resp, err := client.Get(server.URL + "/start")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.Header.Get("Cache-Control") != "no-store" {
			t.Fatal("authorization start can be cached")
		}
		var body struct {
			State string `json:"state"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		cookies := resp.Cookies()
		if len(cookies) != 1 {
			t.Fatalf("start cookies=%v", cookies)
		}
		c := cookies[0]
		if !c.HttpOnly || c.SameSite != http.SameSiteLaxMode || c.Path != "/" || c.Domain != "" || c.MaxAge != 600 {
			t.Fatalf("transaction cookie=%+v", c)
		}
		if strings.Contains(body.State, c.Value) {
			t.Fatal("state leaked cookie secret")
		}
		return body.State
	}
	submit := func(client *http.Client, path, state string, want int) []*http.Cookie {
		resp, err := client.PostForm(server.URL+path, url.Values{"code": {"fresh-code"}, "state": {state}})
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != want {
			t.Fatalf("%s status=%d want=%d", path, resp.StatusCode, want)
		}
		return resp.Cookies()
	}
	attackerState := start(attacker)
	for _, path := range []string{"/login", "/reconnect"} {
		if cookies := submit(victim, path, attackerState, 400); len(cookies) != 0 {
			t.Fatal("foreign state cleared browser cookies")
		}
	}
	expiredState := start(victim)
	origin, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	victim.Jar.SetCookies(origin, []*http.Cookie{{Name: h.oauthCookieName(), Path: "/", Expires: time.Now().Add(-time.Minute), MaxAge: -1}})
	for _, path := range []string{"/login", "/reconnect"} {
		submit(victim, path, expiredState, 400)
	}
	victimState := start(victim)
	for _, path := range []string{"/login", "/reconnect"} {
		submit(victim, path, attackerState, 400)
	}
	if exchanges.Load() != 0 {
		t.Fatal("foreign transaction reached Bungie")
	}
	cookies := submit(victim, "/login", victimState, 200)
	consumed := false
	for _, c := range cookies {
		if c.Name == h.oauthCookieName() && c.MaxAge < 0 {
			consumed = true
		}
	}
	if !consumed {
		t.Fatal("valid login did not consume transaction")
	}
	submit(victim, "/login", victimState, 400)
	old := start(victim)
	latest := start(victim)
	for _, path := range []string{"/login", "/reconnect"} {
		submit(victim, path, old, 400)
	}
	if exchanges.Load() != 1 {
		t.Fatal("superseded transaction reached Bungie")
	}
	cookies = submit(victim, "/reconnect", latest, 204)
	for _, c := range cookies {
		if c.Name == refreshCookieName {
			t.Fatal("reconnect changed Guardian session cookie")
		}
	}
	// The initiating browser can still use its independently pending transaction.
	submit(attacker, "/login", attackerState, 200)
	if exchanges.Load() != 3 {
		t.Fatalf("exchange count=%d", exchanges.Load())
	}
}

func TestOAuthCookie_ProductionHostPrefix(t *testing.T) {
	h, _ := newAuthHandler(t)
	h.cfg.GoEnv = "production"
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	h.GetBungieAuthURL(c)
	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies=%v", cookies)
	}
	cookie := cookies[0]
	if cookie.Name != "__Host-guardian_oauth_transaction" || !cookie.Secure || !cookie.HttpOnly || cookie.Path != "/" || cookie.Domain != "" || cookie.SameSite != http.SameSiteLaxMode || !cookie.Expires.After(time.Now()) {
		t.Fatalf("production transaction cookie=%+v", cookie)
	}
}
