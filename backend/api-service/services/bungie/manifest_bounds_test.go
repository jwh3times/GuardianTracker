package bungie

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"guardian-tracker/api-service/internal/boundedio"
)

func TestRejectedManifestKeepsInstalledVersionAndHooks(t *testing.T) {
	for _, kind := range []string{"archive size", "extracted size", "checksum", "false declared size"} {
		t.Run(kind, func(t *testing.T) {
			payload := manifestZip(t, "NEW-BYTES")
			if kind == "checksum" || kind == "false declared size" {
				central := bytes.Index(payload, []byte{'P', 'K', 1, 2})
				if central < 0 {
					t.Fatal("missing central directory")
				}
				if kind == "checksum" {
					payload[central+16] ^= 1
				} else {
					binary.LittleEndian.PutUint32(payload[central+24:], 1)
				}
			}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/world.content" {
					_, _ = w.Write(payload)
					return
				}
				_, _ = fmt.Fprint(w, `{"ErrorCode":1,"Response":{"version":"new","mobileWorldContentPaths":{"en":"/world.content"}}}`)
			}))
			defer srv.Close()
			client := NewClient("test", srv.URL, 100, 100)
			client.SetCDNBaseURL(srv.URL)
			dbPath := filepath.Join(t.TempDir(), "manifest.sqlite")
			if err := os.WriteFile(dbPath, []byte("OLD-BYTES"), 0600); err != nil {
				t.Fatal(err)
			}
			ms := NewManifestService(client, dbPath, time.Hour)
			ms.currentVersion = "old"
			if err := os.WriteFile(ms.versionPath, []byte("old"), 0600); err != nil {
				t.Fatal(err)
			}
			var events []string
			ms.RegisterParticipant(orderRecorder{events: &events, name: "provider"})
			ms.RegisterObserver(orderRecorder{events: &events, name: "index"})
			if kind == "archive size" {
				client.archiveLimit = int64(len(payload) - 1)
			}
			if kind == "extracted size" || kind == "false declared size" {
				ms.extractedLimit = 4
			}
			err := ms.Download(context.Background())
			if err == nil {
				t.Fatal("invalid manifest installed")
			}
			if (kind == "archive size" || kind == "extracted size") && !errors.Is(err, boundedio.ErrTooLarge) {
				t.Fatalf("error=%v", err)
			}
			if kind == "checksum" && !errors.Is(err, zip.ErrChecksum) {
				t.Fatalf("error=%v, want checksum", err)
			}
			if data, err := os.ReadFile(dbPath); err != nil || string(data) != "OLD-BYTES" {
				t.Fatalf("installed data=%q error=%v", data, err)
			}
			if data, err := os.ReadFile(ms.versionPath); err != nil || string(data) != "old" {
				t.Fatalf("version file=%q error=%v", data, err)
			}
			if ms.Version() != "old" || !ms.lastCheck.IsZero() || len(events) != 0 {
				t.Fatalf("version=%q events=%v lastCheck=%v", ms.Version(), events, ms.lastCheck)
			}
			for _, path := range []string{dbPath + ".tmp", dbPath + ".zip.tmp"} {
				if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("staging file remains: %s", path)
				}
			}
		})
	}
}

func TestManifestExactExtractionLimitSucceeds(t *testing.T) {
	client, cleanup := downloadHarness(t, "new", "NEW-BYTES")
	defer cleanup()
	ms := NewManifestService(client, filepath.Join(t.TempDir(), "manifest.sqlite"), time.Hour)
	ms.extractedLimit = int64(len("NEW-BYTES"))
	if err := ms.Download(context.Background()); err != nil {
		t.Fatal(err)
	}
	if ms.Version() != "new" {
		t.Fatalf("version=%q", ms.Version())
	}
}
