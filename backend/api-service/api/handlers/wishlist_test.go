package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"guardian-tracker/api-service/services/items"
	"guardian-tracker/api-service/services/wishlist"

	"github.com/gin-gonic/gin"
)

// --- stub wish list service ---
//
// The handler owns no wish list rules any more, so these tests drive the seam
// rather than a fake database: what reaches the service, and what each typed
// outcome renders as. The rules themselves, and every part of completing an
// entry, are tested in services/wishlist.

type stubWishlist struct {
	entries []wishlist.Entry
	entry   wishlist.Entry
	bulk    wishlist.BulkResult
	err     error

	gotMembership   wishlist.Membership
	gotAdd          wishlist.AddCommand
	gotUpdateID     wishlist.EntryID
	gotUpdate       wishlist.UpdateCommand
	gotRemoveID     wishlist.EntryID
	gotBulkIDs      []wishlist.EntryID
	gotBulkPriority wishlist.Priority
	gotAction       string
}

func (s *stubWishlist) List(_ context.Context, m wishlist.Membership) ([]wishlist.Entry, error) {
	s.gotMembership = m
	return s.entries, s.err
}

func (s *stubWishlist) Add(_ context.Context, m wishlist.Membership, cmd wishlist.AddCommand) (wishlist.Entry, error) {
	s.gotMembership, s.gotAdd = m, cmd
	return s.entry, s.err
}

func (s *stubWishlist) Update(_ context.Context, m wishlist.Membership, id wishlist.EntryID, patch wishlist.UpdateCommand) (wishlist.Entry, error) {
	s.gotMembership, s.gotUpdateID, s.gotUpdate = m, id, patch
	return s.entry, s.err
}

func (s *stubWishlist) Remove(_ context.Context, m wishlist.Membership, id wishlist.EntryID) error {
	s.gotMembership, s.gotRemoveID = m, id
	return s.err
}

func (s *stubWishlist) DeleteMany(_ context.Context, m wishlist.Membership, ids []wishlist.EntryID) (wishlist.BulkResult, error) {
	s.gotMembership, s.gotBulkIDs, s.gotAction = m, ids, "delete"
	return s.bulk, s.err
}

func (s *stubWishlist) SetPriorityMany(_ context.Context, m wishlist.Membership, ids []wishlist.EntryID, priority wishlist.Priority) (wishlist.BulkResult, error) {
	s.gotMembership, s.gotBulkIDs, s.gotBulkPriority, s.gotAction = m, ids, priority, "set_priority"
	return s.bulk, s.err
}

// --- router setup helper ---

func newTestRouter(h *WishlistHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("membership_id", "test-member-123")
		c.Set("membership_type", 3)
		c.Next()
	})
	r.GET("/api/wishlist", h.GetWishlist)
	r.POST("/api/wishlist", h.AddToWishlist)
	r.PUT("/api/wishlist/:id", h.UpdateWishlistItem)
	r.DELETE("/api/wishlist/:id", h.RemoveFromWishlist)
	r.POST("/api/wishlist/bulk", h.BulkUpdate)
	return r
}

func send(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func savedEntry(id wishlist.EntryID, hash uint32, priority wishlist.Priority) wishlist.Entry {
	return wishlist.Entry{ID: id, ItemHash: hash, Priority: priority, CreatedAt: time.Now()}
}

// --- tests ---

func TestGetWishlist_SerializesStoredEntries(t *testing.T) {
	entries := &stubWishlist{entries: []wishlist.Entry{
		{ID: 1, ItemHash: 1234, Priority: wishlist.PriorityHigh, Notes: "nice roll", CreatedAt: time.Now()},
	}}
	w := send(newTestRouter(NewWishlistHandler(entries)), http.MethodGet, "/api/wishlist", "")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	var resp []wishlistResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp) != 1 {
		t.Fatalf("items = %d, want 1", len(resp))
	}
	if resp[0].ID != "1" || resp[0].ItemHash != 1234 || resp[0].Priority != "HIGH" || resp[0].Notes != "nice roll" {
		t.Errorf("item = %+v", resp[0])
	}
}

// An empty wish list is an empty JSON array, never null: the frontend maps over
// it directly.
func TestGetWishlist_EmptyListSerializesAsAnArray(t *testing.T) {
	w := send(newTestRouter(NewWishlistHandler(&stubWishlist{})), http.MethodGet, "/api/wishlist", "")

	if body := strings.TrimSpace(w.Body.String()); body != "[]" {
		t.Errorf("body = %s, want []", body)
	}
}

