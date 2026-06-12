package cache

import (
	"testing"
	"time"
)

func TestMemoryCache_SetGetDelete(t *testing.T) {
	c := NewMemoryCache(time.Minute, time.Minute)

	if _, ok := c.Get("missing"); ok {
		t.Error("Get on empty cache returned a value")
	}

	c.Set("k", "v", time.Minute)
	got, ok := c.Get("k")
	if !ok || got.(string) != "v" {
		t.Fatalf("Get = %v, %v; want v, true", got, ok)
	}

	c.Delete("k")
	if _, ok := c.Get("k"); ok {
		t.Error("Get after Delete returned a value")
	}
}

func TestMemoryCache_TTLExpiry(t *testing.T) {
	c := NewMemoryCache(time.Minute, time.Minute)
	c.Set("k", 42, 10*time.Millisecond)
	time.Sleep(30 * time.Millisecond)
	if _, ok := c.Get("k"); ok {
		t.Error("value survived past its TTL")
	}
}

func TestMemoryCache_Clear(t *testing.T) {
	c := NewMemoryCache(time.Minute, time.Minute)
	c.Set("a", 1, time.Minute)
	c.Set("b", 2, time.Minute)
	c.Clear()
	if _, ok := c.Get("a"); ok {
		t.Error("value survived Clear")
	}
}

func TestNoOpCache(t *testing.T) {
	c := NewNoOpCache()
	c.Set("k", "v", time.Minute)
	if _, ok := c.Get("k"); ok {
		t.Error("no-op cache returned a stored value")
	}
	c.Delete("k") // must not panic
	c.Clear()     // must not panic
}
