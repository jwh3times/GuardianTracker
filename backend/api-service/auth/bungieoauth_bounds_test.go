package auth

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"guardian-tracker/api-service/internal/boundedio"
)

func TestOAuthResponseBounds(t *testing.T) {
	const tokenBody = `{"access_token":"token","expires_in":3600}`
	const memberBody = `{"Response":{"destinyMemberships":[{"membershipType":3,"membershipId":"id","displayName":"test"}]}}`
	for _, endpoint := range []string{"code", "refresh", "membership"} {
		for _, kind := range []string{"exact", "overflow", "chunked", "gzip", "trailing junk"} {
			t.Run(endpoint+"/"+kind, func(t *testing.T) {
				body := tokenBody
				if endpoint == "membership" {
					body = memberBody
				}
				limit := int64(len(body))
				if kind == "trailing junk" {
					body += " {}"
					limit = int64(len(body))
				}
				if kind == "overflow" || kind == "chunked" {
					limit--
				}
				if kind == "gzip" {
					body += strings.Repeat(" ", 4096)
					limit = 256
				}
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if kind == "chunked" {
						w.(http.Flusher).Flush()
					}
					if kind == "gzip" {
						var compressed bytes.Buffer
						zw := gzip.NewWriter(&compressed)
						_, _ = io.WriteString(zw, body)
						_ = zw.Close()
						w.Header().Set("Content-Encoding", "gzip")
						_, _ = w.Write(compressed.Bytes())
						return
					}
					_, _ = io.WriteString(w, body)
				}))
				defer srv.Close()
				o := newBungieOAuth("client", srv.URL)
				o.jsonLimit = limit
				var err error
				switch endpoint {
				case "code":
					_, err = o.exchangeCode(context.Background(), "code")
				case "refresh":
					_, err = o.refreshTokens(context.Background(), "refresh")
				case "membership":
					_, err = o.primaryDestinyMembership(context.Background(), srv.URL, "key", "token")
				}
				if kind == "exact" {
					if err != nil {
						t.Fatal(err)
					}
					return
				}
				if err == nil {
					t.Fatal("invalid response accepted")
				}
				if kind != "trailing junk" && !errors.Is(err, boundedio.ErrTooLarge) {
					t.Fatalf("error=%v", err)
				}
				if strings.Contains(err.Error(), "access_token") || strings.Contains(err.Error(), "destinyMemberships") {
					t.Fatal("response body leaked into error")
				}
			})
		}
	}
}

type oauthUnreadBody struct {
	read   bool
	closed bool
}

func (b *oauthUnreadBody) Read([]byte) (int, error) {
	b.read = true
	return 0, errors.New("must not read rejected grant")
}
func (b *oauthUnreadBody) Close() error { b.closed = true; return nil }

type oauthResponseTransport func(*http.Request) (*http.Response, error)

func (f oauthResponseTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestOAuthRejectedGrantDoesNotReadBody(t *testing.T) {
	for _, status := range []int{400, 401, 500} {
		body := &oauthUnreadBody{}
		o := newBungieOAuth("client", "http://unused")
		o.client.Transport = oauthResponseTransport(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: status, Body: body}, nil
		})
		_, err := o.refreshTokens(context.Background(), "refresh")
		var grantErr *oauthGrantError
		if !errors.As(err, &grantErr) || grantErr.StatusCode != status {
			t.Fatalf("status=%d error=%v", status, err)
		}
		if body.read || !body.closed {
			t.Fatalf("read=%v closed=%v", body.read, body.closed)
		}
	}
}
