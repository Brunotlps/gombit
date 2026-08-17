package goldentest

import (
	"bytes"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

var gombitRequirePattern = regexp.MustCompile(`(?m)^(require\s+github\.com/LAA-Software-Engineering/gombit\s+)v\S+`)

const (
	goldenRoot     = "testdata/golden"
	gombitModule   = "github.com/LAA-Software-Engineering/gombit"
	fixtureName    = "demo"
	fixtureModule  = "github.com/example/demo"
	fixtureBook    = "Book"
	fixtureFields  = "title:string:required"
	missingAtlas   = "gombit-golden-atlas-not-installed"
	maxDiffPreview = 80
)

var skipSnapshotDirs = map[string]struct{}{
	"node_modules": {},
	".git":         {},
	".gombit":      {},
	".vite":        {},
	"dist":         {},
}

type fileMap map[string][]byte

func moduleRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), ".."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("module root %s: %v", root, err)
	}
	return root
}

func goldenDir(name string) string {
	return filepath.Join(goldenRoot, name)
}

func snapshotTree(t *testing.T, root string) fileMap {
	t.Helper()
	fsys := os.DirFS(root)
	out := make(fileMap)
	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if _, skip := skipSnapshotDirs[d.Name()]; skip {
				return fs.SkipDir
			}
			return nil
		}
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		rel := filepath.ToSlash(path)
		out[rel] = normalizeContent(rel, data)
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot %s: %v", root, err)
	}
	if len(out) == 0 {
		t.Fatalf("snapshot %s: no files", root)
	}
	return out
}

func normalizeContent(rel string, data []byte) []byte {
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	if filepath.Base(rel) == "go.mod" {
		// Keep goldens free of version churn; compile tests pin the local module
		// with a replace in a temp copy, never in committed trees.
		data = gombitRequirePattern.ReplaceAll(data, []byte("${1}v0.0.0"))
	}
	return data
}

func assertNoReplace(t *testing.T, files fileMap) {
	t.Helper()
	for rel, data := range files {
		if filepath.Base(rel) != "go.mod" {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "replace ") {
				t.Errorf("%s contains a replace directive; goldens must not bake machine-specific paths", rel)
			}
		}
	}
}

func loadGolden(t *testing.T, name string) fileMap {
	t.Helper()
	dir := goldenDir(name)
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("missing golden %s (%s); run go test ./goldentest -update", name, dir)
	}
	return snapshotTree(t, dir)
}

func writeGolden(t *testing.T, name string, files fileMap) {
	t.Helper()
	dir := goldenDir(name)
	if err := os.RemoveAll(dir); err != nil {
		t.Fatalf("remove golden %s: %v", dir, err)
	}
	rels := make([]string, 0, len(files))
	for rel := range files {
		rels = append(rels, rel)
	}
	sort.Strings(rels)
	for _, rel := range rels {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(path, files[rel], 0o600); err != nil {
			t.Fatalf("write golden %s: %v", path, err)
		}
	}
}

func compareTrees(t *testing.T, got, want fileMap) {
	t.Helper()
	var extra, missing []string
	for rel := range got {
		if _, ok := want[rel]; !ok {
			extra = append(extra, rel)
		}
	}
	for rel := range want {
		if _, ok := got[rel]; !ok {
			missing = append(missing, rel)
		}
	}
	sort.Strings(extra)
	sort.Strings(missing)
	if len(extra) > 0 {
		t.Errorf("generated tree has unexpected files: %s", strings.Join(extra, ", "))
	}
	if len(missing) > 0 {
		t.Errorf("generated tree is missing files: %s", strings.Join(missing, ", "))
	}
	var diffs []string
	for rel, wantData := range want {
		gotData, ok := got[rel]
		if !ok {
			continue
		}
		if bytes.Equal(gotData, wantData) {
			continue
		}
		diffs = append(diffs, rel+": "+describeDiff(gotData, wantData))
	}
	sort.Strings(diffs)
	for _, d := range diffs {
		t.Error(d)
	}
}

func describeDiff(got, want []byte) string {
	gotLines := strings.Split(string(got), "\n")
	wantLines := strings.Split(string(want), "\n")
	n := min(len(gotLines), len(wantLines))
	for i := 0; i < n; i++ {
		if gotLines[i] == wantLines[i] {
			continue
		}
		return fmt.Sprintf("line %d:\n  got:  %s\n  want: %s", i+1, preview(gotLines[i]), preview(wantLines[i]))
	}
	if len(gotLines) != len(wantLines) {
		return fmt.Sprintf("line count got %d want %d", len(gotLines), len(wantLines))
	}
	return fmt.Sprintf("bytes got %d want %d", len(got), len(want))
}

func preview(s string) string {
	if len(s) > maxDiffPreview {
		return fmt.Sprintf("%q...", s[:maxDiffPreview])
	}
	return fmt.Sprintf("%q", s)
}

func treesEqual(a, b fileMap) bool {
	if len(a) != len(b) {
		return false
	}
	for rel, data := range a {
		other, ok := b[rel]
		if !ok || !bytes.Equal(data, other) {
			return false
		}
	}
	return true
}

func treeDiffSummary(a, b fileMap) string {
	var parts []string
	for rel := range a {
		if _, ok := b[rel]; !ok {
			parts = append(parts, "- extra "+rel)
			continue
		}
		if !bytes.Equal(a[rel], b[rel]) {
			parts = append(parts, "- changed "+rel)
		}
	}
	for rel := range b {
		if _, ok := a[rel]; !ok {
			parts = append(parts, "- missing "+rel)
		}
	}
	sort.Strings(parts)
	return strings.Join(parts, "\n")
}

func assertGoFormatted(t *testing.T, files fileMap) {
	t.Helper()
	for rel, data := range files {
		if !strings.HasSuffix(rel, ".go") {
			continue
		}
		formatted, err := format.Source(data)
		if err != nil {
			t.Errorf("%s is not valid Go: %v", rel, err)
			continue
		}
		if !bytes.Equal(data, formatted) {
			t.Errorf("%s is not gofmt-clean", rel)
		}
	}
}

func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	if err := os.MkdirAll(dst, 0o750); err != nil {
		t.Fatalf("mkdir %s: %v", dst, err)
	}
	if err := os.CopyFS(dst, os.DirFS(src)); err != nil {
		t.Fatalf("copy %s -> %s: %v", src, dst, err)
	}
}
