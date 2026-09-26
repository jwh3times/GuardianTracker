package boundedio

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestBounds(t *testing.T) {
	for _, size := range []int{0, 3, 4, 5, 100} {
		for _, mode := range []string{"read", "copy"} {
			t.Run(mode+"/"+strings.Repeat("x", size), func(t *testing.T) {
				r := strings.NewReader(strings.Repeat("x", size))
				var err error
				if mode == "read" {
					var body []byte
					body, err = ReadAll(r, 4)
					if size > 4 && body != nil {
						t.Fatal("oversized read returned partial data")
					}
					if size <= 4 && len(body) != size {
						t.Fatalf("read %d bytes, want %d", len(body), size)
					}
				} else {
					var dst bytes.Buffer
					err = Copy(&dst, r, 4)
					if dst.Len() != min(size, 4) {
						t.Fatalf("wrote %d bytes, want %d", dst.Len(), min(size, 4))
					}
				}
				if size > 4 && !errors.Is(err, ErrTooLarge) || size <= 4 && err != nil {
					t.Fatalf("size %d: error = %v", size, err)
				}
				if size-r.Len() > 5 {
					t.Fatal("consumed beyond one overflow byte")
				}
			})
		}
	}
}

type failedReader struct{ err error }

func (r failedReader) Read([]byte) (int, error) { return 0, r.err }

type failedWriter struct{ err error }

func (w failedWriter) Write([]byte) (int, error) { return 0, w.err }

func TestReadAndWriteErrors(t *testing.T) {
	failure := errors.New("read failure")
	for _, prefix := range []string{"", "abc", "abcd"} {
		r := io.MultiReader(strings.NewReader(prefix), failedReader{failure})
		if body, err := ReadAll(r, 4); !errors.Is(err, failure) || body != nil {
			t.Fatalf("prefix %q: body=%q error=%v", prefix, body, err)
		}
		r = io.MultiReader(strings.NewReader(prefix), failedReader{failure})
		if err := Copy(io.Discard, r, 4); !errors.Is(err, failure) {
			t.Fatalf("copy prefix %q: %v", prefix, err)
		}
	}
	if err := Copy(failedWriter{failure}, strings.NewReader("abc"), 4); !errors.Is(err, failure) {
		t.Fatalf("write error = %v", err)
	}
	for _, limit := range []int64{0, -1} {
		if _, err := ReadAll(strings.NewReader("a"), limit); err == nil {
			t.Fatal("invalid read limit accepted")
		}
		if err := Copy(io.Discard, strings.NewReader("a"), limit); err == nil {
			t.Fatal("invalid copy limit accepted")
		}
	}
}
