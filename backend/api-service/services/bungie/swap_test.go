package bungie

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// orderRecorder captures the exact interleaving of swap callbacks, so a test can
// assert the contract's ordering rather than just that each part ran.
type orderRecorder struct {
	events *[]string
	name   string
}

func (o orderRecorder) CloseForSwap() { *o.events = append(*o.events, o.name+":close") }
func (o orderRecorder) Reopen() error { *o.events = append(*o.events, o.name+":reopen"); return nil }
func (o orderRecorder) OnVersionChanged(v string) error {
	*o.events = append(*o.events, o.name+":changed("+v+")")
	return nil
}

func manifestZip(t *testing.T, body string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("world_en.content")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// downloadHarness serves a manifest whose version and payload the test picks.
func downloadHarness(t *testing.T, version, body string) (*Client, func()) {
	t.Helper()
	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(manifestZip(t, body))
	}))
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"ErrorCode":1,"Response":{"version":%q,
			"mobileWorldContentPaths":{"en":"/world.content"}}}`, version)
	}))
	client := NewClient("k", api.URL, 100, 100)
	client.SetCDNBaseURL(cdn.URL)
	return client, func() { cdn.Close(); api.Close() }
}

// The contract in one assertion: every participant closes before the rename,
// every participant reopens after it, and only then are observers told. An
// observer that fired before the reopen would query a closed manifest.
func TestSwap_ClosesReopensThenNotifies(t *testing.T) {
	client, cleanup := downloadHarness(t, "v-new", "NEW-BYTES")
	defer cleanup()

	dbPath := filepath.Join(t.TempDir(), "manifest.sqlite")
	ms := NewManifestService(client, dbPath, time.Hour)

	var events []string
	provider := orderRecorder{events: &events, name: "provider"}
	search := orderRecorder{events: &events, name: "search"}
	records := orderRecorder{events: &events, name: "records"}

	ms.RegisterParticipant(provider)
	ms.RegisterParticipant(search)
	ms.RegisterObserver(records)
	ms.RegisterObserver(search)

	if err := ms.Download(context.Background()); err != nil {
		t.Fatalf("Download: %v", err)
	}

	want := []string{
		"provider:close", "search:close",
		"provider:reopen", "search:reopen",
		"records:changed(v-new)", "search:changed(v-new)",
	}
	if len(events) != len(want) {
		t.Fatalf("events = %v, want %v", events, want)
	}
	for i := range want {
		if events[i] != want[i] {
			t.Fatalf("events = %v, want %v", events, want)
		}
	}
}

// The rollback path. A failed rename leaves the OLD manifest in place, so
// participants must be reopened against it — but observers must NOT fire: the
// version did not change, and invalidating caches or rebuilding indexes here
// would throw away correct state for nothing.
func TestSwap_RollbackReopensButDoesNotNotify(t *testing.T) {
	dir := t.TempDir()
	// A directory at dbPath makes os.Rename fail on every platform, which is the
	// portable way to exercise the rollback branch.
	dbPath := filepath.Join(dir, "manifest.sqlite")
	if err := os.Mkdir(dbPath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dbPath, "occupant"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	ms := NewManifestService(NewClient("k", "http://unused", 100, 100), dbPath, time.Hour)

	var events []string
	p := orderRecorder{events: &events, name: "provider"}
	o := orderRecorder{events: &events, name: "records"}
	ms.RegisterParticipant(p)
	ms.RegisterObserver(o)

	zipPath := filepath.Join(dir, "manifest.zip")
	if err := os.WriteFile(zipPath, manifestZip(t, "NEW-BYTES"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := ms.extractManifest(zipPath); err == nil {
		t.Fatal("extractManifest should fail when the rename cannot succeed")
	}

	want := []string{"provider:close", "provider:reopen"}
	if len(events) != len(want) || events[0] != want[0] || events[1] != want[1] {
		t.Fatalf("rollback events = %v, want exactly %v", events, want)
	}
	for _, e := range events {
		if len(e) > 8 && e[len(e)-1] == ')' {
			t.Errorf("observer was notified on the rollback path: %q", e)
		}
	}
}
