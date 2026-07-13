// Command fake-bungie serves GuardianTracker's hermetic browser-test Bungie API.
// It binds to loopback only and generates its tiny SQLite manifest at startup.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"guardian-tracker/api-service/internal/e2efixture"
)

func main() {
	address := envOr("FAKE_BUNGIE_ADDR", e2efixture.DefaultListenAddress)
	if !loopbackAddress(address) {
		slog.Error("fake Bungie must bind to a loopback address", "address", address)
		os.Exit(1)
	}

	fixture, err := e2efixture.NewServer(e2efixture.ServerOptions{
		RedirectURI: envOr("FAKE_BUNGIE_REDIRECT_URI", e2efixture.DefaultRedirectURI),
		ClientID:    envOr("FAKE_BUNGIE_CLIENT_ID", e2efixture.DefaultClientID),
	})
	if err != nil {
		slog.Error("fake Bungie fixture generation failed", "error", err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr:              address,
		Handler:           fixture.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		slog.Info("fake Bungie listening", "address", address, "manifest_version", e2efixture.ManifestVersion)
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			slog.Error("fake Bungie server stopped", "error", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("fake Bungie shutdown failed", "error", err)
			os.Exit(1)
		}
	}
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func loopbackAddress(address string) bool {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return false
	}
	host = strings.Trim(host, "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
