package observability

import (
	"bytes"
	"errors"
	"log/slog"
	"regexp"
	"strings"
	"testing"
)

func TestNewLogger_Configuration(t *testing.T) {
	var textOutput bytes.Buffer
	textLogger, err := NewLogger("info", "text", &textOutput)
	if err != nil {
		t.Fatal(err)
	}
	textLogger.Debug("hidden")
	textLogger.Info("visible", "answer", 42)
	if got := textOutput.String(); strings.Contains(got, "hidden") || !strings.Contains(got, "visible") {
		t.Fatalf("unexpected text output: %s", got)
	}

	var jsonOutput bytes.Buffer
	jsonLogger, err := NewLogger("debug", "json", &jsonOutput)
	if err != nil {
		t.Fatal(err)
	}
	jsonLogger.Debug("visible-debug")
	if got := jsonOutput.String(); !strings.Contains(got, `"level":"DEBUG"`) {
		t.Fatalf("unexpected JSON output: %s", got)
	}

	for _, tc := range []struct{ level, format string }{{"trace", "text"}, {"info", "xml"}} {
		if _, err := NewLogger(tc.level, tc.format, &bytes.Buffer{}); err == nil {
			t.Fatalf("NewLogger(%q, %q) unexpectedly succeeded", tc.level, tc.format)
		}
	}
}

func TestContextLogger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	if got := Logger(WithLogger(t.Context(), logger)); got != logger {
		t.Fatal("Logger did not return the context logger")
	}
}

func TestPseudonym_IsStable24Hex(t *testing.T) {
	first := Pseudonym("guardian-123")
	if second := Pseudonym("guardian-123"); second != first {
		t.Fatalf("pseudonym changed: %q vs %q", first, second)
	}
	if first == Pseudonym("guardian-124") {
		t.Fatal("different identifiers produced the same pseudonym")
	}
	if !regexp.MustCompile(`^[0-9a-f]{24}$`).MatchString(first) {
		t.Fatalf("pseudonym = %q, want 24 lowercase hex digits", first)
	}
}

func TestErr_DoesNotExposeErrorText(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	logger.Error("operation failed", Err(errors.New("https://example.test/path?q=secret-error-marker")))
	if strings.Contains(output.String(), "secret-error-marker") || strings.Contains(output.String(), "https://") {
		t.Fatalf("error text leaked: %s", output.String())
	}
}
