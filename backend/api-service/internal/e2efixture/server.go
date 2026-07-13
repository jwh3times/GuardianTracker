package e2efixture

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
)

const (
	DefaultListenAddress = "127.0.0.1:8090"
	DefaultRedirectURI   = "http://127.0.0.1:5273/auth/callback"
	DefaultClientID      = "e2e-client"
	AuthorizationCode    = "e2e-auth-code"
	AccessToken          = "e2e-bungie-access-token"
	RefreshToken         = "e2e-bungie-refresh-token"
)

const (
	WeeklyNormal = "normal"
	WeeklyEmpty  = "empty"
)

// ServerOptions contains the fake values that must agree with the API E2E
// environment. Empty fields use the canonical localhost defaults above.
type ServerOptions struct {
	RedirectURI string
	ClientID    string
}

// Scenario is the mutable, deliberately tiny surface available to browser
// tests. It is never inferred from query strings on production-shaped routes.
type Scenario struct {
	Weekly string `json:"weekly"`
}

// Server implements the subset of Bungie's API used by GuardianTracker.
type Server struct {
	redirectURI string
	clientID    string
	manifestZip []byte

	mu       sync.RWMutex
	scenario Scenario
	refresh  atomic.Uint64
	handler  http.Handler
}

// NewServer builds the runtime SQLite archive before exposing any HTTP route.
func NewServer(opts ServerOptions) (*Server, error) {
	if opts.RedirectURI == "" {
		opts.RedirectURI = DefaultRedirectURI
	}
	if opts.ClientID == "" {
		opts.ClientID = DefaultClientID
	}
	if _, err := url.ParseRequestURI(opts.RedirectURI); err != nil {
		return nil, fmt.Errorf("invalid redirect URI: %w", err)
	}
	archive, err := GenerateManifestZip()
	if err != nil {
		return nil, err
	}
	s := &Server{
		redirectURI: opts.RedirectURI,
		clientID:    opts.ClientID,
		manifestZip: archive,
		scenario:    Scenario{Weekly: WeeklyNormal},
	}
	s.handler = s.routes()
	return s, nil
}

// Handler returns the hermetic fake-Bungie HTTP surface.
func (s *Server) Handler() http.Handler { return s.handler }

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /en/OAuth/Authorize", s.authorize)
	mux.HandleFunc("POST /platform/app/oauth/token/", s.token)
	mux.HandleFunc("GET /Platform/User/GetMembershipsForCurrentUser/", s.memberships)
	mux.HandleFunc("GET /Platform/Destiny2/Manifest/", s.manifest)
	mux.HandleFunc("GET /fixtures/manifest.zip", s.manifestArchive)

	profilePath := fmt.Sprintf("/Platform/Destiny2/%d/Profile/%s/", MembershipType, MembershipID)
	mux.HandleFunc("GET "+profilePath, s.profile)
	vendorPath := fmt.Sprintf("/Platform/Destiny2/%d/Profile/%s/Character/%s/Vendors/", MembershipType, MembershipID, CharacterID)
	mux.HandleFunc("GET "+vendorPath, s.characterVendors)
	mux.HandleFunc("GET /Platform/Destiny2/Vendors/", s.publicVendors)
	mux.HandleFunc("GET /Platform/Destiny2/Milestones/", s.milestones)
	mux.HandleFunc("GET /Platform/Settings/", s.settings)
	mux.HandleFunc("PUT /__e2e/scenario", s.setScenario)
	mux.HandleFunc("DELETE /__e2e/scenario", s.resetScenario)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		mux.ServeHTTP(w, r)
	})
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "manifestVersion": ManifestVersion})
}