// The wish list is never addressed by client-supplied identity: the whole
// membership pair comes from the JWT the middleware validated. The platform
// half matters because availability is resolved per platform, so dropping it
// would silently resolve vendors for the wrong one.
func TestGetWishlist_UsesTheJWTMembership(t *testing.T) {
	entries := &stubWishlist{}
	send(newTestRouter(NewWishlistHandler(entries)), http.MethodGet, "/api/wishlist", "")

	want := wishlist.Membership{MembershipType: 3, MembershipID: "test-member-123"}
	if entries.gotMembership != want {
		t.Errorf("membership = %+v, want the pair the JWT carried %+v", entries.gotMembership, want)
	}
}

// Every field of a complete entry reaches the wire under the name it has always
// had, including the independent availability pair.
func TestGetWishlist_SerializesEveryFieldOfACompleteEntry(t *testing.T) {
	entries := &stubWishlist{entries: []wishlist.Entry{
		{
			ID:            9,
			ItemHash:      100,
			Priority:      wishlist.PriorityUrgent,
			Notes:         "god roll",
			CreatedAt:     time.Date(2026, 7, 18, 18, 0, 0, 0, time.UTC),
			Item:          wishlist.KnownItem(items.AcquisitionFacts{Name: "Fatebringer", ItemType: "Hand Cannon", Rarity: "Legendary", Icon: "/i/f.png"}),
			AvailableFrom: "Xûr",
		},
		{
			ID:        10,
			ItemHash:  999,
			Priority:  wishlist.PriorityLow,
			CreatedAt: time.Date(2026, 7, 18, 18, 0, 0, 0, time.UTC),
			Item:      wishlist.UnknownItemTombstone(),
		},
	}}

	w := send(newTestRouter(NewWishlistHandler(entries)), http.MethodGet, "/api/wishlist", "")

	var resp []wishlistResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp) != 2 {
		t.Fatalf("items = %d, want 2", len(resp))
	}

	known := resp[0]
	if known.ID != "9" || known.Name != "Fatebringer" || known.ItemType != "Hand Cannon" || known.Rarity != "Legendary" || known.Icon != "/i/f.png" {
		t.Errorf("known item = %+v", known)
	}
	if known.Priority != "URGENT" || known.Notes != "god roll" || known.DateAdded != "2026-07-18T18:00:00Z" {
		t.Errorf("stored fields = %+v", known)
	}
	if !known.AvailableNow || known.AvailableFrom != "Xûr" {
		t.Errorf("availability = %v/%q, want true/Xûr", known.AvailableNow, known.AvailableFrom)
	}

	// A tombstone renders its stand-in, and nothing is claimed to be on sale.
	tombstone := resp[1]
	if tombstone.Name != "Unknown Item" || tombstone.ItemType != "Item" || tombstone.Rarity != "Common" || tombstone.Icon != "" {
		t.Errorf("tombstone = %+v", tombstone)
	}
	if tombstone.AvailableNow || tombstone.AvailableFrom != "" {
		t.Errorf("availability = %v/%q, want false/empty", tombstone.AvailableNow, tombstone.AvailableFrom)
	}
	if tombstone.AcquisitionSources == nil {
		t.Error("acquisitionSources must serialize as [] rather than null")
	}
}

// Every typed outcome the wish list can produce has exactly one wire meaning.
// Flattening any two of them would put domain meaning back in the handler.
func TestWishlistErrors_MapToTheExistingWire(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		status   int
		contains string
	}{
		{"unavailable", wishlist.ErrUnavailable, http.StatusServiceUnavailable, "DB_UNAVAILABLE"},
		{"not found", wishlist.ErrNotFound, http.StatusNotFound, "wishlist item not found"},
		{"duplicate", wishlist.ErrDuplicate, http.StatusConflict, "item already in wishlist"},
		{"items unavailable", wishlist.ErrItemsUnavailable, http.StatusServiceUnavailable, "MANIFEST_NOT_READY"},
		{"unknown item", wishlist.ErrUnknownItem, http.StatusBadRequest, "unknown item hash"},
		{"invalid priority", wishlist.ErrInvalidPriority, http.StatusBadRequest, "priority must be LOW, MEDIUM, HIGH, or URGENT"},
		{"notes too long", wishlist.ErrNotesTooLong, http.StatusBadRequest, "notes must be 500 characters or fewer"},
		{"unexpected", errors.New("connection reset"), http.StatusInternalServerError, "INTERNAL_ERROR"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newTestRouter(NewWishlistHandler(&stubWishlist{err: tc.err}))
			w := send(r, http.MethodPost, "/api/wishlist", `{"itemHash":1234}`)

			if w.Code != tc.status {
				t.Fatalf("status = %d, want %d: %s", w.Code, tc.status, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), tc.contains) {
				t.Errorf("body = %s, want it to contain %q", w.Body.String(), tc.contains)
			}
		})
	}
}

