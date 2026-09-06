package db

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgproto3"
)

// This driver-level regression exercises the documented DATABASE_URL TLS
// settings through pgx's real PostgreSQL SSL negotiation and TLS handshake.
// It neither changes application policy nor requires a database, Docker, or
// network access beyond a generated loopback fixture.
func TestPostgresVerifyFullAuthenticatesServer(t *testing.T) {
	// Keep pgx's environment/service defaults from loading an operator's config.
	// Explicit URL values below also override default client-certificate paths.
	for _, name := range []string{
		"PGHOST", "PGPORT", "PGDATABASE", "PGUSER", "PGPASSWORD", "PGPASSFILE",
		"PGAPPNAME", "PGCONNECT_TIMEOUT", "PGSSLMODE", "PGSSLKEY", "PGSSLCERT",
		"PGSSLSNI", "PGSSLROOTCERT", "PGSSLPASSWORD", "PGSSLNEGOTIATION",
		"PGTARGETSESSIONATTRS", "PGSERVICE", "PGSERVICEFILE", "PGTZ", "PGOPTIONS",
		"PGMINPROTOCOLVERSION", "PGMAXPROTOCOLVERSION", "PGCHANNELBINDING", "PGREQUIREAUTH",
	} {
		t.Setenv(name, "")
	}

	trusted := newPostgresTestCA(t)
	untrusted := newPostgresTestCA(t)
	rootPath := filepath.Join(t.TempDir(), "root.crt")
	if err := os.WriteFile(rootPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: trusted.certificate.Raw}), 0600); err != nil {
		t.Fatal(err)
	}
	// pgx opens passfile even when a password is explicitly supplied. Give it
	// an empty fixture file instead of the operator's default ~/.pgpass.
	passfile := filepath.Join(t.TempDir(), "pgpass")
	if err := os.WriteFile(passfile, nil, 0600); err != nil {
		t.Fatal(err)
	}
	const hostname = "database.guardian.test"
	for _, tc := range []struct {
		name      string
		signer    postgresTestCA
		hostname  string
		rejection string
	}{
		{"trusted matching hostname", trusted, hostname, ""},
		{"untrusted signing CA", untrusted, hostname, "authority"},
		{"trusted certificate for wrong hostname", trusted, "other.guardian.test", "hostname"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			address, completed := startPostgresTLSFixture(t, tc.signer.serverCertificate(t, tc.hostname))
			_, port, err := net.SplitHostPort(address)
			if err != nil {
				t.Fatal(err)
			}
			dsn := url.URL{Scheme: "postgres", User: url.UserPassword("fixture", "synthetic-password"), Host: net.JoinHostPort(hostname, port), Path: "/fixture"}
			dsn.RawQuery = url.Values{
				"sslmode": {"verify-full"}, "sslrootcert": {rootPath},
				"sslcert": {""}, "sslkey": {""}, "passfile": {passfile},
				"sslnegotiation": {"postgres"}, "connect_timeout": {"5"},
			}.Encode()
			config, err := pgx.ParseConfig(dsn.String())
			if err != nil {
				t.Fatal(err)
			}
			// Preserve the parsed TLS ServerName/roots/fallbacks. Only fixture DNS is
			// redirected, so pgx still verifies the hostname supplied in the URL.
			config.LookupFunc = func(_ context.Context, host string) ([]string, error) {
				if host != hostname {
					return nil, fmt.Errorf("unexpected fixture lookup: %s", host)
				}
				return []string{"127.0.0.1"}, nil
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			conn, err := pgx.ConnectConfig(ctx, config)
			if tc.rejection == "" {
				if err != nil {
					t.Fatalf("trusted matching server rejected: %v", err)
				}
				if err := conn.Ping(ctx); err != nil {
					conn.Close(ctx)
					t.Fatalf("TLS PostgreSQL ping failed: %v", err)
				}
				if err := conn.Close(ctx); err != nil {
					t.Fatal(err)
				}
			} else {
				if conn != nil {
					conn.Close(ctx)
				}
				switch tc.rejection {
				case "authority":
					var failure x509.UnknownAuthorityError
					if !errors.As(err, &failure) {
						t.Fatalf("want certificate authority rejection, got %v", err)
					}
				case "hostname":
					var failure x509.HostnameError
					if !errors.As(err, &failure) {
						t.Fatalf("want certificate hostname rejection, got %v", err)
					}
				}
			}
			select {
			case result := <-completed:
				if tc.rejection == "" {
					if result.err != nil || !result.startup || !result.authenticated || !result.ping {
						t.Fatalf("incomplete trusted connection: %+v", result)
					}
				} else if result.startup || result.authenticated || result.ping {
					t.Fatalf("unverified peer received PostgreSQL traffic: %+v", result)
				}
			case <-ctx.Done():
				t.Fatal("TLS fixture did not finish")
			}
		})
	}
}

