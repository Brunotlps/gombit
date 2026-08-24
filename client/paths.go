package client

import (
	"encoding/json"
	"fmt"
	"strings"
)

// TypedAPIPrefix is D8 / DEFAULT_API_PREFIX. Scaffolded pages and the
// placeholder OpenAPI client use these path keys. gombit client generate
// rewrites live Huma paths to this prefix before openapi-typescript so
// `client.GET("/api/v1/...")` keeps typechecking after GOMBIT_API_PREFIX
// changes. rewriteAPIRequest maps them back to the live prefix on the wire.
const TypedAPIPrefix = "/api/v1"

// rewriteSpecPathsForTypedClient copies spec and rewrites OpenAPI path keys
// from the live API prefix to TypedAPIPrefix. The caller's document on disk
// is not modified (examples/client contract-drift still compares live Huma
// paths). A spec that already uses /api/v1 is returned unchanged.
func rewriteSpecPathsForTypedClient(spec []byte) ([]byte, error) {
	var doc map[string]any
	if err := json.Unmarshal(spec, &doc); err != nil {
		return nil, fmt.Errorf("client: parse spec for path rewrite: %w", err)
	}
	rawPaths, ok := doc["paths"].(map[string]any)
	if !ok || len(rawPaths) == 0 {
		return spec, nil
	}
	keys := make([]string, 0, len(rawPaths))
	for path := range rawPaths {
		keys = append(keys, path)
	}
	live := inferLiveAPIPrefix(keys)
	if live == "" || live == TypedAPIPrefix {
		return spec, nil
	}

	rewritten := make(map[string]any, len(rawPaths))
	for path, value := range rawPaths {
		newPath := path
		if path == live || strings.HasPrefix(path, live+"/") {
			newPath = TypedAPIPrefix + path[len(live):]
		}
		rewritten[newPath] = value
	}
	doc["paths"] = rewritten
	out, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("client: marshal rewritten spec: %w", err)
	}
	return out, nil
}

// inferLiveAPIPrefix returns the Huma mount prefix shared by every path.
// Longest common string prefix is walked back to a slash boundary until no
// operation path equals the candidate and every remainder starts with a
// non-parameter segment (so /svc/v2/widgets + /svc/v2/widgets/{id} yields
// /svc/v2, not /svc/v2/widgets).
func inferLiveAPIPrefix(paths []string) string {
	if len(paths) == 0 {
		return TypedAPIPrefix
	}
	if allTypedAPIPrefix(paths) {
		return TypedAPIPrefix
	}
	candidate := paths[0]
	for _, path := range paths[1:] {
		candidate = commonStringPrefix(candidate, path)
	}
	for {
		candidate = strings.TrimSuffix(candidate, "/")
		if candidate == "" || candidate == "/" {
			return TypedAPIPrefix
		}
		if isValidAPIPrefix(candidate, paths) {
			return candidate
		}
		slash := strings.LastIndex(candidate, "/")
		if slash <= 0 {
			return TypedAPIPrefix
		}
		candidate = candidate[:slash]
	}
}

func allTypedAPIPrefix(paths []string) bool {
	for _, path := range paths {
		if path != TypedAPIPrefix && !strings.HasPrefix(path, TypedAPIPrefix+"/") {
			return false
		}
	}
	return true
}

func isValidAPIPrefix(prefix string, paths []string) bool {
	if prefix == "" || strings.Contains(prefix, "{") {
		return false
	}
	for _, path := range paths {
		if path == prefix {
			return false
		}
		if !strings.HasPrefix(path, prefix+"/") {
			return false
		}
		rest := strings.TrimPrefix(path, prefix+"/")
		seg, _, _ := strings.Cut(rest, "/")
		if seg == "" || strings.HasPrefix(seg, "{") {
			return false
		}
	}
	return true
}

func commonStringPrefix(a, b string) string {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	i := 0
	for i < n && a[i] == b[i] {
		i++
	}
	return a[:i]
}
