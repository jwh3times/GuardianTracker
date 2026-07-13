package bungie

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBungieError_Error(t *testing.T) {
	e := &BungieError{ErrorCode: 5, ErrorStatus: "SystemDisabled", Message: "down"}
	if got := e.Error(); !strings.Contains(got, "SystemDisabled") || !strings.Contains(got, "down") {
		t.Errorf("Error() = %q", got)
	}
}

func TestGetCharacters_ParsesAndSendsAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer tok" {
			t.Errorf("Authorization = %q", got)
		}
		fmt.Fprint(w, `{"ErrorCode":1,"Response":{"characters":{"data":{
			"char1":{"characterId":"char1","classType":1,"raceType":1,"light":2010}}}}}`)
	}))
	defer srv.Close()

	c := NewClient("k", srv.URL, 100, 100)
	resp, err := c.GetCharacters(context.Background(), 3, "4611686018467260757", "tok")
	if err != nil {
		t.Fatalf("GetCharacters: %v", err)
	}
	ch, ok := resp.Response.Characters.Data["char1"]
	if !ok || ch.Light != 2010 || ch.ClassType != 1 {
		t.Errorf("character data = %+v", resp.Response.Characters.Data)
	}
}

func TestGetRecords_Parses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ErrorCode":1,"Response":{"profileRecords":{"data":{"records":{
			"123":{"state":4,"objectives":[{"objectiveHash":1,"progress":5,"completionValue":10,"complete":false}]}}}}}}`)
	}))
	defer srv.Close()

	c := NewClient("k", srv.URL, 100, 100)
	resp, err := c.GetRecords(context.Background(), 3, "id", "tok")
	if err != nil {
		t.Fatalf("GetRecords: %v", err)
	}
	rec, ok := resp.Response.ProfileRecords.Data.Records["123"]
	if !ok || rec.State != 4 || len(rec.Objectives) != 1 {
		t.Errorf("records = %+v", resp.Response.ProfileRecords.Data.Records)
	}
}

func TestGetPublicMilestones_ReturnsMap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ErrorCode":1,"Response":{"999":{"milestoneHash":999,
			"activities":[{"activityHash":42,"modifierHashes":[7]}]}}}`)
	}))
	defer srv.Close()

	c := NewClient("k", srv.URL, 100, 100)
	m, err := c.GetPublicMilestones(context.Background())
	if err != nil {
		t.Fatalf("GetPublicMilestones: %v", err)
	}
	ms, ok := m["999"]
	if !ok || ms.MilestoneHash != 999 || len(ms.Activities) != 1 {
		t.Errorf("milestones = %+v", m)
	}
}

func TestGetPublicMilestones_PropagatesBungieError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ErrorCode":5,"ErrorStatus":"SystemDisabled","Message":"down"}`)
	}))
	defer srv.Close()

	c := NewClient("k", srv.URL, 100, 100)
	if _, err := c.GetPublicMilestones(context.Background()); err == nil {
		t.Fatal("expected error for ErrorCode 5")
	}
}

func TestGetCharacterVendors_Parses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer tok" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.URL.Query().Get("components"); got != "400,402" {
			t.Errorf("components = %q, want 400,402", got)
		}
		fmt.Fprint(w, `{"ErrorCode":1,"Response":{"vendors":{"data":{
			"2190858386":{"vendorHash":2190858386,"vendorLocationIndex":0,"enabled":true}}},"sales":{"data":{
			"672118013":{"saleItems":{"0":{"itemHash":100,"costs":[{"itemHash":3,"quantity":1}]}}}}}}}`)
	}))
	defer srv.Close()

	c := NewClient("k", srv.URL, 100, 100)
	resp, err := c.GetCharacterVendors(context.Background(), 3, "id", "char1", "tok")
	if err != nil {
		t.Fatalf("GetCharacterVendors: %v", err)
	}
	if _, ok := resp.Response.Sales.Data["672118013"]; !ok {
		t.Errorf("vendor sales = %+v", resp.Response.Sales.Data)
	}
	xur, ok := resp.Response.Vendors.Data["2190858386"]
	if !ok || xur.VendorLocationIndex != 0 || !xur.Enabled {
		t.Errorf("Xur vendor component = %+v", xur)
	}
}

