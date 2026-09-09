package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/collections"

	"github.com/gin-gonic/gin"
)

func TestIsValidMembershipID(t *testing.T) {
	cases := []struct {
		name string
		id   string
		want bool
	}{
		{"valid", "4611686018467260757", true},
		{"min length 10", "1234567890", true},
		{"too short", "123456789", false},
		{"too long", strings.Repeat("1", 26), false},
		{"letters", "46116860184abc60757", false},
		{"empty", "", false},
		{"sql injection", "1; DROP TABLE users", false},
		{"negative", "-4611686018467260757", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isValidMembershipID(tc.id); got != tc.want {
				t.Errorf("isValidMembershipID(%q) = %v, want %v", tc.id, got, tc.want)
			}
		})
	}
}

func TestIsValidMembershipType(t *testing.T) {
	for _, valid := range []int{1, 2, 3, 4, 5, 6, 10, 254} {
		if !isValidMembershipType(valid) {
			t.Errorf("isValidMembershipType(%d) = false, want true", valid)
		}
	}
	for _, invalid := range []int{0, -1, 7, 255, 1000} {
		if isValidMembershipType(invalid) {
			t.Errorf("isValidMembershipType(%d) = true, want false", invalid)
		}
	}
}

func TestParseMembershipParams(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x/:membershipType/:membershipId", func(c *gin.Context) {
		mt, mid, ok := parseMembershipParams(c)
		if !ok {
			return // parseMembershipParams already wrote the error response
		}
		c.JSON(http.StatusOK, gin.H{"type": mt, "id": mid})
	})

	cases := []struct {
		name string
		path string
		want int
	}{
		{"valid", "/x/3/4611686018467260757", http.StatusOK},
		{"bad type", "/x/99/4611686018467260757", http.StatusBadRequest},
		{"non-numeric type", "/x/steam/4611686018467260757", http.StatusBadRequest},
		{"bad id", "/x/3/short", http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Errorf("GET %s = %d, want %d", tc.path, w.Code, tc.want)
			}
		})
	}
}

func TestHandleBungieError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string // substring of the JSON "code" field
	}{
		{"privacy", &bungie.BungieError{ErrorCode: 5}, http.StatusForbidden, "PRIVACY_RESTRICTION"},
		{"not found", &bungie.BungieError{ErrorCode: 7}, http.StatusNotFound, "ACCOUNT_NOT_FOUND"},
		{"throttled", &bungie.BungieError{ErrorCode: 36, ThrottleSeconds: 12}, http.StatusTooManyRequests, "RATE_LIMITED"},
		{"other bungie error", &bungie.BungieError{ErrorCode: 99}, http.StatusBadGateway, "BUNGIE_ERROR"},
		{"manifest not ready", fmt.Errorf("wrap: %w", collections.ErrManifestNotReady), http.StatusServiceUnavailable, "MANIFEST_NOT_READY"},
		{"plain error", fmt.Errorf("boom"), http.StatusInternalServerError, "INTERNAL_ERROR"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			handleBungieError(c, tc.err)
			if w.Code != tc.wantCode {
				t.Errorf("status = %d, want %d", w.Code, tc.wantCode)
			}
			if !strings.Contains(w.Body.String(), tc.wantBody) {
				t.Errorf("body %s missing code %q", w.Body.String(), tc.wantBody)
			}
		})
	}
}

