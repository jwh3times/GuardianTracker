package e2efixture

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestOAuthTokenAndMembershipRoutes(t *testing.T) {
	server := newHTTPFixture(t)
	defer server.Close()

	query := url.Values{
		"client_id":     {DefaultClientID},
		"response_type": {"code"},
		"redirect_uri":  {DefaultRedirectURI},
		"state":         {"signed.state/value"},
	}
	client := *server.Client()
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := client.Get(server.URL + "/en/OAuth/Authorize?" + query.Encode())
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("authorize status = %d", resp.StatusCode)
	}
	redirect, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		t.Fatalf("parse redirect: %v", err)
	}
	if got := redirect.Query().Get("code"); got != AuthorizationCode {
		t.Errorf("code = %q", got)
	}
	if got := redirect.Query().Get("state"); got != "signed.state/value" {
		t.Errorf("state = %q", got)
	}
	if redirect.Scheme+"://"+redirect.Host+redirect.Path != DefaultRedirectURI {
		t.Errorf("redirect target = %q", redirect.String())
	}

	bad := query
	bad.Set("redirect_uri", "http://evil.example/callback")
	badResp, err := client.Get(server.URL + "/en/OAuth/Authorize?" + bad.Encode())
	if err != nil {
		t.Fatalf("bad authorize: %v", err)
	}
	badResp.Body.Close()
	if badResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad redirect status = %d", badResp.StatusCode)
	}

	authTokens := postForm(t, server, "/platform/app/oauth/token/", url.Values{
		"grant_type": {"authorization_code"}, "code": {AuthorizationCode}, "client_id": {DefaultClientID},
	})
	if authTokens["access_token"] != AccessToken || authTokens["refresh_token"] != RefreshToken || authTokens["membership_id"] != MembershipID {
		t.Fatalf("authorization tokens = %+v", authTokens)
	}
	refreshed := postForm(t, server, "/platform/app/oauth/token/", url.Values{
		"grant_type": {"refresh_token"}, "refresh_token": {RefreshToken}, "client_id": {DefaultClientID},
	})
	if refreshed["access_token"] == AccessToken || !strings.HasPrefix(refreshed["refresh_token"].(string), RefreshToken+"-") {
		t.Fatalf("refreshed tokens = %+v", refreshed)
	}

	memberships := getBungie(t, server, "/Platform/User/GetMembershipsForCurrentUser/", true)
	entries := memberships["destinyMemberships"].([]any)
	member := entries[0].(map[string]any)
	if member["membershipId"] != MembershipID || member["displayName"] != DisplayName || int(member["membershipType"].(float64)) != MembershipType {
		t.Fatalf("membership = %+v", member)
	}
}

func TestManifestProfileVendorMilestoneAndSettingsRoutes(t *testing.T) {
	server := newHTTPFixture(t)
	defer server.Close()

	health := getJSON(t, server, "/health", false)
	if health["status"] != "ok" || health["manifestVersion"] != ManifestVersion {
		t.Fatalf("health = %+v", health)
	}

	manifest := getBungie(t, server, "/Platform/Destiny2/Manifest/", false)
	if manifest["version"] != ManifestVersion {
		t.Fatalf("manifest = %+v", manifest)
	}
	paths := manifest["mobileWorldContentPaths"].(map[string]any)
	if paths["en"] != "/fixtures/manifest.zip" {
		t.Fatalf("manifest paths = %+v", paths)
	}
	archiveResp, err := server.Client().Get(server.URL + "/fixtures/manifest.zip")
	if err != nil {
		t.Fatalf("manifest archive: %v", err)
	}
	archive, err := io.ReadAll(archiveResp.Body)
	archiveResp.Body.Close()
	if err != nil || archiveResp.StatusCode != http.StatusOK || len(archive) < 4 || !bytes.Equal(archive[:2], []byte("PK")) {
		t.Fatalf("archive status=%d bytes=%d err=%v", archiveResp.StatusCode, len(archive), err)
	}

	profileBase := "/Platform/Destiny2/3/Profile/" + MembershipID + "/?components="
	collectibles := getBungie(t, server, profileBase+"800", true)
	if _, ok := collectibles["profileCollectibles"]; !ok {
		t.Fatalf("component 800 = %+v", collectibles)
	}
	characters := getBungie(t, server, profileBase+"200", true)
	characterData := characters["characters"].(map[string]any)["data"].(map[string]any)
	if _, ok := characterData[CharacterID]; !ok {
		t.Fatalf("component 200 = %+v", characters)
	}
	records := getBungie(t, server, profileBase+"900", true)
	recordData := records["profileRecords"].(map[string]any)["data"].(map[string]any)["records"].(map[string]any)
	if len(recordData) != 5 {
		t.Fatalf("component 900 record count = %d", len(recordData))
	}

	characterVendorPath := "/Platform/Destiny2/3/Profile/" + MembershipID + "/Character/" + CharacterID + "/Vendors/?components=400,402"
	characterVendors := getBungie(t, server, characterVendorPath, true)
	vendorData := characterVendors["vendors"].(map[string]any)["data"].(map[string]any)
	if _, ok := vendorData["2190858386"]; !ok {
		t.Fatalf("character vendors = %+v", characterVendors)
	}
	publicVendors := getBungie(t, server, "/Platform/Destiny2/Vendors/?components=400,402", false)
	publicSales := publicVendors["sales"].(map[string]any)["data"].(map[string]any)
	if _, ok := publicSales["2190858386"]; !ok {
		t.Fatalf("public vendors = %+v", publicVendors)
	}

	milestones := getBungie(t, server, "/Platform/Destiny2/Milestones/", false)
	if _, ok := milestones["9500"]; !ok {
		t.Fatalf("milestones = %+v", milestones)
	}
	settings := getBungie(t, server, "/Platform/Settings/", false)
	core := settings["destiny2CoreSettings"].(map[string]any)
	if int(core["activeSealsRootNodeHash"].(float64)) != int(activeSealsRootHash) || int(core["exoticCatalystsRootNodeHash"].(float64)) != int(combinedRecordsRootHash) {
		t.Fatalf("settings = %+v", settings)
	}

	notFound, err := server.Client().Get(server.URL + "/Platform/not-real")
	if err != nil {
		t.Fatalf("unknown route: %v", err)
	}
	notFound.Body.Close()
	if notFound.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown route status = %d", notFound.StatusCode)
	}
}

