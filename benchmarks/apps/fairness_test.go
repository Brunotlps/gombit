//go:build integration

// Package apps_test holds the cross-implementation fairness check (issue
// #141 §15): the same paginated record count, the same ordered IDs and
// content for a known query, and the same route surface for a not-found
// request. It builds and runs the real gin-gorm and gombit binaries against
// their own already-seeded databases and compares them over HTTP — not by
// importing their internals — because gin-gorm is package main (not
// importable) and gombit's routes only exist behind its own framework.App.
// This is also the only shape of comparison that will still work once
// Phase 4 adds Django/Rails/Laravel/NestJS: two Go packages sharing
// internals is not a pattern those can participate in.
package apps_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"testing"
	"time"

	"github.com/gombit-dev/gombit/benchmarks/apps/shared"
)

// Both databases must already be seeded with the canonical dataset
// (`go run ./benchmarks/apps/gin-gorm -seed` /
// `go run ./benchmarks/apps/gombit -seed`) and, for gombit, already
// Atlas-migrated (see benchmarks/apps/gombit/internal/project/handler_test.go's
// doc comment) — this test seeds nothing itself, matching real deployment
// order (migrate/seed is a deploy step, this is a runtime check).
var (
	ginGormDSN = flag.String("gin-gorm.dsn", "", "PostgreSQL DSN for the already-seeded gin-gorm app")
	gombitDSN  = flag.String("gombit.dsn", "", "PostgreSQL DSN for the already-migrated, already-seeded gombit app")
)

// TestCrossImplementationFairness is the Phase 3b fairness check
// docs/plans/BENCH-1-benchmark-suite.md describes: same paginated record
// count, same ordered IDs and content for a known query (page 1 of the
// canonical seed), and same route surface for a not-found id. Timestamps
// are excluded from the row comparison: each binary was seeded at a
// different real wall-clock time, so CreatedAt/UpdatedAt legitimately
// differ — content and ordering do not.
//
// The relative comparison (gin-gorm's page equals gombit's page) is
// necessary but not sufficient: two empty, unseeded databases satisfy it
// too, by both returning total=0/data=[]. assertCanonicalSeed pins each
// side against the actual canonical dataset — shared.SeedProjectCount,
// shared.ProjectName, shared.ProjectOwnerID, the id-DESC insert order —
// before the two are ever compared to each other, so an empty or
// wrongly-seeded pair fails here regardless of whether it happens to agree
// with itself.
func TestCrossImplementationFairness(t *testing.T) {
	if *ginGormDSN == "" || *gombitDSN == "" {
		t.Skip("set -gin-gorm.dsn and -gombit.dsn to run the cross-implementation fairness check")
	}

	ginGormAddr := startApp(t, "gin-gorm", "./gin-gorm", "18091", *ginGormDSN)
	gombitAddr := startApp(t, "gombit", "./gombit", "18090", *gombitDSN)

	ginGormPage := fetchPage(t, ginGormAddr, "/api/projects?page=1&limit=20")
	gombitPage := fetchPage(t, gombitAddr, "/api/projects?page=1&limit=20")

	assertCanonicalSeed(t, "gin-gorm", ginGormPage)
	assertCanonicalSeed(t, "gombit", gombitPage)

	if ginGormPage.Meta != gombitPage.Meta {
		t.Fatalf("page meta differs: gin-gorm=%+v gombit=%+v", ginGormPage.Meta, gombitPage.Meta)
	}
	if len(ginGormPage.Data) != len(gombitPage.Data) {
		t.Fatalf("page length differs: gin-gorm=%d gombit=%d", len(ginGormPage.Data), len(gombitPage.Data))
	}
	for i := range ginGormPage.Data {
		a, b := stripTimestamps(ginGormPage.Data[i]), stripTimestamps(gombitPage.Data[i])
		if a != b {
			t.Fatalf("row %d differs (timestamps excluded): gin-gorm=%+v gombit=%+v", i, a, b)
		}
	}

	// Same route surface: a nonexistent id 404s on both, not just one.
	for _, addr := range []string{ginGormAddr, gombitAddr} {
		if code := getStatus(t, addr, "/api/projects/999999999"); code != http.StatusNotFound {
			t.Fatalf("%s: GET nonexistent id status = %d, want %d", addr, code, http.StatusNotFound)
		}
	}
}

