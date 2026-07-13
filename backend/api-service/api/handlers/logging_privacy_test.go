package handlers

import (
	"bytes"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	"guardian-tracker/api-service/observability"

	"github.com/gin-gonic/gin"
)

func TestGetBungieToken_LogDoesNotExposeMembershipID(t *testing.T) {
	const membershipID = "4611686018467260757"
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/api/test", nil)
	req = req.WithContext(observability.WithLogger(req.Context(), logger))
	c.Request = req

	if _, ok := getBungieToken(c, membershipID, newTokenStore(t)); ok {
		t.Fatal("empty token store unexpectedly returned a token")
	}
	logs := output.String()
	if strings.Contains(logs, membershipID) {
		t.Fatalf("membership ID leaked: %s", logs)
	}
	if !strings.Contains(logs, observability.Pseudonym(membershipID)) {
		t.Fatalf("pseudonym missing: %s", logs)
	}
}
