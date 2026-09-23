package ownedrolls

import (
	"context"
	"errors"
	"testing"

	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/manifest"
)

type fakeProfiles struct {
	resp *bungie.ProfileResponse
	err  error

	gotType       int
	gotID         string
	gotToken      string
	gotComponents []int
}

func (f *fakeProfiles) GetProfile(_ context.Context, membershipType int, membershipID, token string, components []int) (*bungie.ProfileResponse, error) {
	f.gotType, f.gotID, f.gotToken, f.gotComponents = membershipType, membershipID, token, components
	return f.resp, f.err
}

type fakePerks struct {
	cols map[uint32][]manifest.PerkColumn
	err  error
}

func (f fakePerks) GetWeaponPerks(itemHash uint32) ([]manifest.PerkColumn, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.cols[itemHash], nil
}

type fakeCreds struct {
	token string
	err   error
}

func (f fakeCreds) GetValidToken(string) (string, error) { return f.token, f.err }

// weapon 1000 has one trait column with a paired base/enhanced perk and a
// base-only perk. Hash 7777 is a mod plug that is not a perk column.
func weaponPerks() fakePerks {
	return fakePerks{cols: map[uint32][]manifest.PerkColumn{
		1000: {{
			Role: "trait", Label: "Trait 1",
			Perks: []string{"Outlaw", "Firefly"},
			Plugs: []manifest.PerkPlug{
				{Name: "Outlaw", Base: 111, Enhanced: 911},
				{Name: "Firefly", Base: 222},
			},
		}},
	}}
}

func ptr(v uint32) *uint32 { return &v }
func boolPtr(v bool) *bool { return &v }

func profileWith(items []bungie.DestinyItemComponent, sockets map[string]bungie.ItemSocketsComponent) *bungie.ProfileResponse {
	resp := &bungie.ProfileResponse{}
	resp.Response.ProfileInventory = bungie.ComponentEnvelope[bungie.InventoryComponent]{
		Data: &bungie.InventoryComponent{Items: items}, Privacy: 2,
	}
	resp.Response.ItemComponents.Sockets = bungie.ComponentEnvelope[map[string]bungie.ItemSocketsComponent]{
		Data: &sockets, Privacy: 2,
	}
	return resp
}

func svc(p *fakeProfiles) *Service { return NewService(p, weaponPerks(), fakeCreds{token: "t"}) }

// The accepted owner capture recorded privacy = 2 on five of six components
// with the data present on all six. Availability is data presence, not privacy;
// gating on privacy would refuse a working read.
func TestRead_PrivateComponentsStillCarryTheOwnersData(t *testing.T) {
	p := &fakeProfiles{resp: profileWith(
		[]bungie.DestinyItemComponent{{ItemHash: 1000, ItemInstanceID: "a"}},
		map[string]bungie.ItemSocketsComponent{"a": {Sockets: []bungie.ItemSocketState{
			{PlugHash: ptr(111)}, {PlugHash: ptr(222)},
		}}},
	)}
	rolls, err := svc(p).Read(context.Background(), 3, "m1")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(rolls) != 1 {
		t.Fatalf("rolls = %d, want 1", len(rolls))
	}
	// Sorted, matching how a saved target stores its perks.
	if len(rolls[0].Perks) != 2 || rolls[0].Perks[0] != "Firefly" || rolls[0].Perks[1] != "Outlaw" {
		t.Errorf("perks = %v, want [Firefly Outlaw]", rolls[0].Perks)
	}
}

// An enhanced plug resolves to the same name its base would.
func TestRead_EnhancedPlugResolvesToTheBaseName(t *testing.T) {
	p := &fakeProfiles{resp: profileWith(
		[]bungie.DestinyItemComponent{{ItemHash: 1000, ItemInstanceID: "a"}},
		map[string]bungie.ItemSocketsComponent{"a": {Sockets: []bungie.ItemSocketState{{PlugHash: ptr(911)}}}},
	)}
	rolls, _ := svc(p).Read(context.Background(), 3, "m1")
	if len(rolls) != 1 || len(rolls[0].Perks) != 1 || rolls[0].Perks[0] != "Outlaw" {
		t.Errorf("rolls = %+v, want the enhanced plug read as Outlaw", rolls)
	}
}

