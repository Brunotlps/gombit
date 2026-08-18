package adminui

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmbedContainsIndexHTML(t *testing.T) {
	t.Parallel()
	info, err := fs.Stat(FS(), "index.html")
	if err != nil {
		t.Fatalf("FS() missing index.html: %v", err)
	}
	if info.IsDir() {
		t.Fatal("FS() index.html is a directory")
	}
	data, err := fs.ReadFile(FS(), "index.html")
	if err != nil {
		t.Fatalf("ReadFile index.html: %v", err)
	}
	body := strings.ToLower(string(data))
	if !strings.Contains(body, "<div id=\"root\">") && !strings.Contains(body, "<div id='root'>") {
		t.Fatalf("index.html missing #root: %s", truncate(body, 200))
	}
}

func TestSourceHasNoWebStorage(t *testing.T) {
	t.Parallel()
	var files []string
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "node_modules", "dist":
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".ts", ".tsx", ".js", ".jsx", ".mjs", ".html", ".json":
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk source: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no frontend source files under internal/adminui")
	}
	for _, path := range files {
		data, readErr := os.ReadFile(path) // #nosec G304 -- committed package source collected above
		if readErr != nil {
			t.Fatalf("read %s: %v", path, readErr)
		}
		lower := strings.ToLower(string(data))
		if strings.Contains(lower, "localstorage") || strings.Contains(lower, "sessionstorage") {
			t.Errorf("%s contains localStorage/sessionStorage", path)
		}
	}
}

func TestCookieClientInvariants(t *testing.T) {
	t.Parallel()
	client, err := os.ReadFile(filepath.Join("src", "api", "client.ts"))
	if err != nil {
		t.Fatalf("read client.ts: %v", err)
	}
	src := string(client)
	for _, want := range []string{
		"refreshInFlight",
		"X-CSRF-Token",
		"credentials: \"same-origin\"",
		"VITE_API_URL",
		"/api/v1/auth/csrf",
		"/api/v1/auth/login",
		"/api/v1/admin/meta",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("client.ts missing %q", want)
		}
	}
	if strings.Contains(src, "localStorage") || strings.Contains(src, "sessionStorage") {
		t.Error("client.ts uses web storage")
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
