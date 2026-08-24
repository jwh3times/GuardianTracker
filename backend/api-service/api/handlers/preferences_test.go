package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"guardian-tracker/api-service/services/preferences"

	"github.com/gin-gonic/gin"
)

type preferencesRepositoryStub struct {
	getValues   preferences.Values
	getFound    bool
	getErr      error
	getCalls    int
	applyValues preferences.Values
	applyErr    error
	applyCalls  int
	applyPatch  preferences.Patch
}

func (s *preferencesRepositoryStub) Get(context.Context, string) (preferences.Values, bool, error) {
	s.getCalls++
	return s.getValues, s.getFound, s.getErr
}

func (s *preferencesRepositoryStub) Apply(_ context.Context, _ string, _ preferences.Values, patch preferences.Patch) (preferences.Values, error) {
	s.applyCalls++
	s.applyPatch = patch
	return s.applyValues, s.applyErr
}

func preferencesTestRouter(handler *PreferencesHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("membership_id", "membership-1")
		c.Next()
	})
	router.GET("/api/preferences", handler.GetPreferences)
	router.PUT("/api/preferences", handler.UpdatePreferences)
	return router
}

func TestPreferencesGetUnavailableReturnsDefaultsWithProvenance(t *testing.T) {
	repository := &preferencesRepositoryStub{getErr: preferences.ErrUnavailable}
	handler := NewPreferencesHandler(preferences.NewService(repository))
	router := preferencesTestRouter(handler)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/preferences", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", response.Code, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["cardStyle"] != "framed" || body["personalize"] != true || body["onboardedAt"] != nil {
		t.Errorf("values = %#v, want framed/true/null defaults", body)
	}
	if body["persisted"] != false {
		t.Errorf("persisted = %v, want false", body["persisted"])
	}
}

func TestPreferencesGetAuthoritativeValuesIncludesPersistedProvenance(t *testing.T) {
	stamp := time.Date(2026, time.July, 12, 15, 30, 0, 0, time.UTC)
	repository := &preferencesRepositoryStub{getFound: true, getValues: preferences.Values{
		CardStyle: preferences.CardStyleCompact, Personalize: false, OnboardedAt: &stamp,
	}}
	router := preferencesTestRouter(NewPreferencesHandler(preferences.NewService(repository)))
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/preferences", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", response.Code, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["cardStyle"] != "compact" || body["personalize"] != false {
		t.Errorf("values = %#v, want compact/false", body)
	}
	if body["onboardedAt"] != "2026-07-12T15:30:00Z" {
		t.Errorf("onboardedAt = %v, want server timestamp", body["onboardedAt"])
	}
	if body["persisted"] != true {
		t.Errorf("persisted = %v, want true", body["persisted"])
	}
}

func TestPreferencesUpdateAppliesOnlySuppliedFields(t *testing.T) {
	repository := &preferencesRepositoryStub{
		applyValues: preferences.Values{CardStyle: preferences.CardStyleCompact, Personalize: false},
	}
	handler := NewPreferencesHandler(preferences.NewService(repository))
	router := preferencesTestRouter(handler)

	request := httptest.NewRequest(http.MethodPut, "/api/preferences", strings.NewReader(`{"personalize":false}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", response.Code, response.Body.String())
	}
	if repository.getCalls != 0 {
		t.Fatalf("repository Get calls = %d, want 0", repository.getCalls)
	}
	if repository.applyCalls != 1 {
		t.Fatalf("repository Apply calls = %d, want 1", repository.applyCalls)
	}
	if repository.applyPatch.CardStyle != nil {
		t.Errorf("CardStyle presence = true, want false")
	}
	if repository.applyPatch.Personalize == nil || *repository.applyPatch.Personalize {
		t.Errorf("Personalize patch = %v, want present false", repository.applyPatch.Personalize)
	}
	if repository.applyPatch.OnboardingComplete != nil {
		t.Errorf("OnboardingComplete presence = true, want false")
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["cardStyle"] != "compact" || body["personalize"] != false {
		t.Errorf("values = %#v, want compact/false", body)
	}
	if _, ok := body["persisted"]; ok {
		t.Errorf("PUT response included GET-only persisted field: %#v", body)
	}
}

func TestPreferencesUpdatePreservesLegacyValidationErrors(t *testing.T) {
	for _, test := range []struct {
		name      string
		body      string
		wantError string
	}{
		{name: "card style", body: `{"cardStyle":"giant"}`, wantError: "cardStyle must be 'framed' or 'compact'"},
		{name: "onboarding reset", body: `{"onboardingComplete":false}`, wantError: "onboardingComplete can only be set to true"},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &preferencesRepositoryStub{}
			router := preferencesTestRouter(NewPreferencesHandler(preferences.NewService(repository)))
			request := httptest.NewRequest(http.MethodPut, "/api/preferences", strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body = %s", response.Code, response.Body.String())
			}
			var body map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body["error"] != test.wantError {
				t.Errorf("error = %v, want %q", body["error"], test.wantError)
			}
			if repository.applyCalls != 0 {
				t.Errorf("repository Apply calls = %d, want 0", repository.applyCalls)
			}
		})
	}
}

func TestPreferencesUpdateTreatsEmptyCardStyleAsOmitted(t *testing.T) {
	repository := &preferencesRepositoryStub{
		applyValues: preferences.Values{CardStyle: preferences.CardStyleCompact, Personalize: true},
	}
	router := preferencesTestRouter(NewPreferencesHandler(preferences.NewService(repository)))
	request := httptest.NewRequest(http.MethodPut, "/api/preferences", strings.NewReader(`{"cardStyle":""}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", response.Code, response.Body.String())
	}
	if repository.applyCalls != 1 {
		t.Fatalf("repository Apply calls = %d, want 1", repository.applyCalls)
	}
	if repository.applyPatch.CardStyle != nil {
		t.Errorf("CardStyle patch = %v, want omitted", repository.applyPatch.CardStyle)
	}
}

func TestPreferencesUpdateUnavailableReturnsStandard503(t *testing.T) {
	repository := &preferencesRepositoryStub{applyErr: preferences.ErrUnavailable}
	router := preferencesTestRouter(NewPreferencesHandler(preferences.NewService(repository)))
	request := httptest.NewRequest(http.MethodPut, "/api/preferences", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body = %s", response.Code, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["code"] != "DB_UNAVAILABLE" {
		t.Errorf("code = %v, want DB_UNAVAILABLE", body["code"])
	}
}

func TestPreferencesUnexpectedFailuresReturnStandard500(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		t.Run(method, func(t *testing.T) {
			repository := &preferencesRepositoryStub{getErr: errors.New("read detail"), applyErr: errors.New("write detail")}
			router := preferencesTestRouter(NewPreferencesHandler(preferences.NewService(repository)))
			var request *http.Request
			if method == http.MethodGet {
				request = httptest.NewRequest(method, "/api/preferences", nil)
			} else {
				request = httptest.NewRequest(method, "/api/preferences", strings.NewReader(`{}`))
				request.Header.Set("Content-Type", "application/json")
			}
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			if response.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d, want 500; body = %s", response.Code, response.Body.String())
			}
			if strings.Contains(response.Body.String(), "detail") {
				t.Errorf("response leaked persistence detail: %s", response.Body.String())
			}
			var body map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body["code"] != "INTERNAL_ERROR" {
				t.Errorf("code = %v, want INTERNAL_ERROR", body["code"])
			}
		})
	}
}