// Component 305 is positional over every socket. A plug that is not in the
// weapon's perk pool is a mod or a cosmetic, not part of the roll. An empty
// socket has a null plug hash and must not be read as plug 0.
func TestRead_IgnoresNonPerkPlugsAndEmptySockets(t *testing.T) {
	p := &fakeProfiles{resp: profileWith(
		[]bungie.DestinyItemComponent{{ItemHash: 1000, ItemInstanceID: "a"}},
		map[string]bungie.ItemSocketsComponent{"a": {Sockets: []bungie.ItemSocketState{
			{PlugHash: ptr(7777)}, // a mod
			{PlugHash: nil},       // empty socket
			{PlugHash: ptr(111)},  // a real perk
		}}},
	)}
	rolls, _ := svc(p).Read(context.Background(), 3, "m1")
	if len(rolls) != 1 || len(rolls[0].Perks) != 1 || rolls[0].Perks[0] != "Outlaw" {
		t.Errorf("rolls = %+v, want only the real perk", rolls)
	}
}

// An item with no perk columns is not a weapon roll.
func TestRead_SkipsItemsThatAreNotWeapons(t *testing.T) {
	p := &fakeProfiles{resp: profileWith(
		[]bungie.DestinyItemComponent{{ItemHash: 4242, ItemInstanceID: "a"}},
		map[string]bungie.ItemSocketsComponent{"a": {Sockets: []bungie.ItemSocketState{{PlugHash: ptr(111)}}}},
	)}
	rolls, err := svc(p).Read(context.Background(), 3, "m1")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(rolls) != 0 {
		t.Errorf("rolls = %+v, want none", rolls)
	}
}

// Items come from the vault, each character's inventory, and equipped slots.
func TestRead_GathersVaultCharacterAndEquipped(t *testing.T) {
	resp := &bungie.ProfileResponse{}
	resp.Response.ProfileInventory = bungie.ComponentEnvelope[bungie.InventoryComponent]{
		Data: &bungie.InventoryComponent{Items: []bungie.DestinyItemComponent{{ItemHash: 1000, ItemInstanceID: "vault"}}},
	}
	charInv := map[string]bungie.InventoryComponent{
		"c1": {Items: []bungie.DestinyItemComponent{{ItemHash: 1000, ItemInstanceID: "carried"}}},
	}
	resp.Response.CharacterInventories = bungie.ComponentEnvelope[map[string]bungie.InventoryComponent]{Data: &charInv}
	equipped := map[string]bungie.CharacterEquipmentComponent{
		"c1": {Items: []bungie.DestinyItemComponent{{ItemHash: 1000, ItemInstanceID: "equipped"}}},
	}
	resp.Response.CharacterEquipment = bungie.ComponentEnvelope[map[string]bungie.CharacterEquipmentComponent]{Data: &equipped}
	sockets := map[string]bungie.ItemSocketsComponent{
		"vault":    {Sockets: []bungie.ItemSocketState{{PlugHash: ptr(111)}}},
		"carried":  {Sockets: []bungie.ItemSocketState{{PlugHash: ptr(111)}}},
		"equipped": {Sockets: []bungie.ItemSocketState{{PlugHash: ptr(111)}}},
	}
	resp.Response.ItemComponents.Sockets = bungie.ComponentEnvelope[map[string]bungie.ItemSocketsComponent]{Data: &sockets}

	rolls, err := svc(&fakeProfiles{resp: resp}).Read(context.Background(), 3, "m1")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(rolls) != 3 {
		t.Fatalf("rolls = %d, want 3 (vault, carried, equipped)", len(rolls))
	}
}