func TestGetWishlist_UnavailablePersistenceReturns503(t *testing.T) {
	r := newTestRouter(NewWishlistHandler(&stubWishlist{err: wishlist.ErrUnavailable}))
	w := send(r, http.MethodGet, "/api/wishlist", "")

	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), "DB_UNAVAILABLE") {
		t.Errorf("status = %d body = %s, want 503 DB_UNAVAILABLE", w.Code, w.Body.String())
	}
}

func TestAddToWishlist_PassesTheRequestThroughAsACommand(t *testing.T) {
	entries := &stubWishlist{entry: savedEntry(9, 1234, wishlist.PriorityUrgent)}
	r := newTestRouter(NewWishlistHandler(entries))

	w := send(r, http.MethodPost, "/api/wishlist", `{"itemHash":1234,"priority":"URGENT","notes":"soon"}`)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", w.Code, w.Body.String())
	}
	want := wishlist.AddCommand{ItemHash: 1234, Priority: wishlist.PriorityUrgent, Notes: "soon"}
	if entries.gotAdd != want {
		t.Errorf("command = %+v, want %+v", entries.gotAdd, want)
	}
}

// An omitted priority reaches the service empty. The default is the wish
// list's to choose, and the handler must not quietly pick one first.
func TestAddToWishlist_OmittedPriorityStaysUnsetAtTheSeam(t *testing.T) {
	entries := &stubWishlist{entry: savedEntry(9, 1234, wishlist.PriorityMedium)}
	r := newTestRouter(NewWishlistHandler(entries))

	send(r, http.MethodPost, "/api/wishlist", `{"itemHash":1234}`)

	if entries.gotAdd.Priority != "" {
		t.Errorf("priority = %q, want it left to the wish list", entries.gotAdd.Priority)
	}
}