// fullResponse spreads one complete result across three wire fields. Every item
// is described once, an owned item also names itself in collectedHashes, and an
// item on sale also names its vendor — and nothing else appears in either.
func TestFullResponse_TranscribesItemsIntoTheWireFields(t *testing.T) {
	resp := fullResponse(collections.Full{
		Items: []collections.CollectionItem{
			{Item: collections.DestinyItem{ItemHash: "100", Name: "Fatebringer"}, Collected: true},
			{Item: collections.DestinyItem{ItemHash: "200", Name: "The Palindrome"}, AvailableFrom: "Banshee-44"},
			{Item: collections.DestinyItem{ItemHash: "300", Name: "Hawkmoon"}, Collected: true, AvailableFrom: "Xûr"},
		},
		Totals:    collections.CategorySummary{Weapons: collections.CategoryCount{Total: 3, Collected: 2}},
		FetchedAt: time.Date(2026, 7, 18, 18, 0, 0, 0, time.UTC),
	})

	if len(resp.Items) != 3 || resp.Items["200"].Name != "The Palindrome" {
		t.Errorf("items = %+v, want all three keyed by item hash", resp.Items)
	}
	if want := []string{"100", "300"}; !slices.Equal(resp.CollectedHashes, want) {
		t.Errorf("collectedHashes = %v, want %v", resp.CollectedHashes, want)
	}
	if len(resp.AvailableNow) != 2 || resp.AvailableNow["200"] != "Banshee-44" || resp.AvailableNow["300"] != "Xûr" {
		t.Errorf("availableNow = %v, want only the two items on sale", resp.AvailableNow)
	}
	if resp.Summary.Weapons.Total != 3 || !resp.FetchedAt.Equal(time.Date(2026, 7, 18, 18, 0, 0, 0, time.UTC)) {
		t.Errorf("summary/fetchedAt must pass through: %+v %v", resp.Summary, resp.FetchedAt)
	}
}

// collectedHashes inherits the ordered item slice's order rather than sorting
// anything at the boundary, so the wire stays byte-identical across requests.
func TestFullResponse_CollectedHashesFollowItemOrder(t *testing.T) {
	resp := fullResponse(collections.Full{Items: []collections.CollectionItem{
		{Item: collections.DestinyItem{ItemHash: "100"}, Collected: true},
		{Item: collections.DestinyItem{ItemHash: "101"}},
		{Item: collections.DestinyItem{ItemHash: "300"}, Collected: true},
		{Item: collections.DestinyItem{ItemHash: "9000"}, Collected: true},
	}})

	if want := []string{"100", "300", "9000"}; !slices.Equal(resp.CollectedHashes, want) {
		t.Errorf("collectedHashes = %v, want %v", resp.CollectedHashes, want)
	}
}

// A collection with nothing owned and nothing on sale omits those fields
// entirely rather than serializing empty containers, which is what the wire has
// always done and what the frontend's optional fields expect.
func TestFullResponse_OmitsEmptyItemDerivedFields(t *testing.T) {
	resp := fullResponse(collections.Full{Items: []collections.CollectionItem{
		{Item: collections.DestinyItem{ItemHash: "100"}},
	}})

	blob, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var wire map[string]any
	if err := json.Unmarshal(blob, &wire); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	for _, field := range []string{"collectedHashes", "availableNow"} {
		if _, present := wire[field]; present {
			t.Errorf("empty %q must be omitted: %s", field, blob)
		}
	}
	if _, present := wire["items"]; !present {
		t.Errorf("items must still be present: %s", blob)
	}
}

// A Destiny membership is the pair. Comparing only the id let a caller name a
// platform their token never authenticated against, and the route would go on
// to resolve a Bungie token and key a cache entry on that unvouched-for pair.
func TestOwnershipCheck_ComparesTheMembershipPair(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const authenticatedID = "4611686018467260757"

	cases := []struct {
		name           string
		routeType      int
		routeID        string
		wantAuthorized bool
	}{
		{"same pair", 3, authenticatedID, true},
		{"same id, different platform", 2, authenticatedID, false},
		{"same platform, different id", 3, "4611686018467260999", false},
		{"neither matches", 1, "4611686018467260999", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Set("membership_id", authenticatedID)
			c.Set("membership_type", 3)

			got := ownershipCheck(c, tc.routeType, tc.routeID)

			if got != tc.wantAuthorized {
				t.Fatalf("ownershipCheck = %v, want %v", got, tc.wantAuthorized)
			}
			if !tc.wantAuthorized {
				if w.Code != http.StatusForbidden {
					t.Errorf("status = %d, want 403", w.Code)
				}
				if !strings.Contains(w.Body.String(), "FORBIDDEN") {
					t.Errorf("body = %s, want the FORBIDDEN code", w.Body.String())
				}
				// The refusal must abort, not merely write. Its header is
				// already flushed, so a call site that forgot to return would
				// otherwise append the caller's data to this 403's body.
				if !c.IsAborted() {
					t.Error("a refused request must be aborted")
				}
			}
		})
	}
}

