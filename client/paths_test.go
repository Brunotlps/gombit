package client

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRewriteSpecPathsForTypedClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		paths     []string
		wantPaths []string
		wantSame  bool
	}{
		{
			name:      "already typed prefix",
			paths:     []string{"/api/v1/widgets", "/api/v1/widgets/{id}"},
			wantPaths: []string{"/api/v1/widgets", "/api/v1/widgets/{id}"},
			wantSame:  true,
		},
		{
			name:      "custom prefix one resource",
			paths:     []string{"/svc/v2/widgets", "/svc/v2/widgets/{id}"},
			wantPaths: []string{"/api/v1/widgets", "/api/v1/widgets/{id}"},
		},
		{
			name:      "api v2 with auth and resource",
			paths:     []string{"/api/v2/auth/login", "/api/v2/me", "/api/v2/products"},
			wantPaths: []string{"/api/v1/auth/login", "/api/v1/me", "/api/v1/products"},
		},
		{
			name:      "single detail path",
			paths:     []string{"/svc/v2/widgets/{id}"},
			wantPaths: []string{"/api/v1/widgets/{id}"},
		},
		{
			name:     "empty paths object",
			paths:    nil,
			wantSame: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			spec := specWithPaths(t, tt.paths)
			got, err := rewriteSpecPathsForTypedClient(spec)
			if err != nil {
				t.Fatalf("rewriteSpecPathsForTypedClient() error = %v", err)
			}
			if tt.wantSame && string(got) != string(spec) {
				t.Fatal("identity rewrite must return the original spec bytes")
			}
			gotKeys := pathKeys(t, got)
			if tt.wantPaths == nil {
				if len(gotKeys) != 0 {
					t.Fatalf("path keys = %v, want empty", gotKeys)
				}
				return
			}
			if !sameStringSet(gotKeys, tt.wantPaths) {
				t.Fatalf("path keys = %v, want %v", gotKeys, tt.wantPaths)
			}
			for _, key := range gotKeys {
				if strings.HasPrefix(key, "/svc/v2") {
					t.Fatalf("rewritten spec still has live prefix key %q", key)
				}
			}
		})
	}
}

func TestInferLiveAPIPrefix(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		paths []string
		want  string
	}{
		{name: "typed", paths: []string{"/api/v1/widgets", "/api/v1/me"}, want: "/api/v1"},
		{name: "svc v2", paths: []string{"/svc/v2/widgets", "/svc/v2/widgets/{id}"}, want: "/svc/v2"},
		{name: "single detail", paths: []string{"/svc/v2/widgets/{id}"}, want: "/svc/v2"},
		{name: "auth only", paths: []string{"/api/v2/auth/login", "/api/v2/auth/refresh"}, want: "/api/v2/auth"},
		{name: "auth and resource", paths: []string{"/api/v2/auth/login", "/api/v2/me", "/api/v2/products"}, want: "/api/v2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := inferLiveAPIPrefix(tt.paths); got != tt.want {
				t.Fatalf("inferLiveAPIPrefix(%v) = %q, want %q", tt.paths, got, tt.want)
			}
		})
	}
}

func specWithPaths(t *testing.T, paths []string) []byte {
	t.Helper()
	doc := map[string]any{
		"openapi": "3.1.0",
		"info":    map[string]any{"title": "t", "version": "0"},
		"paths":   map[string]any{},
	}
	pathMap := doc["paths"].(map[string]any)
	for _, path := range paths {
		pathMap[path] = map[string]any{"get": map[string]any{"operationId": "op"}}
	}
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return data
}

func pathKeys(t *testing.T, spec []byte) []string {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(spec, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	raw, _ := doc["paths"].(map[string]any)
	keys := make([]string, 0, len(raw))
	for path := range raw {
		keys = append(keys, path)
	}
	return keys
}

func sameStringSet(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	seen := make(map[string]int, len(want))
	for _, w := range want {
		seen[w]++
	}
	for _, g := range got {
		seen[g]--
		if seen[g] < 0 {
			return false
		}
	}
	return true
}
