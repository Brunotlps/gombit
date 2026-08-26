//go:build integration

package project_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gombit-dev/gombit/benchmarks/apps/gombit/internal/project"
	"github.com/gombit-dev/gombit/benchmarks/apps/shared"
	"github.com/gombit-dev/gombit/config"
	"github.com/gombit-dev/gombit/database"
	"github.com/gombit-dev/gombit/framework"
	"go.uber.org/zap"
	"gorm.io/gorm/logger"
)

// Run against a real PostgreSQL instance with the Atlas migration already
// applied (this package doesn't run `atlas migrate apply` itself — that's a
// deploy-time step, not a test-time one, matching how a real Gombit app is
// operated):
//
//	docker compose -f benchmarks/compose.yml up -d postgres
//	createdb -h 127.0.0.1 -p 55432 -U gombit gombit_bench_gombit   # once
//	(cd benchmarks/apps/gombit && \
//	  GOMBIT_DATABASE_DRIVER=postgres \
//	  GOMBIT_DATABASE_DSN="postgres://gombit:gombit@127.0.0.1:55432/gombit_bench_gombit?sslmode=disable" \
//	  go run github.com/gombit-dev/gombit/cmd/gombit db migrate)
//	go test -tags integration ./benchmarks/apps/gombit/... \
//	  -database.dsn "postgres://gombit:gombit@127.0.0.1:55432/gombit_bench_gombit?sslmode=disable"
var databaseDSN = flag.String("database.dsn", "", "PostgreSQL DSN for gombit app integration tests (schema already migrated)")

