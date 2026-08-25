package cache

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/gombit-dev/gombit/config"
)

type recordingCache struct {
	keys []string
}

func (r *recordingCache) Get(_ context.Context, key string, _ any) (bool, error) {
	r.keys = append(r.keys, key)
	return false, nil
}

func (r *recordingCache) Set(_ context.Context, key string, _ any, _ time.Duration) error {
	r.keys = append(r.keys, key)
	return nil
}

func (r *recordingCache) Delete(_ context.Context, keys ...string) error {
	r.keys = append(r.keys, keys...)
	return nil
}

func (r *recordingCache) Increment(_ context.Context, key string, _ int64) (int64, error) {
	r.keys = append(r.keys, key)
	return 0, nil
}

func TestNamespacePrefixesCallerKeysAsIs(t *testing.T) {
	inner := &recordingCache{}
	c := Namespace("gombit:test", inner)
	ctx := context.Background()

	if err := c.Set(ctx, "foo", "a", 0); err != nil {
		t.Fatalf("Set(foo) error = %v, want nil", err)
	}
	if err := c.Set(ctx, ":foo", "b", 0); err != nil {
		t.Fatalf("Set(:foo) error = %v, want nil", err)
	}
	if err := c.Set(ctx, "::foo", "c", 0); err != nil {
		t.Fatalf("Set(::foo) error = %v, want nil", err)
	}
	if _, err := c.Get(ctx, "foo", nil); err != nil {
		t.Fatalf("Get(foo) error = %v, want nil", err)
	}
	if err := c.Delete(ctx, "foo", ":foo"); err != nil {
		t.Fatalf("Delete() error = %v, want nil", err)
	}
	if _, err := c.Increment(ctx, ":foo", 1); err != nil {
		t.Fatalf("Increment(:foo) error = %v, want nil", err)
	}

	want := []string{
		"gombit:test:foo",
		"gombit:test::foo",
		"gombit:test:::foo",
		"gombit:test:foo",
		"gombit:test:foo",
		"gombit:test::foo",
		"gombit:test::foo",
	}
	if !reflect.DeepEqual(inner.keys, want) {
		t.Fatalf("namespaced keys = %#v, want %#v", inner.keys, want)
	}
}

func TestNamespaceDoesNotCollapseKeysDifferingByLeadingColons(t *testing.T) {
	ctx := context.Background()
	cfg := config.Default().Cache
	cfg.Driver = config.CacheDriverMemory
	cfg.Namespace = "gombit:test"

	store, err := Open(cfg)
	if err != nil {
		t.Fatalf("Open() error = %v, want nil", err)
	}

	if err := store.Set(ctx, "foo", "a", 0); err != nil {
		t.Fatalf("Set(foo) error = %v, want nil", err)
	}
	if err := store.Set(ctx, ":foo", "b", 0); err != nil {
		t.Fatalf("Set(:foo) error = %v, want nil", err)
	}

	var got string
	found, err := store.Get(ctx, "foo", &got)
	if err != nil {
		t.Fatalf("Get(foo) error = %v, want nil", err)
	}
	if !found {
		t.Fatal("Get(foo) found = false, want true")
	}
	if got != "a" {
		t.Fatalf("Get(foo) = %q, want %q (Set(\":foo\") must not overwrite foo)", got, "a")
	}

	found, err = store.Get(ctx, ":foo", &got)
	if err != nil {
		t.Fatalf("Get(:foo) error = %v, want nil", err)
	}
	if !found {
		t.Fatal("Get(:foo) found = false, want true")
	}
	if got != "b" {
		t.Fatalf("Get(:foo) = %q, want %q", got, "b")
	}

	if err := store.Delete(ctx, ":foo"); err != nil {
		t.Fatalf("Delete(:foo) error = %v, want nil", err)
	}
	found, err = store.Get(ctx, "foo", &got)
	if err != nil {
		t.Fatalf("Get(foo) after Delete(:foo) error = %v, want nil", err)
	}
	if !found || got != "a" {
		t.Fatalf("Get(foo) after Delete(:foo) = (%t, %q), want (true, a)", found, got)
	}
}
