package cache

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/gombit-dev/gombit/config"
)

type cachedWidget struct {
	ID   int
	Name string
}

func TestMemoryGetSetDeleteValueSemantics(t *testing.T) {
	ctx := context.Background()
	c := NewMemory()
	want := cachedWidget{ID: 1, Name: "stored"}

	if err := c.Set(ctx, "widgets:1", want, time.Minute); err != nil {
		t.Fatalf("Set() error = %v, want nil", err)
	}

	var got cachedWidget
	found, err := c.Get(ctx, "widgets:1", &got)
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	if !found {
		t.Fatal("Get() found = false, want true")
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Get() value = %#v, want %#v", got, want)
	}

	if err := c.Delete(ctx, "widgets:1"); err != nil {
		t.Fatalf("Delete() error = %v, want nil", err)
	}
	found, err = c.Get(ctx, "widgets:1", &got)
	if err != nil {
		t.Fatalf("Get() after Delete() error = %v, want nil", err)
	}
	if found {
		t.Fatal("Get() after Delete() found = true, want false")
	}
}

func TestMemoryExpiresValues(t *testing.T) {
	ctx := context.Background()
	c := NewMemory()
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return now }

	if err := c.Set(ctx, "short", "value", time.Second); err != nil {
		t.Fatalf("Set() error = %v, want nil", err)
	}
	now = now.Add(time.Second)

	var got string
	found, err := c.Get(ctx, "short", &got)
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	if found {
		t.Fatal("Get() found = true, want false after ttl")
	}
}

func TestMemoryIncrementPreservesExistingTTL(t *testing.T) {
	ctx := context.Background()
	c := NewMemory()
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return now }

	if err := c.Set(ctx, "counter", int64(1), time.Second); err != nil {
		t.Fatalf("Set() error = %v, want nil", err)
	}
	count, err := c.Increment(ctx, "counter", 1)
	if err != nil {
		t.Fatalf("Increment() error = %v, want nil", err)
	}
	if count != 2 {
		t.Fatalf("Increment() = %d, want 2", count)
	}

	now = now.Add(time.Second)
	var got int64
	found, err := c.Get(ctx, "counter", &got)
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	if found {
		t.Fatalf("Get() found = true, want false after original ttl; value = %d", got)
	}
}

func TestMemoryIncrementValueSemantics(t *testing.T) {
	ctx := context.Background()
	c := NewMemory()

	got, err := c.Increment(ctx, "rate:client", 1)
	if err != nil {
		t.Fatalf("Increment() error = %v, want nil", err)
	}
	if got != 1 {
		t.Fatalf("Increment() = %d, want 1", got)
	}

	got, err = c.Increment(ctx, "rate:client", 4)
	if err != nil {
		t.Fatalf("Increment() error = %v, want nil", err)
	}
	if got != 5 {
		t.Fatalf("Increment() = %d, want 5", got)
	}
}

func TestMemoryIncrementRejectsNonIntegerValue(t *testing.T) {
	ctx := context.Background()
	c := NewMemory()
	if err := c.Set(ctx, "counter", "not-an-int", 0); err != nil {
		t.Fatalf("Set() error = %v, want nil", err)
	}

	_, err := c.Increment(ctx, "counter", 1)
	if err == nil {
		t.Fatal("Increment() error = nil, want error")
	}
}

func TestNoopDriver(t *testing.T) {
	ctx := context.Background()
	var c Cache = Noop{}

	if err := c.Set(ctx, "ignored", cachedWidget{ID: 1}, time.Minute); err != nil {
		t.Fatalf("Set() error = %v, want nil", err)
	}
	var got cachedWidget
	found, err := c.Get(ctx, "ignored", &got)
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	if found {
		t.Fatal("Get() found = true, want false")
	}
	count, err := c.Increment(ctx, "rate:client", 1)
	if err != nil {
		t.Fatalf("Increment() error = %v, want nil", err)
	}
	if count != 0 {
		t.Fatalf("Increment() = %d, want 0", count)
	}
}

func TestOpenMemoryDriverFromConfig(t *testing.T) {
	cfg := config.Default().Cache
	cfg.Driver = config.CacheDriverMemory
	cfg.Namespace = "test"

	store, err := Open(cfg)
	if err != nil {
		t.Fatalf("Open() error = %v, want nil", err)
	}
	if store.Driver() != DriverMemory {
		t.Fatalf("Driver() = %q, want %q", store.Driver(), DriverMemory)
	}
	if store.Redis() != nil {
		t.Fatalf("Redis() = %v, want nil", store.Redis())
	}

	if err := store.Set(context.Background(), "key", "value", 0); err != nil {
		t.Fatalf("Set() error = %v, want nil", err)
	}
	var got string
	found, err := store.Get(context.Background(), "key", &got)
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	if !found || got != "value" {
		t.Fatalf("Get() = (%t, %q), want (true, value)", found, got)
	}
}

func TestCacheAndRateLimiterUsersCompileAgainstInterface(t *testing.T) {
	ctx := context.Background()
	c := NewMemory()

	if err := writeCacheUser(ctx, c); err != nil {
		t.Fatalf("writeCacheUser() error = %v, want nil", err)
	}
	limited, err := rateLimiterUser(ctx, c, "client:1", 2)
	if err != nil {
		t.Fatalf("rateLimiterUser() error = %v, want nil", err)
	}
	if limited {
		t.Fatal("rateLimiterUser() limited = true, want false")
	}
	limited, err = rateLimiterUser(ctx, c, "client:1", 2)
	if err != nil {
		t.Fatalf("rateLimiterUser() error = %v, want nil", err)
	}
	if limited {
		t.Fatal("rateLimiterUser() limited = true, want false")
	}
	limited, err = rateLimiterUser(ctx, c, "client:1", 2)
	if err != nil {
		t.Fatalf("rateLimiterUser() error = %v, want nil", err)
	}
	if !limited {
		t.Fatal("rateLimiterUser() limited = false, want true")
	}
}

func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := NewMemory()

	if err := c.Set(ctx, "key", "value", 0); !errors.Is(err, context.Canceled) {
		t.Fatalf("Set() error = %v, want context.Canceled", err)
	}
	if _, err := c.Get(ctx, "key", nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("Get() error = %v, want context.Canceled", err)
	}
	if err := c.Delete(ctx, "key"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Delete() error = %v, want context.Canceled", err)
	}
	if _, err := c.Increment(ctx, "key", 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("Increment() error = %v, want context.Canceled", err)
	}
}

func writeCacheUser(ctx context.Context, c Cache) error {
	return c.Set(ctx, "widgets:list", []cachedWidget{{ID: 1, Name: "one"}}, time.Minute)
}

func rateLimiterUser(ctx context.Context, c Cache, key string, limit int64) (bool, error) {
	count, err := c.Increment(ctx, "rate:"+key, 1)
	if err != nil {
		return false, err
	}
	return count > limit, nil
}