// assertCanonicalSeed checks one implementation's page-1 response against
// the actual canonical dataset (benchmarks/docs/schema.md), independent of
// what the other implementation returns — the oracle the relative
// comparison in TestCrossImplementationFairness alone can't provide.
func assertCanonicalSeed(t *testing.T, label string, page pageEnvelope) {
	t.Helper()

	if page.Meta.Total != shared.SeedProjectCount {
		t.Fatalf("%s: meta.total = %d, want %d (canonical seed size — is this database actually seeded?)",
			label, page.Meta.Total, shared.SeedProjectCount)
	}
	if len(page.Data) != 20 {
		t.Fatalf("%s: len(data) = %d, want 20 (page size)", label, len(page.Data))
	}

	// id DESC, sequential insert: page 1 starts at the last seeded project.
	last := shared.SeedProjectCount
	first := page.Data[0]
	if first.Name != shared.ProjectName(last) {
		t.Fatalf("%s: data[0].Name = %q, want %q (last seeded project — wrong seed content or wrong ordering)",
			label, first.Name, shared.ProjectName(last))
	}
	if first.Description != shared.ProjectDescription(last) {
		t.Fatalf("%s: data[0].Description = %q, want %q", label, first.Description, shared.ProjectDescription(last))
	}
	wantOwner := shared.ProjectOwnerID(last, shared.SeedUserCount)
	if first.OwnerID != wantOwner {
		t.Fatalf("%s: data[0].OwnerID = %d, want %d (round-robin owner of the last seeded project)",
			label, first.OwnerID, wantOwner)
	}
	if first.OwnerName != shared.UserName(int(wantOwner)) {
		t.Fatalf("%s: data[0].OwnerName = %q, want %q", label, first.OwnerName, shared.UserName(int(wantOwner)))
	}
}

func stripTimestamps(p shared.ProjectData) shared.ProjectData {
	p.CreatedAt, p.UpdatedAt = time.Time{}, time.Time{}
	return p
}

type pageEnvelope struct {
	Data []shared.ProjectData `json:"data"`
	Meta shared.PageMeta      `json:"meta"`
}

func fetchPage(t *testing.T, addr, path string) pageEnvelope {
	t.Helper()
	body := getBody(t, addr, path)
	var env pageEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("decode %s%s: %v; body: %s", addr, path, err, body)
	}
	return env
}

func getBody(t *testing.T, addr, path string) []byte {
	t.Helper()
	resp, err := http.Get("http://" + addr + path)
	if err != nil {
		t.Fatalf("GET %s%s: %v", addr, path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read GET %s%s body: %v", addr, path, err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s%s status = %d; body: %s", addr, path, resp.StatusCode, body)
	}
	return body
}

func getStatus(t *testing.T, addr, path string) int {
	t.Helper()
	resp, err := http.Get("http://" + addr + path)
	if err != nil {
		t.Fatalf("GET %s%s: %v", addr, path, err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

// startApp builds and runs one implementation's binary against dsn on port,
// waits for /livez, and returns its address. The process is killed via
// t.Cleanup.
func startApp(t *testing.T, name, pkgDir, port, dsn string) string {
	t.Helper()

	binPath := t.TempDir() + "/" + name
	build := exec.Command("go", "build", "-o", binPath, pkgDir)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build %s: %v\n%s", name, err, out)
	}

	cmd := exec.Command(binPath)
	// Appending, not prepending or stripping first, is deliberate and safe
	// even if the parent process already exports DATABASE_URL/PORT (both
	// apps document that env var name, so a caller's shell easily could):
	// os/exec.Cmd.Start dedupes cmd.Env before exec, keeping the *last*
	// occurrence of each key (os/exec/exec.go's dedupEnvCase, "Construct
	// the output in reverse order, to preserve the last occurrence of each
	// key") — verified directly, including with a real child process
	// reading via os.Getenv while the parent had DATABASE_URL set to a
	// different value. These appended values always win.
	cmd.Env = append(cmd.Environ(), "DATABASE_URL="+dsn, "PORT="+port)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start %s: %v", name, err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	addr := "127.0.0.1:" + port
	waitForHealth(t, name, addr)
	return addr
}

func waitForHealth(t *testing.T, name, addr string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("%s did not become healthy on %s within 10s", name, addr)
		default:
		}
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			resp, err := http.Get(fmt.Sprintf("http://%s/livez", addr))
			if err == nil {
				_ = resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}