func TestScenarioControlDrivesWeeklyAndVendorEmptyState(t *testing.T) {
	server := newHTTPFixture(t)
	defer server.Close()

	requestScenario(t, server, http.MethodPut, `{"weekly":"empty"}`, http.StatusOK)
	if milestones := getBungie(t, server, "/Platform/Destiny2/Milestones/", false); len(milestones) != 0 {
		t.Fatalf("empty milestones = %+v", milestones)
	}
	public := getBungie(t, server, "/Platform/Destiny2/Vendors/?components=400,402", false)
	if sales := public["sales"].(map[string]any)["data"].(map[string]any); len(sales) != 0 {
		t.Fatalf("empty public vendor sales = %+v", sales)
	}
	characterPath := "/Platform/Destiny2/3/Profile/" + MembershipID + "/Character/" + CharacterID + "/Vendors/?components=400,402"
	character := getBungie(t, server, characterPath, true)
	if sales := character["sales"].(map[string]any)["data"].(map[string]any); len(sales) != 0 {
		t.Fatalf("empty character vendor sales = %+v", sales)
	}

	requestScenario(t, server, http.MethodDelete, "", http.StatusOK)
	if milestones := getBungie(t, server, "/Platform/Destiny2/Milestones/", false); len(milestones) != 1 {
		t.Fatalf("reset milestones = %+v", milestones)
	}
	public = getBungie(t, server, "/Platform/Destiny2/Vendors/?components=400,402", false)
	if sales := public["sales"].(map[string]any)["data"].(map[string]any); len(sales) != 1 {
		t.Fatalf("reset public vendor sales = %+v", sales)
	}

	requestScenario(t, server, http.MethodPut, `{"weekly":"surprise"}`, http.StatusBadRequest)
}

func TestScenarioControlRejectsNonLoopback(t *testing.T) {
	fixture, err := NewServer(ServerOptions{})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	req := httptest.NewRequest(http.MethodPut, "/__e2e/scenario", strings.NewReader(`{"weekly":"empty"}`))
	req.RemoteAddr = "203.0.113.10:4242"
	recorder := httptest.NewRecorder()
	fixture.Handler().ServeHTTP(recorder, req)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("non-loopback status = %d", recorder.Code)
	}
}

func newHTTPFixture(t *testing.T) *httptest.Server {
	t.Helper()
	fixture, err := NewServer(ServerOptions{})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	return httptest.NewServer(fixture.Handler())
}

func postForm(t *testing.T, server *httptest.Server, path string, values url.Values) map[string]any {
	t.Helper()
	resp, err := server.Client().PostForm(server.URL+path, values)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST %s status = %d", path, resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode POST %s: %v", path, err)
	}
	return body
}

func getJSON(t *testing.T, server *httptest.Server, path string, authenticated bool) map[string]any {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, server.URL+path, nil)
	if err != nil {
		t.Fatalf("new GET %s: %v", path, err)
	}
	if authenticated {
		req.Header.Set("Authorization", "Bearer "+AccessToken)
	}
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET %s status=%d body=%s", path, resp.StatusCode, body)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode GET %s: %v", path, err)
	}
	return body
}

func getBungie(t *testing.T, server *httptest.Server, path string, authenticated bool) map[string]any {
	t.Helper()
	body := getJSON(t, server, path, authenticated)
	if int(body["ErrorCode"].(float64)) != 1 {
		t.Fatalf("GET %s Bungie error = %+v", path, body)
	}
	response, ok := body["Response"].(map[string]any)
	if !ok {
		t.Fatalf("GET %s Response = %#v", path, body["Response"])
	}
	return response
}

func requestScenario(t *testing.T, server *httptest.Server, method, body string, wantStatus int) {
	t.Helper()
	req, err := http.NewRequest(method, server.URL+"/__e2e/scenario", strings.NewReader(body))
	if err != nil {
		t.Fatalf("new scenario request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("scenario request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != wantStatus {
		responseBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("scenario status=%d want=%d body=%s", resp.StatusCode, wantStatus, responseBody)
	}
}