// The wire shape is contract. This pins both responses byte for byte, so a
// field rename, a dropped omitempty, or a reordered struct has to be a
// deliberate edit here rather than something that quietly reaches the frontend.
//
// The summary shape is what the deleted MembershipCollections.Lightweight()
// used to produce: counted nodes with no item names anywhere, and no
// item-derived keys at all.
func TestCollectionsResponse_WireShapeIsPinned(t *testing.T) {
	fetchedAt := time.Date(2026, 7, 18, 18, 0, 0, 0, time.UTC)
	tree := []collections.CollectionNode{{
		Hash: "10", Name: "Weapons", Icon: "/i/w.png", Collected: 1, Total: 2,
		Children: []collections.CollectionNode{{
			Hash: "11", Name: "Hand Cannons", Icon: "/i/hc.png", Collected: 1, Total: 2,
			Items: []string{"100", "101"},
		}},
	}}
	totals := collections.CategorySummary{Weapons: collections.CategoryCount{Total: 2, Collected: 1}}

	t.Run("summary", func(t *testing.T) {
		// A summary tree names no items, which is what strips the node key.
		counted := []collections.CollectionNode{{
			Hash: "10", Name: "Weapons", Icon: "/i/w.png", Collected: 1, Total: 2,
			Children: []collections.CollectionNode{{
				Hash: "11", Name: "Hand Cannons", Icon: "/i/hc.png", Collected: 1, Total: 2,
			}},
		}}
		got := marshal(t, collectionsResponse{Tree: counted, Summary: totals, FetchedAt: fetchedAt})
		want := `{"tree":[{"hash":"10","name":"Weapons","icon":"/i/w.png","collected":1,"total":2,` +
			`"children":[{"hash":"11","name":"Hand Cannons","icon":"/i/hc.png","collected":1,"total":2}]}],` +
			`"summary":{"weapons":{"total":2,"collected":1},"armor":{"total":0,"collected":0},` +
			`"exotics":{"total":0,"collected":0},"cosmetics":{"total":0,"collected":0}},` +
			`"fetchedAt":"2026-07-18T18:00:00Z"}`
		if got != want {
			t.Errorf("summary wire shape changed:\n got %s\nwant %s", got, want)
		}
	})

	t.Run("full", func(t *testing.T) {
		got := marshal(t, fullResponse(collections.Full{
			Tree: tree,
			Items: []collections.CollectionItem{
				{Item: collections.DestinyItem{ItemHash: "100", Name: "Fatebringer", ItemType: "Hand Cannon", TierType: 5, Rarity: "Legendary"}, Collected: true},
				{Item: collections.DestinyItem{ItemHash: "101", Name: "The Palindrome", ItemType: "Hand Cannon", TierType: 5, Rarity: "Legendary"}, AvailableFrom: "Banshee-44"},
			},
			Totals:    totals,
			FetchedAt: fetchedAt,
		}))
		want := `{"tree":[{"hash":"10","name":"Weapons","icon":"/i/w.png","collected":1,"total":2,` +
			`"children":[{"hash":"11","name":"Hand Cannons","icon":"/i/hc.png","collected":1,"total":2,"items":["100","101"]}]}],` +
			`"items":{` +
			`"100":{"itemHash":"100","name":"Fatebringer","description":"","icon":"","itemType":"Hand Cannon","tierType":5,"rarity":"Legendary","farmOnly":false,"acquisitionSources":null,"isExotic":false},` +
			`"101":{"itemHash":"101","name":"The Palindrome","description":"","icon":"","itemType":"Hand Cannon","tierType":5,"rarity":"Legendary","farmOnly":false,"acquisitionSources":null,"isExotic":false}},` +
			`"collectedHashes":["100"],"availableNow":{"101":"Banshee-44"},` +
			`"summary":{"weapons":{"total":2,"collected":1},"armor":{"total":0,"collected":0},` +
			`"exotics":{"total":0,"collected":0},"cosmetics":{"total":0,"collected":0}},` +
			`"fetchedAt":"2026-07-18T18:00:00Z"}`
		if got != want {
			t.Errorf("full wire shape changed:\n got %s\nwant %s", got, want)
		}
	})
}

func marshal(t *testing.T, v any) string {
	t.Helper()
	blob, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	return string(blob)
}