func (s *Server) authorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("client_id") != s.clientID || q.Get("response_type") != "code" || q.Get("state") == "" || q.Get("redirect_uri") != s.redirectURI {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_oauth_request"})
		return
	}
	redirect, err := url.Parse(q.Get("redirect_uri"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_redirect_uri"})
		return
	}
	values := redirect.Query()
	values.Set("code", AuthorizationCode)
	values.Set("state", q.Get("state"))
	redirect.RawQuery = values.Encode()
	http.Redirect(w, r, redirect.String(), http.StatusFound)
}

func (s *Server) token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, "invalid_request")
		return
	}
	if r.PostForm.Get("client_id") != s.clientID {
		writeOAuthError(w, "invalid_client")
		return
	}

	switch r.PostForm.Get("grant_type") {
	case "authorization_code":
		if r.PostForm.Get("code") != AuthorizationCode {
			writeOAuthError(w, "invalid_grant")
			return
		}
		writeJSON(w, http.StatusOK, oauthTokens(AccessToken, RefreshToken))
	case "refresh_token":
		if !strings.HasPrefix(r.PostForm.Get("refresh_token"), RefreshToken) {
			writeOAuthError(w, "invalid_grant")
			return
		}
		sequence := s.refresh.Add(1)
		writeJSON(w, http.StatusOK, oauthTokens(
			fmt.Sprintf("%s-%d", AccessToken, sequence),
			fmt.Sprintf("%s-%d", RefreshToken, sequence),
		))
	default:
		writeOAuthError(w, "unsupported_grant_type")
	}
}

func oauthTokens(access, refresh string) map[string]any {
	return map[string]any{
		"access_token": access, "refresh_token": refresh, "token_type": "Bearer",
		"expires_in": 3600, "refresh_expires_in": 7776000, "membership_id": MembershipID,
	}
}

func writeOAuthError(w http.ResponseWriter, code string) {
	writeJSON(w, http.StatusBadRequest, map[string]any{"error": code})
}

func (s *Server) memberships(w http.ResponseWriter, r *http.Request) {
	if !validAccessToken(r) {
		writeBungieError(w, http.StatusUnauthorized, 99, "WebAuthRequired", "Authentication required")
		return
	}
	writeBungie(w, map[string]any{
		"destinyMemberships": []any{map[string]any{
			"membershipType": MembershipType, "membershipId": MembershipID, "displayName": DisplayName,
		}},
		"primaryMembershipId": MembershipID,
	})
}

func (s *Server) manifest(w http.ResponseWriter, _ *http.Request) {
	writeBungie(w, map[string]any{
		"version":                 ManifestVersion,
		"mobileWorldContentPaths": map[string]any{"en": "/fixtures/manifest.zip"},
	})
}

func (s *Server) manifestArchive(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="manifest.zip"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(s.manifestZip)
}

func (s *Server) profile(w http.ResponseWriter, r *http.Request) {
	if !validAccessToken(r) {
		writeBungieError(w, http.StatusUnauthorized, 99, "WebAuthRequired", "Authentication required")
		return
	}
	switch r.URL.Query().Get("components") {
	case "800":
		writeBungie(w, collectibleProfile())
	case "200":
		writeBungie(w, characterProfile())
	case "900":
		writeBungie(w, recordsProfile())
	default:
		writeBungieError(w, http.StatusBadRequest, 5, "InvalidParameters", "Unsupported profile component")
	}
}

func collectibleProfile() map[string]any {
	return map[string]any{
		"profileCollectibles": map[string]any{
			"data": map[string]any{"collectibles": map[string]any{
				"1000": map[string]any{"state": 1},
				"1001": map[string]any{"state": 0},
				"2000": map[string]any{"state": 1},
				"3000": map[string]any{"state": 0},
				"4000": map[string]any{"state": 1},
				"4001": map[string]any{"state": 0},
				"5000": map[string]any{"state": 1},
			}},
			"privacy": 1,
		},
		"characterCollectibles": map[string]any{
			"data": map[string]any{CharacterID: map[string]any{"collectibles": map[string]any{}}},
		},
	}
}

