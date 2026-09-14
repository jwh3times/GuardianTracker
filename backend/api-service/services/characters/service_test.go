package characters

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/items"
)

func newTestService(t *testing.T, handler http.HandlerFunc) (*Service, func()) {
	t.Helper()
	srv := httptest.NewServer(handler)
	client := bungie.NewClient("k", srv.URL, 100, 100)
	c := cache.NewMemoryCache(time.Minute, 0)
	return NewService(client, nil, c, time.Minute), srv.Close
}

type fakeEquipmentItems struct {
	facts  map[uint32]items.AcquisitionFacts
	err    error
	hashes []uint32
}

func (f *fakeEquipmentItems) Lookup(_ context.Context, hashes []uint32) (map[uint32]items.AcquisitionFacts, error) {
	f.hashes = append([]uint32(nil), hashes...)
	return f.facts, f.err
}

func TestGetCharacters_MapsAndSortsMostRecentFirst(t *testing.T) {
	svc, closeSrv := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ErrorCode":1,"Response":{"characters":{"data":{
			"older":{"characterId":"older","classType":0,"raceType":0,"light":2000,
				"emblemPath":"/e1.png","dateLastPlayed":"2026-06-01T00:00:00Z"},
			"newer":{"characterId":"newer","classType":2,"raceType":2,"light":2010,
				"emblemPath":"","dateLastPlayed":"2026-06-10T00:00:00Z"}}}}}`)
	})
	defer closeSrv()

	chars, err := svc.GetCharacters(context.Background(), 3, "id", "tok")
	if err != nil {
		t.Fatalf("GetCharacters: %v", err)
	}
	if len(chars) != 2 {
		t.Fatalf("len = %d, want 2", len(chars))
	}
	// Most recently played sorts first.
	if chars[0].CharacterID != "newer" {
		t.Errorf("chars[0] = %q, want newer", chars[0].CharacterID)
	}
	if chars[0].ClassName != "Warlock" || chars[0].RaceName != "Exo" {
		t.Errorf("newer class/race = %q/%q", chars[0].ClassName, chars[0].RaceName)
	}
	// absoluteURL prefixes a non-empty path and leaves empty paths empty.
	if chars[1].EmblemPath != "https://www.bungie.net/e1.png" {
		t.Errorf("older emblem = %q", chars[1].EmblemPath)
	}
	if chars[0].EmblemPath != "" {
		t.Errorf("empty emblem should stay empty, got %q", chars[0].EmblemPath)
	}
}

func TestGetCharacters_UsesCacheOnSecondCall(t *testing.T) {
	var calls int
	svc, closeSrv := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		fmt.Fprint(w, `{"ErrorCode":1,"Response":{"characters":{"data":{
			"c1":{"characterId":"c1","classType":1,"raceType":1,"light":2000,"dateLastPlayed":"2026-06-10T00:00:00Z"}}}}}`)
	})
	defer closeSrv()

	if _, err := svc.GetCharacters(context.Background(), 3, "id", "tok"); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if _, err := svc.GetCharacters(context.Background(), 3, "id", "tok"); err != nil {
		t.Fatalf("second call: %v", err)
	}
	if calls != 1 {
		t.Errorf("upstream calls = %d, want 1 (second served from cache)", calls)
	}

	// InvalidateCache forces a refetch.
	svc.InvalidateCache(3, "id")
	if _, err := svc.GetCharacters(context.Background(), 3, "id", "tok"); err != nil {
		t.Fatalf("third call: %v", err)
	}
	if calls != 2 {
		t.Errorf("upstream calls = %d, want 2 after invalidation", calls)
	}
}

func TestGetCharacters_PropagatesUpstreamError(t *testing.T) {
	svc, closeSrv := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ErrorCode":5,"ErrorStatus":"SystemDisabled","Message":"down"}`)
	})
	defer closeSrv()

	if _, err := svc.GetCharacters(context.Background(), 3, "id", "tok"); err == nil {
		t.Fatal("expected error from upstream ErrorCode 5")
	}
}

