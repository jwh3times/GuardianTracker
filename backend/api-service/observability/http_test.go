package observability

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func jsonRecords(t *testing.T, output string) []map[string]any {
	t.Helper()
	var records []map[string]any
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		var record map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			t.Fatalf("decode log record: %v\n%s", err, scanner.Text())
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return records
}

func findRecord(t *testing.T, records []map[string]any, message string) map[string]any {
	t.Helper()
	for _, record := range records {
		if record["msg"] == message {
			return record
		}
	}
	t.Fatalf("missing log record %q: %#v", message, records)
	return nil
}

func TestHTTPMiddleware_RequestIDRouteTemplateAndPrivacy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var output bytes.Buffer
	logger, err := NewLogger("debug", "json", &output)
	if err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	r.Use(HTTPMiddleware(logger))
	r.POST("/items/:itemHash", func(c *gin.Context) {
		Logger(c.Request.Context()).InfoContext(c.Request.Context(), "handler event")
		c.String(http.StatusCreated, "created")
	})

	req := httptest.NewRequest(http.MethodPost, "/items/123?q=secret-query-marker", strings.NewReader("secret-body-marker"))
	req.Header.Set(RequestIDHeader, "attacker-request-id")
	req.Header.Set("Authorization", "Bearer secret-auth-marker")
	req.Header.Set("User-Agent", "secret-agent-marker")
	req.RemoteAddr = "203.0.113.50:4567"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	requestID := w.Header().Get(RequestIDHeader)
	if _, err := uuid.Parse(requestID); err != nil {
		t.Fatalf("X-Request-ID = %q: %v", requestID, err)
	}
	if requestID == "attacker-request-id" {
		t.Fatal("middleware trusted a client-supplied request ID")
	}

	records := jsonRecords(t, output.String())
	inside := findRecord(t, records, "handler event")
	access := findRecord(t, records, "http request")
	if inside["request_id"] != requestID || access["request_id"] != requestID {
		t.Fatalf("request ID was not propagated: inside=%v access=%v", inside, access)
	}
	if access["route"] != "/items/:itemHash" || access["method"] != http.MethodPost || access["status"] != float64(http.StatusCreated) {
		t.Fatalf("unexpected access record: %#v", access)
	}
	if access["level"] != "INFO" {
		t.Fatalf("access level = %v, want INFO", access["level"])
	}

	allLogs := output.String()
	for _, forbidden := range []string{
		"/items/123", "secret-query-marker", "secret-body-marker", "secret-auth-marker",
		"secret-agent-marker", "203.0.113.50", "attacker-request-id",
	} {
		if strings.Contains(allLogs, forbidden) {
			t.Errorf("logs exposed %q: %s", forbidden, allLogs)
		}
	}
}

func TestAccessLevel(t *testing.T) {
	for _, tc := range []struct {
		route  string
		status int
		want   string
	}{
		{route: "/health", status: 200, want: "DEBUG"},
		{route: "/ready", status: 200, want: "DEBUG"},
		{route: "/api/items", status: 200, want: "INFO"},
		{route: "/api/items", status: 404, want: "WARN"},
		{route: "/api/items", status: 500, want: "ERROR"},
	} {
		if got := accessLevel(tc.route, tc.status).String(); got != tc.want {
			t.Errorf("accessLevel(%q, %d) = %s, want %s", tc.route, tc.status, got, tc.want)
		}
	}
}

func TestHTTPMiddleware_RecoversPanicWithoutLoggingValue(t *testing.T) {
	var output bytes.Buffer
	logger, err := NewLogger("debug", "json", &output)
	if err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	r.Use(HTTPMiddleware(logger))
	r.GET("/panic", func(*gin.Context) { panic("secret-panic-marker") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/panic", nil))
	if w.Code != http.StatusInternalServerError || !strings.Contains(w.Body.String(), "INTERNAL_ERROR") {
		t.Fatalf("panic response = %d %s", w.Code, w.Body.String())
	}
	if strings.Contains(output.String(), "secret-panic-marker") {
		t.Fatalf("panic value leaked: %s", output.String())
	}
	records := jsonRecords(t, output.String())
	findRecord(t, records, "panic recovered")
	access := findRecord(t, records, "http request")
	if access["status"] != float64(http.StatusInternalServerError) || access["level"] != "ERROR" {
		t.Fatalf("panic access record = %#v", access)
	}
}