func TestGetPublicVendors_Parses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ErrorCode":1,"Response":{"sales":{"data":{
			"2190858386":{"saleItems":{"0":{"itemHash":555}}}}}}}`)
	}))
	defer srv.Close()

	c := NewClient("k", srv.URL, 100, 100)
	resp, err := c.GetPublicVendors(context.Background())
	if err != nil {
		t.Fatalf("GetPublicVendors: %v", err)
	}
	if _, ok := resp.Response.Sales.Data["2190858386"]; !ok {
		t.Errorf("public vendor sales = %+v", resp.Response.Sales.Data)
	}
}

func TestGetCommonSettings_ParsesRootNodes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// GetCommonSettings strips "/Destiny2" and calls /Settings/.
		if !strings.HasSuffix(r.URL.Path, "/Settings/") {
			t.Errorf("path = %q, want .../Settings/", r.URL.Path)
		}
		fmt.Fprint(w, `{"ErrorCode":1,"Response":{"destiny2CoreSettings":{
			"activeSealsRootNodeHash":1,"legacySealsRootNodeHash":2,
			"exoticCatalystsRootNodeHash":3,"craftingRootNodeHash":4}}}`)
	}))
	defer srv.Close()

	// baseURL ends in /Destiny2 so the trim path is exercised.
	c := NewClient("k", srv.URL+"/Destiny2", 100, 100)
	s, err := c.GetCommonSettings(context.Background())
	if err != nil {
		t.Fatalf("GetCommonSettings: %v", err)
	}
	if s.ActiveSealsRootNodeHash != 1 || s.CraftingRootNodeHash != 4 {
		t.Errorf("settings = %+v", s)
	}
}

func TestGetCommonSettings_BungieError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ErrorCode":5,"ErrorStatus":"SystemDisabled"}`)
	}))
	defer srv.Close()

	c := NewClient("k", srv.URL, 100, 100)
	if _, err := c.GetCommonSettings(context.Background()); err == nil {
		t.Fatal("expected BungieError for ErrorCode 5")
	}
}

func TestDownloadFileToPath_WritesFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("manifest-bytes"))
	}))
	defer srv.Close()

	c := NewClient("k", "http://unused", 100, 100)
	dest := filepath.Join(t.TempDir(), "world.content")
	if err := c.DownloadFileToPath(context.Background(), srv.URL+"/world.content", dest); err != nil {
		t.Fatalf("DownloadFileToPath: %v", err)
	}
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read dest: %v", err)
	}
	if string(data) != "manifest-bytes" {
		t.Errorf("data = %q", data)
	}
}

func TestDownloadFileToPath_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewClient("k", "http://unused", 100, 100)
	dest := filepath.Join(t.TempDir(), "world.content")
	if err := c.DownloadFileToPath(context.Background(), srv.URL+"/missing", dest); err == nil {
		t.Fatal("expected error for 404 download")
	}
}

func TestSetCDNBaseURL(t *testing.T) {
	c := NewClient("k", "http://unused", 100, 100)
	if c.cdnBaseURL != "https://www.bungie.net" {
		t.Errorf("default cdnBaseURL = %q", c.cdnBaseURL)
	}
	c.SetCDNBaseURL("http://fake-bungie:8090/")
	if c.cdnBaseURL != "http://fake-bungie:8090" {
		t.Errorf("cdnBaseURL after SetCDNBaseURL = %q, want trailing slash trimmed", c.cdnBaseURL)
	}
	// Empty string must not clobber the existing value.
	c.SetCDNBaseURL("")
	if c.cdnBaseURL != "http://fake-bungie:8090" {
		t.Errorf("cdnBaseURL after empty SetCDNBaseURL = %q, want unchanged", c.cdnBaseURL)
	}
}
