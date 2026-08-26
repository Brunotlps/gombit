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
func TestCrossImplementationFairness(t *testing.T) {
	if *ginGormDSN == "" || *gombitDSN == "" {
		t.Skip("set -gin-gorm.dsn and -gombit.dsn to run the cross-implementation fairness check")
	}

	ginGormAddr := startApp(t, "gin-gorm", "./gin-gorm", "18091", *ginGormDSN)
	gombitAddr := startApp(t, "gombit", "./gombit", "18090", *gombitDSN)

	ginGormPage := fetchPage(t, ginGormAddr, "/api/projects?page=1&limit=20")
	gombitPage := fetchPage(t, gombitAddr, "/api/projects?page=1&limit=20")

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