type postgresTestCA struct {
	certificate *x509.Certificate
	key         *ecdsa.PrivateKey
}

func newPostgresTestCA(t *testing.T) postgresTestCA {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "Ephemeral PostgreSQL test CA"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return postgresTestCA{certificate, key}
}

func (ca postgresTestCA) serverCertificate(t *testing.T, hostname string) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(2), DNSNames: []string{hostname},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, ca.certificate, &key.PublicKey, ca.key)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{der, ca.certificate.Raw}, PrivateKey: key}
}

type postgresTLSResult struct {
	startup, authenticated, ping bool
	err                          error
}

func startPostgresTLSFixture(t *testing.T, certificate tls.Certificate) (string, <-chan postgresTLSResult) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	completed := make(chan postgresTLSResult, 1)
	go func() {
		result := postgresTLSResult{}
		defer func() { completed <- result }()
		conn, err := listener.Accept()
		if err != nil {
			result.err = err
			return
		}
		defer conn.Close()
		if err = conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
			result.err = err
			return
		}
		plain := pgproto3.NewBackend(conn, conn)
		request, err := plain.ReceiveStartupMessage()
		if err != nil {
			result.err = err
			return
		}
		if _, ok := request.(*pgproto3.SSLRequest); !ok {
			result.err = fmt.Errorf("expected SSLRequest, got %T", request)
			return
		}
		if _, err = conn.Write([]byte{'S'}); err != nil {
			result.err = err
			return
		}
		secure := tls.Server(conn, &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12})
		if err = secure.Handshake(); err != nil {
			result.err = err
			return
		}
		backend := pgproto3.NewBackend(secure, secure)
		request, err = backend.ReceiveStartupMessage()
		if err != nil {
			result.err = err
			return
		}
		if _, ok := request.(*pgproto3.StartupMessage); !ok {
			result.err = fmt.Errorf("expected StartupMessage, got %T", request)
			return
		}
		result.startup = true
		backend.Send(&pgproto3.AuthenticationCleartextPassword{})
		if err = backend.Flush(); err != nil {
			result.err = err
			return
		}
		message, err := backend.Receive()
		if err != nil {
			result.err = err
			return
		}
		password, ok := message.(*pgproto3.PasswordMessage)
		if !ok || password.Password != "synthetic-password" {
			result.err = fmt.Errorf("expected synthetic PasswordMessage, got %T", message)
			return
		}
		result.authenticated = true
		backend.Send(&pgproto3.AuthenticationOk{})
		backend.Send(&pgproto3.ParameterStatus{Name: "server_version", Value: "18.0"})
		backend.Send(&pgproto3.ReadyForQuery{TxStatus: 'I'})
		if err = backend.Flush(); err != nil {
			result.err = err
			return
		}
		message, err = backend.Receive()
		if err != nil {
			result.err = err
			return
		}
		query, ok := message.(*pgproto3.Query)
		if !ok || query.String != "-- ping" {
			result.err = fmt.Errorf("expected ping query, got %T", message)
			return
		}
		result.ping = true
		backend.Send(&pgproto3.EmptyQueryResponse{})
		backend.Send(&pgproto3.ReadyForQuery{TxStatus: 'I'})
		if err = backend.Flush(); err != nil {
			result.err = err
			return
		}
		message, err = backend.Receive()
		if err != nil {
			result.err = err
			return
		}
		if _, ok := message.(*pgproto3.Terminate); !ok {
			result.err = fmt.Errorf("expected Terminate, got %T", message)
		}
	}()
	return listener.Addr().String(), completed
}