func characterProfile() map[string]any {
	return map[string]any{
		"characters": map[string]any{"data": map[string]any{
			CharacterID: map[string]any{
				"characterId": CharacterID, "classType": 2, "raceType": 0, "light": 2010,
				"emblemPath": "/e2e/emblem.png", "emblemBackgroundPath": "/e2e/emblem-background.png",
				"dateLastPlayed": "2026-07-11T18:00:00Z",
			},
		}},
	}
}

func recordsProfile() map[string]any {
	return map[string]any{
		"profileRecords": map[string]any{"data": map[string]any{"records": map[string]any{
			"9100": map[string]any{
				"state": 4,
				"objectives": []any{map[string]any{
					"objectiveHash": uint32(7101), "progress": 125, "completionValue": 500,
					"complete": false, "visible": true, "progressDescription": "Enemies defeated",
				}},
			},
			"9101": map[string]any{
				"state": 4,
				"objectives": []any{map[string]any{
					"objectiveHash": uint32(7102), "progress": 3, "completionValue": 5,
					"complete": false, "visible": true, "progressDescription": "Patterns extracted",
				}},
			},
			// Redeemed state is authoritative even though its objective is stale.
			"9301": map[string]any{
				"state": 1,
				"objectives": []any{map[string]any{
					"objectiveHash": uint32(7200), "progress": 0, "completionValue": 1,
					"complete": false, "visible": true, "progressDescription": "Triumph complete",
				}},
			},
			"9302": map[string]any{
				"state": 4,
				"objectives": []any{
					map[string]any{"objectiveHash": uint32(7201), "progress": 3, "completionValue": 5, "complete": false, "visible": true, "progressDescription": "Matches won"},
					// Visibility is intentionally absent: Bungie treats omission as visible.
					map[string]any{"objectiveHash": uint32(7202), "progress": 2, "completionValue": 2, "complete": true, "progressDescription": "Opponents defeated"},
					map[string]any{"objectiveHash": uint32(7203), "progress": 1, "completionValue": 10, "complete": false, "visible": false, "progressDescription": "Hidden objective"},
				},
			},
			"9400": map[string]any{
				"state": 4,
				"intervalObjectives": []any{
					map[string]any{"objectiveHash": uint32(7300), "progress": 1, "completionValue": 1, "complete": true, "visible": true, "progressDescription": "Gilded once"},
					map[string]any{"objectiveHash": uint32(7301), "progress": 0, "completionValue": 1, "complete": false, "visible": true, "progressDescription": "Next gilding"},
				},
			},
		}}},
		"characterRecords": map[string]any{"data": map[string]any{CharacterID: map[string]any{"records": map[string]any{}}}},
	}
}

func (s *Server) characterVendors(w http.ResponseWriter, r *http.Request) {
	if !validAccessToken(r) {
		writeBungieError(w, http.StatusUnauthorized, 99, "WebAuthRequired", "Authentication required")
		return
	}
	if r.URL.Query().Get("components") != "400,402" {
		writeBungieError(w, http.StatusBadRequest, 5, "InvalidParameters", "Unsupported vendor components")
		return
	}
	if s.weeklyScenario() == WeeklyEmpty {
		writeBungie(w, map[string]any{
			"vendors": map[string]any{"data": map[string]any{}},
			"sales":   map[string]any{"data": map[string]any{}},
		})
		return
	}
	xurKey := fmt.Sprint(XurVendorHash)
	bansheeKey := "672118013"
	writeBungie(w, map[string]any{
		"vendors": map[string]any{"data": map[string]any{
			xurKey:     map[string]any{"vendorHash": XurVendorHash, "vendorLocationIndex": 0, "enabled": true},
			bansheeKey: map[string]any{"vendorHash": uint32(672118013), "vendorLocationIndex": 0, "enabled": true},
		}},
		"sales": map[string]any{"data": map[string]any{
			xurKey:     xurSales(),
			bansheeKey: map[string]any{"saleItems": map[string]any{"0": map[string]any{"itemHash": testingModHash, "costs": []any{}}}},
		}},
	})
}

