package search

import (
	"testing"
)

func TestSearchRanking(t *testing.T) {
	svc := &Service{
		entries: []Entry{
			{Hash: 1, Name: "Vex Mythoclast", nameLower: "vex mythoclast", Type: "Fusion Rifle", Rarity: "Exotic"},
			{Hash: 2, Name: "Hawkmoon", nameLower: "hawkmoon", Type: "Hand Cannon", Rarity: "Exotic"},
			{Hash: 3, Name: "Exotic Cipher", nameLower: "exotic cipher", Type: "Item", Rarity: "Exotic"},
		},
		builtVersion: "test",
	}

	// Prefix match "vex" should return Vex Mythoclast first
	results := svc.Search("vex", 10)
	if len(results) == 0 {
		t.Fatal("expected results for 'vex'")
	}
	if results[0].Name != "Vex Mythoclast" {
		t.Errorf("expected Vex Mythoclast first, got %s", results[0].Name)
	}

	// "exotic" matches "Exotic Cipher" (prefix) before others (no match)
	results = svc.Search("exotic", 10)
	if len(results) == 0 {
		t.Fatal("expected results for 'exotic'")
	}
	if results[0].Name != "Exotic Cipher" {
		t.Errorf("expected Exotic Cipher first (prefix match), got %s", results[0].Name)
	}
}

func TestSearchLimitClamping(t *testing.T) {
	svc := &Service{
		entries: []Entry{
			{Hash: 1, Name: "Item One", nameLower: "item one"},
			{Hash: 2, Name: "Item Two", nameLower: "item two"},
			{Hash: 3, Name: "Item Three", nameLower: "item three"},
		},
		builtVersion: "test",
	}

	results := svc.Search("item", 2)
	if len(results) > 2 {
		t.Errorf("expected at most 2 results with limit=2, got %d", len(results))
	}

	// limit=0 defaults to 20
	results = svc.Search("item", 0)
	if len(results) > 20 {
		t.Errorf("expected at most 20 results with limit=0, got %d", len(results))
	}
}

func TestSearchCaseInsensitive(t *testing.T) {
	svc := &Service{
		entries: []Entry{
			{Hash: 1, Name: "Hawkmoon", nameLower: "hawkmoon"},
		},
		builtVersion: "test",
	}

	results := svc.Search("HAWK", 10)
	if len(results) == 0 || results[0].Name != "Hawkmoon" {
		t.Errorf("expected case-insensitive match for HAWK, got %v", results)
	}
}

func TestSearchEmptyWhenNotBuilt(t *testing.T) {
	svc := &Service{}
	results := svc.Search("anything", 10)
	if results != nil {
		t.Errorf("expected nil before index built, got %v", results)
	}
}
