package handlers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"guardian-tracker/api-service/observability"
	"guardian-tracker/api-service/services/digest"
	"guardian-tracker/api-service/services/items"
	"guardian-tracker/api-service/services/preferences"

	"github.com/gin-gonic/gin"
)

// abandonedRequest builds a request whose own context is already canceled, the
// state net/http leaves it in once the client disconnects or aborts, with a
// logger that records everything at debug and above into the returned buffer.
func abandonedRequest(t *testing.T, method, target string, canceled bool) (*http.Request, *bytes.Buffer) {
	t.Helper()
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx, cancel := context.WithCancel(observability.WithLogger(context.Background(), logger))
	if canceled {
		cancel()
	} else {
		t.Cleanup(cancel)
	}
	return httptest.NewRequest(method, target, nil).WithContext(ctx), &logs
}

func TestAbandonedByClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name     string
		canceled bool
		err      error
		want     bool
	}{
		{"client canceled, service reports the cancellation", true, fmt.Errorf("read: %w", context.Canceled), true},
		// A real failure that coincides with a disconnect is still a failure.
		{"client canceled, unrelated failure", true, errors.New("boom"), false},
		// Work canceled by something other than the caller, such as a detached
		// shared load, is not the client's doing.
		{"client present, internal cancellation", false, context.Canceled, false},
		{"client canceled, server deadline", true, context.DeadlineExceeded, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = abandonedRequest(t, http.MethodGet, "/x", tc.canceled)

			if got := abandonedByClient(c, tc.err); got != tc.want {
				t.Fatalf("abandonedByClient = %v, want %v", got, tc.want)
			}
			if tc.want {
				if w.Code != observability.StatusClientClosedRequest || !c.IsAborted() {
					t.Errorf("status = %d aborted = %v, want %d aborted", w.Code, c.IsAborted(), observability.StatusClientClosedRequest)
				}
			} else if c.Writer.Written() {
				t.Errorf("wrote status %d for a request it did not handle", w.Code)
			}
		})
	}
}

// Every shared failure mapper must record a client-abandoned request as a
// cancellation — no internal-error body and no error-level log line.
func TestErrorMappersRecordClientCancellation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for name, mapper := range map[string]func(*gin.Context, error){
		"handleBungieError":      handleBungieError,
		"handleRollTargetError":  func(c *gin.Context, err error) { handleRollTargetError(c, err, "matches failed") },
		"handlePreferencesError": func(c *gin.Context, err error) { handlePreferencesError(c, err, "preferences failed") },
		"handleWishlistError":    func(c *gin.Context, err error) { handleWishlistError(c, err, "wishlist failed") },
		"HandleStoreError":       func(c *gin.Context, err error) { HandleStoreError(c, err, "store failed") },
	} {
		t.Run(name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			var logs *bytes.Buffer
			c.Request, logs = abandonedRequest(t, http.MethodGet, "/x", true)

			mapper(c, fmt.Errorf("service: %w", context.Canceled))

			assertRecordedCancellation(t, w, logs)
		})
	}
}

func TestWeeklyRecordsClientCancellation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewWeeklyHandler(&fakeWeeklyService{err: context.Canceled}, fakeWeeklyTokenStore{})
	router := gin.New()
	router.GET("/api/weekly/recommendations", func(c *gin.Context) {
		c.Set("membership_id", "member-1")
		c.Set("membership_type", 3)
		h.GetWeekly(c)
	})
	req, logs := abandonedRequest(t, http.MethodGet, "/api/weekly/recommendations", true)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assertRecordedCancellation(t, w, logs)
}

func TestPreferencesReadRecordsClientCancellation(t *testing.T) {
	repository := &preferencesRepositoryStub{getErr: context.Canceled}
	router := preferencesTestRouter(NewPreferencesHandler(preferences.NewService(repository)))
	req, logs := abandonedRequest(t, http.MethodGet, "/api/preferences", true)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assertRecordedCancellation(t, w, logs)
}

type canceledDigestRepository struct{ digest.Repository }

func (canceledDigestRepository) Get(context.Context, string) (digest.Snapshot, bool, error) {
	return digest.Snapshot{}, false, context.Canceled
}

type unusedDigestCollections struct{}

func (unusedDigestCollections) CollectedState(context.Context, int, string, string) (map[uint32]bool, int, time.Time, error) {
	panic("not reached: the state read fails first")
}

type unusedDigestItems struct{}

func (unusedDigestItems) Lookup(context.Context, []uint32) (map[uint32]items.AcquisitionFacts, error) {
	panic("not reached: the state read fails first")
}

func TestDigestRecordsClientCancellation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const membershipID = "4611686018467260757"
	tokens := newTokenStore(t)
	storeValidToken(tokens, membershipID)
	h := NewDigestHandler(digest.NewService(canceledDigestRepository{}, unusedDigestCollections{}, unusedDigestItems{}), tokens)
	router := gin.New()
	router.GET("/api/digest/:membershipType/:membershipId", func(c *gin.Context) {
		c.Set("membership_id", membershipID)
		c.Set("membership_type", 3)
		h.GetDigest(c)
	})
	req, logs := abandonedRequest(t, http.MethodGet, "/api/digest/3/"+membershipID, true)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assertRecordedCancellation(t, w, logs)
}

// End to end through the real Bungie client: a caller that leaves while the
// upstream read is in flight must surface as context.Canceled, not as a
// wrapped transport error the mappers cannot recognise.
func TestCharactersRecordsCancellationDuringUpstreamRead(t *testing.T) {
	gin.SetMode(gin.TestMode)
	arrived := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		close(arrived)
		<-r.Context().Done()
	}))
	defer upstream.Close()
	ts := newTokenStore(t)
	storeValidToken(ts, testUserID)
	h := charactersHandler(t, upstream.URL, ts)
	r := authedRouter(http.MethodGet, "/api/characters/:membershipType/:membershipId", testUserID, h.GetCharacters)

	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx, cancel := context.WithCancel(observability.WithLogger(context.Background(), logger))
	defer cancel()
	req := httptest.NewRequest(http.MethodGet, "/api/characters/3/"+testUserID, nil).WithContext(ctx)
	go func() {
		<-arrived
		cancel()
	}()
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assertRecordedCancellation(t, w, &logs)
}

func assertRecordedCancellation(t *testing.T, w *httptest.ResponseRecorder, logs *bytes.Buffer) {
	t.Helper()
	if w.Code != observability.StatusClientClosedRequest {
		t.Errorf("status = %d, want %d; body = %s", w.Code, observability.StatusClientClosedRequest, w.Body.String())
	}
	if w.Body.Len() != 0 {
		t.Errorf("wrote a body for a departed client: %s", w.Body.String())
	}
	if bytes.Contains(logs.Bytes(), []byte(`"level":"ERROR"`)) || bytes.Contains(logs.Bytes(), []byte(`"level":"WARN"`)) {
		t.Errorf("logged a client cancellation as a failure: %s", logs.String())
	}
}