func TestAddToWishlist_MissingItemHashIsRejectedBeforeTheSeam(t *testing.T) {
	entries := &stubWishlist{}
	r := newTestRouter(NewWishlistHandler(entries))

	w := send(r, http.MethodPost, "/api/wishlist", `{"priority":"HIGH"}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	if entries.gotAdd.ItemHash != 0 {
		t.Error("a request with no item hash reached the wish list")
	}
}

func TestUpdateWishlistItem_SendsAPartialPatch(t *testing.T) {
	entries := &stubWishlist{entry: savedEntry(7, 1234, wishlist.PriorityLow)}
	r := newTestRouter(NewWishlistHandler(entries))

	w := send(r, http.MethodPut, "/api/wishlist/7", `{"notes":"changed"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	if entries.gotUpdateID != 7 {
		t.Errorf("entry id = %d, want 7", entries.gotUpdateID)
	}
	if entries.gotUpdate.Priority != nil {
		t.Error("an absent priority was sent as a change")
	}
	if entries.gotUpdate.Notes == nil || *entries.gotUpdate.Notes != "changed" {
		t.Errorf("notes patch = %v, want \"changed\"", entries.gotUpdate.Notes)
	}
}

func TestUpdateWishlistItem_NotFoundReturns404(t *testing.T) {
	r := newTestRouter(NewWishlistHandler(&stubWishlist{err: wishlist.ErrNotFound}))

	if w := send(r, http.MethodPut, "/api/wishlist/7", `{"notes":"x"}`); w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestWishlistItemID_MustBeNumeric(t *testing.T) {
	entries := &stubWishlist{}
	r := newTestRouter(NewWishlistHandler(entries))

	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		w := send(r, method, "/api/wishlist/abc", `{}`)
		if w.Code != http.StatusBadRequest {
			t.Errorf("%s status = %d, want 400", method, w.Code)
		}
	}
	if entries.gotUpdateID != 0 || entries.gotRemoveID != 0 {
		t.Error("a malformed id reached the wish list")
	}
}

func TestRemoveFromWishlist_SuccessReturns204(t *testing.T) {
	entries := &stubWishlist{}
	r := newTestRouter(NewWishlistHandler(entries))

	w := send(r, http.MethodDelete, "/api/wishlist/5", "")

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", w.Code)
	}
	if entries.gotRemoveID != 5 {
		t.Errorf("removed id = %d, want 5", entries.gotRemoveID)
	}
}

func TestRemoveFromWishlist_NotFoundReturns404(t *testing.T) {
	r := newTestRouter(NewWishlistHandler(&stubWishlist{err: wishlist.ErrNotFound}))

	if w := send(r, http.MethodDelete, "/api/wishlist/5", ""); w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestBulkUpdate_RoutesEachActionToItsCommand(t *testing.T) {
	t.Run("delete", func(t *testing.T) {
		entries := &stubWishlist{bulk: wishlist.BulkResult{Updated: 2, Skipped: 1}}
		r := newTestRouter(NewWishlistHandler(entries))

		w := send(r, http.MethodPost, "/api/wishlist/bulk", `{"action":"delete","ids":[1,2,999]}`)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
		}
		if entries.gotAction != "delete" || len(entries.gotBulkIDs) != 3 {
			t.Errorf("action = %q with ids %v", entries.gotAction, entries.gotBulkIDs)
		}
		var body struct{ Updated, Skipped int }
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body.Updated != 2 || body.Skipped != 1 {
			t.Errorf("body = %+v, want {2 1}", body)
		}
	})

	t.Run("set priority", func(t *testing.T) {
		entries := &stubWishlist{bulk: wishlist.BulkResult{Updated: 1}}
		r := newTestRouter(NewWishlistHandler(entries))

		w := send(r, http.MethodPost, "/api/wishlist/bulk", `{"action":"set_priority","ids":[4],"priority":"HIGH"}`)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
		}
		if entries.gotAction != "set_priority" || entries.gotBulkPriority != wishlist.PriorityHigh {
			t.Errorf("action = %q priority = %q", entries.gotAction, entries.gotBulkPriority)
		}
	})
}

// The action string is the handler's own vocabulary — it names which command to
// call, so an unknown one never reaches the wish list.
func TestBulkUpdate_UnknownActionIsRejectedBeforeTheSeam(t *testing.T) {
	entries := &stubWishlist{}
	r := newTestRouter(NewWishlistHandler(entries))

	w := send(r, http.MethodPost, "/api/wishlist/bulk", `{"action":"explode","ids":[1]}`)

	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "action must be") {
		t.Fatalf("status = %d body = %s, want 400", w.Code, w.Body.String())
	}
	if entries.gotAction != "" {
		t.Errorf("action %q reached the wish list", entries.gotAction)
	}
}

func TestBulkUpdate_ValidationRefusalsRenderAsTheExistingMessages(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		contains string
	}{
		{"empty", wishlist.ErrNoEntries, "ids must be a non-empty list"},
		{"too many", wishlist.ErrTooManyEntries, "at most 100 ids per request"},
		{"invalid priority", wishlist.ErrInvalidPriority, "priority must be LOW, MEDIUM, HIGH, or URGENT"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newTestRouter(NewWishlistHandler(&stubWishlist{err: tc.err}))
			w := send(r, http.MethodPost, "/api/wishlist/bulk", `{"action":"delete","ids":[1]}`)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400: %s", w.Code, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), tc.contains) {
				t.Errorf("body = %s, want %q", w.Body.String(), tc.contains)
			}
		})
	}
}

func TestBulkUpdate_UnavailablePersistenceReturns503(t *testing.T) {
	r := newTestRouter(NewWishlistHandler(&stubWishlist{err: wishlist.ErrUnavailable}))
	w := send(r, http.MethodPost, "/api/wishlist/bulk", `{"action":"delete","ids":[1]}`)

	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), "DB_UNAVAILABLE") {
		t.Errorf("status = %d body = %s, want 503 DB_UNAVAILABLE", w.Code, w.Body.String())
	}
}

// --- completion (still handler-owned until the complete wish list service) ---
