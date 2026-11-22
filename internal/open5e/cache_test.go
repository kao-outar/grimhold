package open5e

import (
	"testing"
	"time"
)

type fakeClock struct {
	t time.Time
}

func (f *fakeClock) Now() time.Time { return f.t }

func TestCacheExpiry(t *testing.T) {
	clk := &fakeClock{t: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	c := newCache(5*time.Minute, time.Hour, clk)
	defer c.Close()

	c.Set("k", []byte("hello"))
	if data, ok := c.Get("k"); !ok || string(data) != "hello" {
		t.Fatalf("expected cache hit, got ok=%v data=%q", ok, data)
	}

	clk.t = clk.t.Add(6 * time.Minute)
	if _, ok := c.Get("k"); ok {
		t.Fatal("expected miss after TTL")
	}
}

func TestPurgeExpired(t *testing.T) {
	clk := &fakeClock{t: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	c := newCache(1*time.Minute, time.Hour, clk)
	defer c.Close()

	c.Set("a", []byte("1"))
	clk.t = clk.t.Add(2 * time.Minute)
	c.Set("b", []byte("2"))
	c.PurgeExpired()
	if c.Len() != 1 {
		t.Fatalf("want 1 entry after purge, got %d", c.Len())
	}
	if _, ok := c.Get("b"); !ok {
		t.Fatal("fresh entry should remain")
	}
}
