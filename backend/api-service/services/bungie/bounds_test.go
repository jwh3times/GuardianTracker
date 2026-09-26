package bungie

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"guardian-tracker/api-service/internal/boundedio"
)

func TestEveryJSONEndpointBoundsResponse(t *testing.T) {
	endpoints := map[string]func(*Client) error{
		"manifest":   func(c *Client) error { _, e := c.GetManifest(context.Background()); return e },
		"profile":    func(c *Client) error { _, e := c.GetProfile(context.Background(), 3, "id", "", nil); return e },
		"characters": func(c *Client) error { _, e := c.GetCharacters(context.Background(), 3, "id", ""); return e },
		"history": func(c *Client) error {
			_, e := c.GetActivityHistory(context.Background(), 3, "id", "char", "", 0, 5)
			return e
		},
		"milestones": func(c *Client) error { _, e := c.GetPublicMilestones(context.Background()); return e },
		"character vendors": func(c *Client) error {
			_, e := c.GetCharacterVendors(context.Background(), 3, "id", "char", "")
			return e
		},
		"public vendors": func(c *Client) error { _, e := c.GetPublicVendors(context.Background()); return e },
		"records":        func(c *Client) error { _, e := c.GetRecords(context.Background(), 3, "id", ""); return e },
		"settings":       func(c *Client) error { _, e := c.GetCommonSettings(context.Background()); return e },
	}
	const body = `{"ErrorCode":1,"Response":{}}`
	for name, call := range endpoints {
		t.Run(name, func(t *testing.T) {
			for _, encoding := range []string{"plain", "chunked", "gzip"} {
				t.Run(encoding, func(t *testing.T) {
					srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if encoding == "gzip" {
							w.Header().Set("Content-Encoding", "gzip")
							zw := gzip.NewWriter(w)
							_, _ = io.WriteString(zw, body)
							_ = zw.Close()
						} else {
							if encoding == "chunked" {
								w.(http.Flusher).Flush()
							}
							_, _ = io.WriteString(w, body)
						}
					}))
					defer srv.Close()
					c := NewClient("test", srv.URL, 1000, 1000)
					c.jsonLimit = int64(len(body))
					if err := call(c); err != nil {
						t.Fatalf("exact limit: %v", err)
					}
					c.jsonLimit--
					if err := call(c); !errors.Is(err, boundedio.ErrTooLarge) {
						t.Fatalf("overflow: %v", err)
					}
				})
			}
		})
	}
}

type trackedBody struct {
	io.Reader
	closed   bool
	closeErr error
}

func (b *trackedBody) Close() error { b.closed = true; return b.closeErr }

type responseTransport func(*http.Request) (*http.Response, error)

func (f responseTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestJSONBoundsValidateWholeBodyAndClose(t *testing.T) {
	for _, body := range []string{`{"ErrorCode":1} {}`, `{"ErrorCode":1}junk`, `{malformed`, strings.Repeat("x", 33)} {
		b := &trackedBody{Reader: strings.NewReader(body)}
		if value, err := parseResponse[ManifestResponse](&http.Response{Body: b}, 32); err == nil || value != nil {
			t.Fatalf("accepted %q", body)
		}
		if !b.closed {
			t.Fatal("body not closed")
		}
	}
}

func TestJSONLimitAppliesAfterGzipExpansion(t *testing.T) {
	var compressed bytes.Buffer
	zw := gzip.NewWriter(&compressed)
	_, _ = io.WriteString(zw, `{"ErrorCode":1,"Response":{}}`+strings.Repeat(" ", 4096))
	_ = zw.Close()
	if compressed.Len() >= 128 {
		t.Fatal("fixture must fit compressed limit")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		_, _ = w.Write(compressed.Bytes())
	}))
	defer srv.Close()
	c := NewClient("test", srv.URL, 100, 100)
	c.jsonLimit = 128
	if _, err := c.GetManifest(context.Background()); !errors.Is(err, boundedio.ErrTooLarge) {
		t.Fatalf("error=%v", err)
	}
}

func TestDownloadChunkedOverflow(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.(http.Flusher).Flush()
		_, _ = io.WriteString(w, "abcde")
	}))
	defer srv.Close()
	c := NewClient("test", "unused", 100, 100)
	c.archiveLimit = 4
	dest := filepath.Join(t.TempDir(), "manifest.zip.tmp")
	if err := c.DownloadFileToPath(context.Background(), srv.URL, dest); !errors.Is(err, boundedio.ErrTooLarge) {
		t.Fatalf("error=%v", err)
	}
	if calls != 1 {
		t.Fatalf("requests=%d, want 1", calls)
	}
	if _, err := os.Stat(dest); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("partial archive remains")
	}
}

