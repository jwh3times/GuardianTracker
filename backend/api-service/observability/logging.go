// Package observability configures structured application logging and carries
// request-scoped loggers through context.Context.
package observability

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"strconv"
)

type loggerContextKey struct{}

// NewLogger builds the configured standard-library slog logger.
func NewLogger(levelName, format string, output io.Writer) (*slog.Logger, error) {
	var level slog.Level
	switch levelName {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		return nil, fmt.Errorf("invalid log level %q", levelName)
	}

	options := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	switch format {
	case "text":
		handler = slog.NewTextHandler(output, options)
	case "json":
		handler = slog.NewJSONHandler(output, options)
	default:
		return nil, fmt.Errorf("invalid log format %q", format)
	}
	return slog.New(handler), nil
}

// WithLogger attaches a logger to a context for downstream handlers/services.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey{}, logger)
}

// Logger returns the request-scoped logger, falling back to slog.Default for
// background work and contexts created outside HTTP middleware.
func Logger(ctx context.Context) *slog.Logger {
	if ctx != nil {
		if logger, ok := ctx.Value(loggerContextKey{}).(*slog.Logger); ok && logger != nil {
			return logger
		}
	}
	return slog.Default()
}

// Pseudonym returns the first 12 bytes of SHA-256 as 24 lowercase hex digits.
// It is deterministic across processes without exposing the source identifier.
func Pseudonym(identifier string) string {
	digest := sha256.Sum256([]byte(identifier))
	return hex.EncodeToString(digest[:12])
}

// ID records a privacy-safe reference for a string identifier.
func ID(key, identifier string) slog.Attr {
	return slog.String(key+"_ref", Pseudonym(identifier))
}

// IntID records a privacy-safe reference for a numeric identifier.
func IntID(key string, identifier int64) slog.Attr {
	return ID(key, strconv.FormatInt(identifier, 10))
}

// Err records only an error's concrete type. Error strings from HTTP, file, and
// database libraries can embed URLs, paths, identifiers, or response bodies.
func Err(err error) slog.Attr {
	if err == nil {
		return slog.String("error", "<nil>")
	}
	return slog.String("error", fmt.Sprintf("%T", err))
}
