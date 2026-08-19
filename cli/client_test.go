package cli

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestClientCheckURLFetchesLiveSpecInsteadOfSampleApp exercises the path a
// generated app uses: gombit is a separately compiled binary with no
// Go-level access to the app's huma.API, so `client check --url` must fetch
// the live /openapi.json and compare/write against that, never against the
// framework's own SampleApp.
func TestClientCheckURLFetchesLiveSpecInsteadOfSampleApp(t *testing.T) {
	const liveSpec = `{"openapi":"3.1.0","info":{"title":"generated-app-from-url-test","version":"0.0.0"},"paths":{}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(liveSpec))
	}))
	defer srv.Close()

	dir := t.TempDir()
	t.Chdir(dir)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	err := ExecuteRoot(context.Background(), NewRoot(stdout, stderr), []string{
		"client", "check", "--write", "--url", srv.URL,
	})
	if err != nil {
		t.Fatalf("gombit client check --write --url: %v\nstderr: %s", err, stderr.String())
	}

	// #nosec G304 -- test-controlled path under t.TempDir()
	written, err := os.ReadFile(filepath.Join(dir, "openapi.json"))
	if err != nil {
		t.Fatalf("read written spec: %v", err)
	}
	if !strings.Contains(string(written), "generated-app-from-url-test") {
		t.Fatalf("written spec = %s, want content fetched from --url", written)
	}
}

func TestClientCheckDefaultsMatchGenerateNotExamplesClient(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	err := ExecuteRoot(context.Background(), NewRoot(stdout, stderr), []string{"client", "check", "--help"})
	if err != nil {
		t.Fatalf("gombit client check --help: %v", err)
	}
	out := stdout.String() + stderr.String()
	if strings.Contains(out, `default "examples/client`) {
		t.Fatalf("client check --help still defaults to framework-repo-only examples/client paths:\n%s", out)
	}
	if !strings.Contains(out, `default "openapi.json"`) {
		t.Fatalf("client check --help missing openapi.json default:\n%s", out)
	}
	if !strings.Contains(out, `default "frontend/src/api/generated"`) {
		t.Fatalf("client check --help missing frontend/src/api/generated default:\n%s", out)
	}
	if !strings.Contains(out, "--url") {
		t.Fatalf("client check --help missing --url flag:\n%s", out)
	}
}