func (s *Server) publicVendors(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("components") != "400,402" {
		writeBungieError(w, http.StatusBadRequest, 5, "InvalidParameters", "Unsupported vendor components")
		return
	}
	if s.weeklyScenario() == WeeklyEmpty {
		writeBungie(w, map[string]any{
			"sales":        map[string]any{"data": map[string]any{}},
			"vendorGroups": map[string]any{"data": map[string]any{}},
		})
		return
	}
	writeBungie(w, map[string]any{
		"sales":        map[string]any{"data": map[string]any{fmt.Sprint(XurVendorHash): xurSales()}},
		"vendorGroups": map[string]any{"data": map[string]any{}},
	})
}

func xurSales() map[string]any {
	return map[string]any{"saleItems": map[string]any{
		"0": map[string]any{
			"itemHash": gjallarhornHash,
			"costs":    []any{map[string]any{"itemHash": strangeCoinHash, "quantity": 23}},
		},
	}}
}

func (s *Server) milestones(w http.ResponseWriter, _ *http.Request) {
	weekly := s.weeklyScenario()
	if weekly == WeeklyEmpty {
		writeBungie(w, map[string]any{})
		return
	}
	writeBungie(w, map[string]any{
		fmt.Sprint(weeklyMilestoneHash): map[string]any{
			"milestoneHash":   weeklyMilestoneHash,
			"availableQuests": []any{},
			"activities": []any{map[string]any{
				"activityHash": weeklyActivityHash, "modifierHashes": []uint32{weeklyModifierHash},
			}},
		},
	})
}

func (s *Server) weeklyScenario() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.scenario.Weekly
}

func (s *Server) settings(w http.ResponseWriter, _ *http.Request) {
	writeBungie(w, map[string]any{"destiny2CoreSettings": map[string]any{
		"activeSealsRootNodeHash":     activeSealsRootHash,
		"legacySealsRootNodeHash":     uint32(0),
		"exoticCatalystsRootNodeHash": combinedRecordsRootHash,
		"craftingRootNodeHash":        patternNodeHash,
	}})
}

func (s *Server) setScenario(w http.ResponseWriter, r *http.Request) {
	if !requestIsLoopback(r) {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "loopback_only"})
		return
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024))
	decoder.DisallowUnknownFields()
	var next Scenario
	if err := decoder.Decode(&next); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_scenario"})
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_scenario"})
		return
	}
	if next.Weekly != WeeklyNormal && next.Weekly != WeeklyEmpty {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_weekly_scenario"})
		return
	}
	s.mu.Lock()
	s.scenario = next
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, next)
}

func (s *Server) resetScenario(w http.ResponseWriter, r *http.Request) {
	if !requestIsLoopback(r) {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "loopback_only"})
		return
	}
	next := Scenario{Weekly: WeeklyNormal}
	s.mu.Lock()
	s.scenario = next
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, next)
}

func validAccessToken(r *http.Request) bool {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	return token == AccessToken || strings.HasPrefix(token, AccessToken+"-")
}

func requestIsLoopback(r *http.Request) bool {
	host := r.RemoteAddr
	if parsedHost, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		host = parsedHost
	}
	host = strings.Trim(host, "[]")
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func writeBungie(w http.ResponseWriter, response any) {
	writeJSON(w, http.StatusOK, map[string]any{
		"Response": response, "ErrorCode": 1, "ErrorStatus": "Success", "Message": "Ok", "ThrottleSeconds": 0,
	})
}

func writeBungieError(w http.ResponseWriter, status, code int, errorStatus, message string) {
	writeJSON(w, status, map[string]any{
		"Response": nil, "ErrorCode": code, "ErrorStatus": errorStatus, "Message": message, "ThrottleSeconds": 0,
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