func TestDownloadBoundsAndCleanup(t *testing.T) {
	for _, size := range []int{4, 5} {
		t.Run(strings.Repeat("x", size), func(t *testing.T) {
			calls := 0
			body := &trackedBody{Reader: strings.NewReader(strings.Repeat("x", size))}
			c := NewClient("test", "unused", 100, 100)
			c.archiveLimit = 4
			c.downloadClient.Transport = responseTransport(func(*http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: 200, Body: body, ContentLength: -1}, nil
			})
			dest := filepath.Join(t.TempDir(), "manifest.zip.tmp")
			err := c.DownloadFileToPath(context.Background(), "http://unused/file", dest)
			if size == 4 {
				if err != nil {
					t.Fatal(err)
				}
				data, err := os.ReadFile(dest)
				if err != nil || string(data) != "xxxx" {
					t.Fatalf("data=%q error=%v", data, err)
				}
			} else {
				if !errors.Is(err, boundedio.ErrTooLarge) {
					t.Fatalf("error=%v", err)
				}
				if _, err := os.Stat(dest); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("partial archive remains")
				}
			}
			if !body.closed || calls != 1 {
				t.Fatalf("closed=%v requests=%d", body.closed, calls)
			}
		})
	}
}

func TestDownloadBodyCloseFailureRemovesFile(t *testing.T) {
	failure := errors.New("close failed")
	c := NewClient("test", "unused", 100, 100)
	c.downloadClient.Transport = responseTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: &trackedBody{Reader: strings.NewReader("data"), closeErr: failure}}, nil
	})
	dest := filepath.Join(t.TempDir(), "manifest.zip.tmp")
	if err := c.downloadToPathOnce(context.Background(), "http://unused/file", dest); !errors.Is(err, failure) {
		t.Fatalf("error=%v", err)
	}
	if _, err := os.Stat(dest); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("archive remains after close failure")
	}
}

func TestDownloadDoesNotTrustContentLength(t *testing.T) {
	c := NewClient("test", "unused", 100, 100)
	c.archiveLimit = 4
	body := &trackedBody{Reader: strings.NewReader("abcde")}
	c.downloadClient.Transport = responseTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, ContentLength: 1, Body: body}, nil
	})
	dest := filepath.Join(t.TempDir(), "manifest.zip.tmp")
	if err := c.DownloadFileToPath(context.Background(), "http://unused/file", dest); !errors.Is(err, boundedio.ErrTooLarge) {
		t.Fatalf("error=%v", err)
	}
	if _, err := os.Stat(dest); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("oversized archive remains")
	}
	if !body.closed {
		t.Fatal("body not closed")
	}
}

func TestDownloadLimitAppliesAfterGzipExpansion(t *testing.T) {
	var compressed bytes.Buffer
	zw := gzip.NewWriter(&compressed)
	_, _ = io.WriteString(zw, strings.Repeat("x", 4096))
	_ = zw.Close()
	if compressed.Len() >= 128 {
		t.Fatal("fixture must fit compressed limit")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		_, _ = w.Write(compressed.Bytes())
	}))
	defer srv.Close()
	c := NewClient("test", "unused", 100, 100)
	c.archiveLimit = 128
	dest := filepath.Join(t.TempDir(), "manifest.zip.tmp")
	if err := c.DownloadFileToPath(context.Background(), srv.URL, dest); !errors.Is(err, boundedio.ErrTooLarge) {
		t.Fatalf("error=%v", err)
	}
	if _, err := os.Stat(dest); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("oversized decoded archive remains")
	}
}

type failedCloseWriter struct {
	bytes.Buffer
	closed bool
	err    error
}

func (w *failedCloseWriter) Close() error { w.closed = true; return w.err }

func TestCopyAndClosePropagatesOutputFailure(t *testing.T) {
	failure := errors.New("output close failed")
	for _, body := range []string{"abcd", "abcde"} {
		out := &failedCloseWriter{err: failure}
		err := copyAndClose(out, strings.NewReader(body), 4)
		if !out.closed || !errors.Is(err, failure) {
			t.Fatalf("closed=%v error=%v", out.closed, err)
		}
		if len(body) > 4 && !errors.Is(err, boundedio.ErrTooLarge) {
			t.Fatalf("lost size failure: %v", err)
		}
	}
}