func newProjectApp(t *testing.T) *framework.App {
	t.Helper()

	if *databaseDSN == "" {
		t.Skip("set -database.dsn to run gombit app integration tests")
	}

	db, err := database.Open(config.DatabaseConfig{
		Driver: config.DatabaseDriverPostgres,
		DSN:    *databaseDSN,
	})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.Exec("TRUNCATE TABLE projects, users RESTART IDENTITY CASCADE").Error; err != nil {
		t.Fatalf("truncate (is the Atlas migration applied? see this file's doc comment): %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	cfg := config.DefaultFor(config.EnvironmentTest)
	cfg.HTTP.Addr = "127.0.0.1:0"
	cfg.API.Prefix = "/api" // matches main.go; see its comment
	app, err := framework.New(
		framework.WithConfig(cfg),
		framework.WithDatabase(db),
		framework.WithLogger(zap.NewNop()),
	)
	if err != nil {
		t.Fatalf("build app: %v", err)
	}
	project.Register(app)
	return app
}

func doJSON(t *testing.T, app *framework.App, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	var request *http.Request
	if body == "" {
		request = httptest.NewRequest(method, path, nil)
	} else {
		request = httptest.NewRequest(method, path, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	app.Router().ServeHTTP(response, request)
	return response
}

func decodeData(t *testing.T, body []byte, into *shared.ProjectData) {
	t.Helper()
	var envelope struct {
		Data shared.ProjectData `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode data envelope: %v; body: %s", err, body)
	}
	*into = envelope.Data
}

// TestCRUDRoundTrip exercises create -> get -> update -> delete -> get (404)
// against a real PostgreSQL instance, mirroring
// benchmarks/apps/gin-gorm/main_test.go's TestCRUDRoundTrip so the two
// suites cover the same contract.
func TestCRUDRoundTrip(t *testing.T) {
	app := newProjectApp(t)
	if err := app.DB().Create(&project.User{Email: "owner@example.com", Name: "Owner"}).Error; err != nil {
		t.Fatalf("seed owner: %v", err)
	}

	create := doJSON(t, app, http.MethodPost, "/api/projects", `{"owner_id":1,"name":"Test Project","description":"desc"}`)
	if create.Code != http.StatusCreated {
		t.Fatalf("POST /api/projects status = %d, want %d; body: %s", create.Code, http.StatusCreated, create.Body.String())
	}
	var created shared.ProjectData
	decodeData(t, create.Body.Bytes(), &created)
	if created.Name != "Test Project" || created.OwnerName != "Owner" {
		t.Fatalf("created project = %+v, want Name=Test Project OwnerName=Owner", created)
	}

	get := doJSON(t, app, http.MethodGet, fmt.Sprintf("/api/projects/%d", created.ID), "")
	if get.Code != http.StatusOK {
		t.Fatalf("GET /api/projects/%d status = %d, want %d", created.ID, get.Code, http.StatusOK)
	}

	update := doJSON(t, app, http.MethodPatch, fmt.Sprintf("/api/projects/%d", created.ID), `{"name":"Renamed"}`)
	if update.Code != http.StatusOK {
		t.Fatalf("PATCH status = %d, want %d; body: %s", update.Code, http.StatusOK, update.Body.String())
	}
	var updated shared.ProjectData
	decodeData(t, update.Body.Bytes(), &updated)
	if updated.Name != "Renamed" || updated.Description != "desc" {
		t.Fatalf("updated project = %+v, want Name=Renamed Description=desc (unchanged)", updated)
	}
	if updated.UpdatedAt.Before(created.UpdatedAt) {
		t.Fatalf("updated.UpdatedAt = %v, want not before created.UpdatedAt = %v", updated.UpdatedAt, created.UpdatedAt)
	}

	del := doJSON(t, app, http.MethodDelete, fmt.Sprintf("/api/projects/%d", created.ID), "")
	if del.Code != http.StatusOK {
		t.Fatalf("DELETE status = %d, want %d", del.Code, http.StatusOK)
	}

	getAfterDelete := doJSON(t, app, http.MethodGet, fmt.Sprintf("/api/projects/%d", created.ID), "")
	if getAfterDelete.Code != http.StatusNotFound {
		t.Fatalf("GET after delete status = %d, want %d", getAfterDelete.Code, http.StatusNotFound)
	}
}

// TestCreateRejectsBlankName and TestUpdateRejectsBlankName check the same
// blank/whitespace-only name rule benchmarks/apps/gin-gorm enforces,
// checked symmetrically here from the start (see handler.go's
// blankNameError). Table-driven over both the empty string (which Huma's
// own minLength:"1" already rejects, verified separately) and a
// whitespace-only string (which minLength alone does not catch), so the
// two rejection paths can't silently diverge from each other the way
// gin-gorm's create/update once did.
var blankNames = []string{"", "   "}

func TestCreateRejectsBlankName(t *testing.T) {
	app := newProjectApp(t)
	if err := app.DB().Create(&project.User{Email: "owner@example.com", Name: "Owner"}).Error; err != nil {
		t.Fatalf("seed owner: %v", err)
	}

	for _, name := range blankNames {
		t.Run(fmt.Sprintf("name=%q", name), func(t *testing.T) {
			body := fmt.Sprintf(`{"owner_id":1,"name":%q,"description":"x"}`, name)
			response := doJSON(t, app, http.MethodPost, "/api/projects", body)
			if response.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusUnprocessableEntity, response.Body.String())
			}
		})
	}
}

func TestUpdateRejectsBlankName(t *testing.T) {
	app := newProjectApp(t)
	if err := app.DB().Create(&project.User{Email: "owner@example.com", Name: "Owner"}).Error; err != nil {
		t.Fatalf("seed owner: %v", err)
	}
	create := doJSON(t, app, http.MethodPost, "/api/projects", `{"owner_id":1,"name":"Original","description":"desc"}`)
	var created shared.ProjectData
	decodeData(t, create.Body.Bytes(), &created)

	for _, name := range blankNames {
		t.Run(fmt.Sprintf("name=%q", name), func(t *testing.T) {
			body := fmt.Sprintf(`{"name":%q}`, name)
			response := doJSON(t, app, http.MethodPatch, fmt.Sprintf("/api/projects/%d", created.ID), body)
			if response.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusUnprocessableEntity, response.Body.String())
			}
		})
	}

	// A rejected update must not have partially applied.
	get := doJSON(t, app, http.MethodGet, fmt.Sprintf("/api/projects/%d", created.ID), "")
	var unchanged shared.ProjectData
	decodeData(t, get.Body.Bytes(), &unchanged)
	if unchanged.Name != "Original" {
		t.Fatalf("project name after rejected PATCH = %q, want unchanged %q", unchanged.Name, "Original")
	}
}

// TestCreateInvalidOwnerIDReturnsInternalError documents, rather than hides,
// a real Gombit framework gap discovered while building this app:
// database.MapPersistError (github.com/gombit-dev/gombit/database) only
// special-cases unique-constraint violations, so a foreign-key violation —
// exactly what a nonexistent owner_id produces — falls through to internal
// (500). benchmarks/apps/gin-gorm's control implementation does not use
// that framework helper and maps the same input to 422 (see its
// TestCreateRejectsInvalidOwnerID); this app uses Gombit's normal public
// APIs unmodified (issue #141: "do not bypass ... normal Gombit response
// handling"), so it inherits the gap. This test pins the framework's actual
// current behavior so a silent fix (or regression) shows up here, not just
// as an unexplained fairness-check failure when Phase 3b's
// cross-implementation checks run. See
// docs/plans/BENCH-1-benchmark-suite.md Phase 3b for the discovered-gap
// writeup and why it's out of scope to fix as part of BENCH-1.
func TestCreateInvalidOwnerIDReturnsInternalError(t *testing.T) {
	app := newProjectApp(t)

	response := doJSON(t, app, http.MethodPost, "/api/projects", `{"owner_id":999999,"name":"Orphan","description":"no such owner"}`)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("POST with nonexistent owner_id status = %d, want %d (see test doc comment); body: %s",
			response.Code, http.StatusInternalServerError, response.Body.String())
	}
}

// TestCreateRejectsZeroOwnerID checks a different input than
// TestCreateInvalidOwnerIDReturnsInternalError above: owner_id:0 is not "a
// nonexistent but well-formed id" (999999), it's the zero value — the same
// thing gin-gorm's binding:"required" on OwnerID rejects at the validation
// layer before it ever reaches GORM. Without minimum:"1" on
// createProjectBody.OwnerID, this case fell through to the same
// FK-violation 500 as the nonexistent-id case, conflating a validation gap
// this app can close with the discovered framework gap it deliberately
// doesn't.
func TestCreateRejectsZeroOwnerID(t *testing.T) {
	app := newProjectApp(t)

	response := doJSON(t, app, http.MethodPost, "/api/projects", `{"owner_id":0,"name":"x","description":"y"}`)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("POST with owner_id:0 status = %d, want %d; body: %s",
			response.Code, http.StatusUnprocessableEntity, response.Body.String())
	}
}

// TestListPaginationAndOrdering mirrors
// benchmarks/apps/gin-gorm/main_test.go's test of the same name: pagination
// meta shape, deterministic id-DESC ordering across a page boundary, and
// that the owner relationship is actually populated.
func TestListPaginationAndOrdering(t *testing.T) {
	app := newProjectApp(t)
	seedFixture(t, app, 3, 25)

	first := doJSON(t, app, http.MethodGet, "/api/projects?page=1&limit=20", "")
	if first.Code != http.StatusOK {
		t.Fatalf("GET page 1 status = %d, want %d", first.Code, http.StatusOK)
	}
	var page1 struct {
		Data []shared.ProjectData `json:"data"`
		Meta shared.PageMeta      `json:"meta"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &page1); err != nil {
		t.Fatalf("decode page 1: %v; body: %s", err, first.Body.String())
	}
	if page1.Meta != (shared.PageMeta{Page: 1, Limit: 20, Total: 25}) {
		t.Fatalf("page 1 meta = %+v, want {Page:1 Limit:20 Total:25}", page1.Meta)
	}
	if len(page1.Data) != 20 {
		t.Fatalf("page 1 len(data) = %d, want 20", len(page1.Data))
	}
	if page1.Data[0].ID <= page1.Data[1].ID {
		t.Fatalf("page 1 not descending: data[0].ID=%d data[1].ID=%d", page1.Data[0].ID, page1.Data[1].ID)
	}
	for _, row := range page1.Data {
		if row.OwnerName == "" {
			t.Fatalf("project %d has empty OwnerName; owner preload did not populate it", row.ID)
		}
	}

	second := doJSON(t, app, http.MethodGet, "/api/projects?page=2&limit=20", "")
	var page2 struct {
		Data []shared.ProjectData `json:"data"`
		Meta shared.PageMeta      `json:"meta"`
	}
	if err := json.Unmarshal(second.Body.Bytes(), &page2); err != nil {
		t.Fatalf("decode page 2: %v; body: %s", err, second.Body.String())
	}
	if len(page2.Data) != 5 {
		t.Fatalf("page 2 len(data) = %d, want 5 (25 total - 20 on page 1)", len(page2.Data))
	}
	if page1.Data[19].ID <= page2.Data[0].ID {
		t.Fatalf("page boundary not descending: page1[19].ID=%d page2[0].ID=%d", page1.Data[19].ID, page2.Data[0].ID)
	}
}

// TestListDoesNotN1 and TestListDoesNotN1EmptyPage mirror
// benchmarks/apps/gin-gorm/main_test.go's tests of the same name: the list
// endpoint issues exactly 3 SQL statements for a non-empty page (count,
// page, one batched owner IN (...)) and 2 for an empty one — see
// benchmarks/docs/schema.md "Canonical CRUD API" for the pinned shape. The
// counting logger is attached by mutating app.DB()'s shared *gorm.Config
// (gorm.DB embeds *Config, so this affects every query issued through this
// app instance) rather than by constructing a second database.DB, since
// database.DB's driver/capabilities fields aren't exported for a test in
// another package to set independently.
func TestListDoesNotN1(t *testing.T) {
	app := newProjectApp(t)
	seedFixture(t, app, 5, 20)

	counter := &queryCounter{}
	app.DB().Logger = counter

	response := doJSON(t, app, http.MethodGet, "/api/projects?page=1&limit=20", "")
	if response.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if got := counter.Count(); got != 3 {
		t.Fatalf("list issued %d SQL statements, want exactly 3 (count + page + batched owner preload); got query log: %v", got, counter.Queries())
	}
}

func TestListDoesNotN1EmptyPage(t *testing.T) {
	app := newProjectApp(t)
	seedFixture(t, app, 5, 20)

	counter := &queryCounter{}
	app.DB().Logger = counter

	response := doJSON(t, app, http.MethodGet, "/api/projects?page=99&limit=20", "")
	if response.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if got := counter.Count(); got != 2 {
		t.Fatalf("empty-page list issued %d SQL statements, want exactly 2 (count + empty page, no owners to preload); got query log: %v", got, counter.Queries())
	}
}

// seedFixture inserts userCount users and projectCount projects (round-robin
// ownership) directly through GORM, independent of seed.go's full
// 1,000/100,000 dataset, so tests run in milliseconds — matching
// benchmarks/apps/gin-gorm/main_test.go's helper of the same name.
func seedFixture(t *testing.T, app *framework.App, userCount, projectCount int) {
	t.Helper()

	for i := 1; i <= userCount; i++ {
		if err := app.DB().Create(&project.User{Email: fmt.Sprintf("fixture-%d@example.com", i), Name: fmt.Sprintf("Fixture User %d", i)}).Error; err != nil {
			t.Fatalf("seed fixture user %d: %v", i, err)
		}
	}
	for i := 1; i <= projectCount; i++ {
		p := project.Project{
			OwnerID:     shared.ProjectOwnerID(i, userCount),
			Name:        fmt.Sprintf("Fixture Project %d", i),
			Description: "fixture",
		}
		if err := app.DB().Create(&p).Error; err != nil {
			t.Fatalf("seed fixture project %d: %v", i, err)
		}
	}
}

// queryCounter is a minimal gorm.Logger that counts SQL statements traced
// via Trace, guarded by a mutex — matching
// benchmarks/apps/gin-gorm/main_test.go's queryCounter exactly (same
// reasoning: GORM doesn't guarantee Trace runs on the caller's goroutine).
type queryCounter struct {
	mu      sync.Mutex
	count   int
	queries []string
}

func (q *queryCounter) LogMode(logger.LogLevel) logger.Interface      { return q }
func (q *queryCounter) Info(context.Context, string, ...interface{})  {}
func (q *queryCounter) Warn(context.Context, string, ...interface{})  {}
func (q *queryCounter) Error(context.Context, string, ...interface{}) {}
func (q *queryCounter) Trace(_ context.Context, _ time.Time, fc func() (string, int64), _ error) {
	sql, _ := fc()
	q.mu.Lock()
	defer q.mu.Unlock()
	q.count++
	q.queries = append(q.queries, sql)
}

func (q *queryCounter) Count() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.count
}

func (q *queryCounter) Queries() []string {
	q.mu.Lock()
	defer q.mu.Unlock()
	return append([]string(nil), q.queries...)
}
