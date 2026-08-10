package db

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestNewStores_NilPoolIsDegradedNotNil(t *testing.T) {
	s := NewStores(nil)

	if s.Available() {
		t.Error("Available() should be false without a pool")
	}
	// The whole point: no field is nil, so no caller can dereference one.
	v := reflect.ValueOf(*s)
	for i := range v.NumField() {
		f := v.Field(i)
		if f.Kind() != reflect.Interface {
			continue
		}
		if f.IsNil() {
			t.Errorf("Stores.%s is nil in degraded mode", v.Type().Field(i).Name)
		}
	}
}

// Every method of every degraded store must report ErrUnavailable. Driven by
// reflection so a method added to a store later cannot quietly ship a degraded
// implementation that returns a zero value — which would look like real,
// empty data to the caller.
func TestDegradedStores_EveryMethodReportsUnavailable(t *testing.T) {
	s := NewStores(nil)
	for _, tc := range []struct {
		name  string
		store any
	}{
		{"Users", s.Users},
		{"Tokens", s.Tokens},
		{"Wishlist", s.Wishlist},
		{"Prefs", s.Prefs},
		{"Flags", s.Flags},
		{"Audit", s.Audit},
		{"Pinger", s.Pinger},
	} {
		v := reflect.ValueOf(tc.store)
		typ := v.Type()
		if typ.NumMethod() == 0 {
			t.Errorf("%s exposes no methods", tc.name)
		}
		for i := range typ.NumMethod() {
			m := typ.Method(i)
			args := make([]reflect.Value, m.Type.NumIn()-1)
			for j := range args {
				at := m.Type.In(j + 1)
				if at == reflect.TypeFor[context.Context]() {
					args[j] = reflect.ValueOf(context.Background())
					continue
				}
				args[j] = reflect.Zero(at)
			}
			out := v.Method(i).Call(args)
			if len(out) == 0 {
				t.Errorf("%s.%s returns nothing; it cannot report unavailability", tc.name, m.Name)
				continue
			}
			last, ok := out[len(out)-1].Interface().(error)
			if !ok {
				t.Errorf("%s.%s does not return an error last", tc.name, m.Name)
				continue
			}
			if !errors.Is(last, ErrUnavailable) {
				t.Errorf("%s.%s returned %v, want ErrUnavailable", tc.name, m.Name, last)
			}
		}
	}
}

// A degraded read must not look like a successful empty read: an empty wishlist
// and an empty admin roster are indistinguishable from real data.
func TestDegradedStores_ReadsDoNotReturnEmptySuccess(t *testing.T) {
	s := NewStores(nil)
	ctx := context.Background()

	if items, err := s.Wishlist.List(ctx, 1); err == nil {
		t.Errorf("wishlist List returned %v with no error", items)
	}
	if users, err := s.Users.ListUsers(ctx, "", 10); err == nil {
		t.Errorf("ListUsers returned %v with no error", users)
	}
	if flags, err := s.Flags.List(ctx); err == nil {
		t.Errorf("flag List returned %v with no error", flags)
	}
	if entries, _, err := s.Audit.List(ctx, AuditFilter{}); err == nil {
		t.Errorf("audit List returned %v with no error", entries)
	}
	if _, _, err := s.Users.RotateSession(ctx, "s", "m", "a", "b", time.Now()); err == nil {
		t.Error("RotateSession succeeded without a database")
	}
}