func TestGetEquipment_ProjectsVerifiedComponentsAndOrdersSlots(t *testing.T) {
	const characterID = "2305843009263456789"
	var components string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		components = r.URL.Query().Get("components")
		fmt.Fprintf(w, `{"ErrorCode":1,"Response":{
			"characters":{"privacy":1,"data":{"%s":{"characterId":"%s"}}},
			"characterEquipment":{"privacy":1,"data":{"%s":{"items":[
				{"itemHash":20,"itemInstanceId":"instance-20","bucketHash":3448274439},
				{"itemHash":10,"itemInstanceId":"instance-10","bucketHash":1498876634},
				{"itemHash":30,"bucketHash":999}
			]}}},
			"itemComponents":{"instances":{"privacy":2,"data":{
				"instance-10":{"primaryStat":{"statHash":1480404414,"value":550}},
				"instance-20":{"primaryStat":{"statHash":3897883278,"value":551}}
			}}}
		}}`, characterID, characterID, characterID)
	}))
	defer srv.Close()

	reader := &fakeEquipmentItems{facts: map[uint32]items.AcquisitionFacts{
		10: {ItemHash: 10, Name: "Kinetic weapon", ItemType: "Hand Cannon", Rarity: "Legendary", Icon: "/10.png"},
		20: {ItemHash: 20, Name: "Helmet", ItemType: "Helmet", Rarity: "Exotic", Icon: "/20.png"},
		30: {ItemHash: 30, Name: "Future slot", ItemType: "Item", Rarity: "Rare", Icon: "/30.png"},
	}}
	svc := NewService(bungie.NewClient("k", srv.URL, 100, 100), reader, cache.NewNoOpCache(), time.Minute)

	detail, err := svc.GetEquipment(context.Background(), 3, "membership", characterID, "token")
	if err != nil {
		t.Fatalf("GetEquipment: %v", err)
	}
	if components != "200,205,300" {
		t.Errorf("components = %q, want 200,205,300", components)
	}
	if detail.State != EquipmentReady || len(detail.Items) != 3 {
		t.Fatalf("detail = %+v, want ready with 3 items", detail)
	}
	if got := reader.hashes; fmt.Sprint(got) != "[20 10 30]" {
		t.Errorf("Lookup hashes = %v, want equipment order", got)
	}
	if got := detail.Items[0]; got.ItemHash != "10" || got.Slot != "Kinetic" || got.Group != "Weapons" || got.Power == nil || *got.Power != 550 {
		t.Errorf("first item = %+v, want resolved Kinetic weapon at 550", got)
	}
	if got := detail.Items[1]; got.ItemHash != "20" || got.Slot != "Helmet" || got.Group != "Armor" || got.Power == nil || *got.Power != 551 {
		t.Errorf("second item = %+v, want resolved Helmet at 551", got)
	}
	if got := detail.Items[2]; got.Slot != "Item" || got.Group != "Equipment" || got.Power != nil {
		t.Errorf("unknown-bucket item = %+v, want visible Equipment fallback without Power", got)
	}
}

func TestGetEquipment_DistinguishesUnavailableFromEmpty(t *testing.T) {
	const characterID = "2305843009263456789"
	tests := []struct {
		name      string
		equipment string
		wantState EquipmentState
	}{
		{"absent data", `{"privacy":1}`, EquipmentUnavailable},
		{"disabled data", `{"privacy":1,"disabled":true,"data":{"` + characterID + `":{"items":[]}}}`, EquipmentUnavailable},
		{"ready empty", `{"privacy":1,"data":{"` + characterID + `":{"items":[]}}}`, EquipmentReady},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				fmt.Fprintf(w, `{"ErrorCode":1,"Response":{
					"characters":{"privacy":1,"data":{"%s":{"characterId":"%s"}}},
					"characterEquipment":%s
				}}`, characterID, characterID, tc.equipment)
			}))
			defer srv.Close()

			svc := NewService(bungie.NewClient("k", srv.URL, 100, 100), &fakeEquipmentItems{}, cache.NewNoOpCache(), time.Minute)
			detail, err := svc.GetEquipment(context.Background(), 3, "membership", characterID, "token")
			if err != nil {
				t.Fatalf("GetEquipment: %v", err)
			}
			if detail.State != tc.wantState || detail.Items == nil || len(detail.Items) != 0 {
				t.Errorf("detail = %+v, want state %q and allocated empty items", detail, tc.wantState)
			}
		})
	}
}

func TestGetEquipment_RejectsCharacterOutsideMembership(t *testing.T) {
	svc, closeSrv := newTestService(t, func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"ErrorCode":1,"Response":{
			"characters":{"privacy":1,"data":{"2305843009263456789":{"characterId":"2305843009263456789"}}},
			"characterEquipment":{"privacy":1,"data":{}}
		}}`)
	})
	defer closeSrv()

	_, err := svc.GetEquipment(context.Background(), 3, "membership", "2305843009000000000", "token")
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Fatalf("GetEquipment error = %v, want ErrCharacterNotFound", err)
	}
}

func TestGetEquipment_PropagatesItemLookupFailure(t *testing.T) {
	const characterID = "2305843009263456789"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `{"ErrorCode":1,"Response":{
			"characters":{"privacy":1,"data":{"%s":{"characterId":"%s"}}},
			"characterEquipment":{"privacy":1,"data":{"%s":{"items":[{"itemHash":10,"bucketHash":1498876634}]}}}
		}}`, characterID, characterID, characterID)
	}))
	defer srv.Close()

	want := errors.New("manifest unavailable")
	svc := NewService(bungie.NewClient("k", srv.URL, 100, 100), &fakeEquipmentItems{err: want}, cache.NewNoOpCache(), time.Minute)
	_, err := svc.GetEquipment(context.Background(), 3, "membership", characterID, "token")
	if !errors.Is(err, want) {
		t.Fatalf("GetEquipment error = %v, want wrapped item lookup error", err)
	}
}