// "We could not read your inventory" and "you hold nothing" are different
// facts and must not render alike.
func TestRead_UnavailableComponentsAreNotAnEmptyInventory(t *testing.T) {
	// No inventory component at all.
	resp := &bungie.ProfileResponse{}
	sockets := map[string]bungie.ItemSocketsComponent{}
	resp.Response.ItemComponents.Sockets = bungie.ComponentEnvelope[map[string]bungie.ItemSocketsComponent]{Data: &sockets}
	if _, err := svc(&fakeProfiles{resp: resp}).Read(context.Background(), 3, "m1"); !errors.Is(err, ErrInventoryUnavailable) {
		t.Errorf("missing inventory err = %v, want ErrInventoryUnavailable", err)
	}

	// Inventory present but the sockets component is disabled.
	resp2 := profileWith(nil, nil)
	resp2.Response.ItemComponents.Sockets.Disabled = boolPtr(true)
	if _, err := svc(&fakeProfiles{resp: resp2}).Read(context.Background(), 3, "m1"); !errors.Is(err, ErrInventoryUnavailable) {
		t.Errorf("disabled sockets err = %v, want ErrInventoryUnavailable", err)
	}

	// Present and genuinely empty is an answer, not a failure.
	empty := profileWith([]bungie.DestinyItemComponent{}, map[string]bungie.ItemSocketsComponent{})
	rolls, err := svc(&fakeProfiles{resp: empty}).Read(context.Background(), 3, "m1")
	if err != nil {
		t.Fatalf("empty inventory err = %v, want nil", err)
	}
	if len(rolls) != 0 {
		t.Errorf("rolls = %+v, want none", rolls)
	}
}

// Without a Bungie authorization there is no answer at all — unlike a vendor
// read, an inventory read has nothing public to fall back to.
func TestRead_RequiresACredential(t *testing.T) {
	p := &fakeProfiles{}
	for _, creds := range []fakeCreds{{err: errors.New("expired")}, {token: ""}} {
		s := NewService(p, weaponPerks(), creds)
		if _, err := s.Read(context.Background(), 3, "m1"); !errors.Is(err, ErrNoCredential) {
			t.Errorf("err = %v, want ErrNoCredential", err)
		}
	}
	if p.gotID != "" {
		t.Error("the profile was fetched without a credential")
	}
}

// 310 is deliberately not requested: it answers what may be swapped in now,
// not what the weapon rolled, and it dominated the captured response size.
func TestRead_RequestsOnlyTheComponentsItNeeds(t *testing.T) {
	p := &fakeProfiles{resp: profileWith(nil, map[string]bungie.ItemSocketsComponent{})}
	svc(p).Read(context.Background(), 3, "m1")
	want := map[int]bool{102: true, 201: true, 205: true, 305: true}
	if len(p.gotComponents) != len(want) {
		t.Fatalf("components = %v, want exactly %v", p.gotComponents, want)
	}
	for _, c := range p.gotComponents {
		if !want[c] {
			t.Errorf("requested component %d, which is not needed", c)
		}
		if c == 310 {
			t.Error("component 310 was requested")
		}
	}
	if p.gotToken != "t" || p.gotType != 3 || p.gotID != "m1" {
		t.Errorf("profile read got %d/%q/%q", p.gotType, p.gotID, p.gotToken)
	}
}

// A perk pool that cannot be read — a manifest still downloading or mid-swap —
// is its own answer. It must not pass through as an anonymous failure, and it
// must not read as a weapon with nothing to report.
func TestRead_UnreadablePerkPoolIsItsOwnFailure(t *testing.T) {
	p := &fakeProfiles{resp: profileWith(
		[]bungie.DestinyItemComponent{{ItemHash: 1000, ItemInstanceID: "a"}},
		map[string]bungie.ItemSocketsComponent{"a": {Sockets: []bungie.ItemSocketState{{PlugHash: ptr(111)}}}},
	)}
	s := NewService(p, fakePerks{err: manifest.ErrNotReady}, fakeCreds{token: "t"})
	rolls, err := s.Read(context.Background(), 3, "m1")
	if !errors.Is(err, ErrPerksUnavailable) {
		t.Fatalf("err = %v, want ErrPerksUnavailable", err)
	}
	if rolls != nil {
		t.Errorf("rolls = %+v, want none alongside the failure", rolls)
	}
}

func TestRead_ProfileFailurePassesThrough(t *testing.T) {
	boom := errors.New("bungie is down")
	_, err := svc(&fakeProfiles{err: boom}).Read(context.Background(), 3, "m1")
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the original failure", err)
	}
}
