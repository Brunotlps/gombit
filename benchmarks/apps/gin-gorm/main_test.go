//go:build integration

package main

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

	"github.com/gin-gonic/gin"
	"github.com/gombit-dev/gombit/benchmarks/apps/shared"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Run with a live PostgreSQL instance, e.g. the one benchmarks/compose.yml
// provides:
//
//	docker compose -f benchmarks/compose.yml up -d postgres
//	go test -tags integration ./benchmarks/apps/gin-gorm/... \
//	  -database.dsn "postgres://gombit:gombit@127.0.0.1:55432/gombit_bench?sslmode=disable"
var databaseDSN = flag.String("database.dsn", "", "PostgreSQL DSN for gin-gorm integration tests")

func testDB(t *testing.T) *gorm.DB {
	t.Helper()

	if *databaseDSN == "" {
		t.Skip("set -database.dsn to run gin-gorm integration tests")
	}

	db, err := gorm.Open(postgres.Open(*databaseDSN), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&User{}, &Project{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Exec("TRUNCATE TABLE projects, users RESTART IDENTITY CASCADE").Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func testRouter(t *testing.T, db *gorm.DB) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	return newRouter(db)
}

func doJSON(t *testing.T, router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	var request *http.Request
	if body == "" {
		request = httptest.NewRequest(method, path, nil)
	} else {
		request = httptest.NewRequest(method, path, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

// TestCRUDRoundTrip exercises create -> get -> update -> delete -> get (404)
// and one validation failure, against a real PostgreSQL instance.
func TestCRUDRoundTrip(t *testing.T) {
	db := testDB(t)
	if err := db.Create(&User{Email: "owner@example.com", Name: "Owner"}).Error; err != nil {
		t.Fatalf("seed owner: %v", err)
	}
	router := testRouter(t, db)

	create := doJSON(t, router, http.MethodPost, "/api/projects", `{"owner_id":1,"name":"Test Project","description":"desc"}`)
	if create.Code != http.StatusCreated {
		t.Fatalf("POST /api/projects status = %d, want %d; body: %s", create.Code, http.StatusCreated, create.Body.String())
	}
	var created shared.ProjectData
	decodeData(t, create.Body.Bytes(), &created)
	if created.Name != "Test Project" || created.OwnerName != "Owner" {
		t.Fatalf("created project = %+v, want Name=Test Project OwnerName=Owner", created)
	}

	get := doJSON(t, router, http.MethodGet, fmt.Sprintf("/api/projects/%d", created.ID), "")
	if get.Code != http.StatusOK {
		t.Fatalf("GET /api/projects/%d status = %d, want %d", created.ID, get.Code, http.StatusOK)
	}

	update := doJSON(t, router, http.MethodPatch, fmt.Sprintf("/api/projects/%d", created.ID), `{"name":"Renamed"}`)
	if update.Code != http.StatusOK {
		t.Fatalf("PATCH status = %d, want %d; body: %s", update.Code, http.StatusOK, update.Body.String())
	}
	var updated shared.ProjectData
	decodeData(t, update.Body.Bytes(), &updated)
	if updated.Name != "Renamed" || updated.Description != "desc" {
		t.Fatalf("updated project = %+v, want Name=Renamed Description=desc (unchanged)", updated)
	}
	// Not a strict .After(): POST and PATCH are two separate DB round trips
	// and almost certainly land on different timestamps in practice, but
	// "strictly later" isn't actually the invariant that matters here --
	// "not earlier" is, and asserting the stronger claim risks flaking if
	// they ever land within the same timestamp resolution window.
	if updated.UpdatedAt.Before(created.UpdatedAt) {
		t.Fatalf("updated.UpdatedAt = %v, want not before created.UpdatedAt = %v", updated.UpdatedAt, created.UpdatedAt)
	}

	del := doJSON(t, router, http.MethodDelete, fmt.Sprintf("/api/projects/%d", created.ID), "")
	if del.Code != http.StatusOK {
		t.Fatalf("DELETE status = %d, want %d", del.Code, http.StatusOK)
	}

	getAfterDelete := doJSON(t, router, http.MethodGet, fmt.Sprintf("/api/projects/%d", created.ID), "")
	if getAfterDelete.Code != http.StatusNotFound {
		t.Fatalf("GET after delete status = %d, want %d", getAfterDelete.Code, http.StatusNotFound)
	}

	invalid := doJSON(t, router, http.MethodPost, "/api/projects", `{"owner_id":1}`)
	if invalid.Code != http.StatusUnprocessableEntity {
		t.Fatalf("POST missing name status = %d, want %d; body: %s", invalid.Code, http.StatusUnprocessableEntity, invalid.Body.String())
	}
}

// TestCreateRejectsInvalidOwnerID checks that a foreign-key violation
// (owner_id referencing no existing user) is rejected as client error, not
// reported as an internal (500) failure. Issue #141 §15 requires every
// implementation reject equivalent invalid input the same way; a bad
// client-supplied reference is invalid input, not a server fault.
func TestCreateRejectsInvalidOwnerID(t *testing.T) {
	db := testDB(t)
	router := testRouter(t, db)

	response := doJSON(t, router, http.MethodPost, "/api/projects", `{"owner_id":999999,"name":"Orphan","description":"no such owner"}`)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("POST with nonexistent owner_id status = %d, want %d; body: %s", response.Code, http.StatusUnprocessableEntity, response.Body.String())
	}
	var envelope struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode error envelope: %v; body: %s", err, response.Body.String())
	}
	if envelope.Error.Code != "validation_error" {
		t.Fatalf("error code = %q, want %q", envelope.Error.Code, "validation_error")
	}
}

// blankNames covers both ways a client can submit a name with no real
// content: outright empty, and whitespace-only. Shared by
// TestCreateRejectsBlankName and TestUpdateRejectsBlankName because the
// whole point of both tests is that create and update reject the same
// inputs — a table one of the two forgot to update would silently
// reintroduce the asymmetry that motivated blankNameError in the first
// place (verified against live Postgres: POST used to accept "   "
// verbatim as a 201 while PATCH already rejected it as 422).
var blankNames = []string{"", "   "}

// TestCreateRejectsBlankName checks that POST /api/projects rejects both an
// empty name and a whitespace-only one. binding's `required` on a plain
// string only rejects the empty string, not "   " -- verified directly
// against live Postgres before blankNameError existed: POST with
// {"name":"   "} returned 201 Created with the name stored as three
// spaces, while the same value already failed on PATCH.
func TestCreateRejectsBlankName(t *testing.T) {
	db := testDB(t)
	if err := db.Create(&User{Email: "blank-name-create-owner@example.com", Name: "Owner"}).Error; err != nil {
		t.Fatalf("seed owner: %v", err)
	}
	router := testRouter(t, db)

	for _, name := range blankNames {
		t.Run(fmt.Sprintf("name=%q", name), func(t *testing.T) {
			body, err := json.Marshal(map[string]any{"owner_id": 1, "name": name, "description": "desc"})
			if err != nil {
				t.Fatalf("marshal request body: %v", err)
			}
			response := doJSON(t, router, http.MethodPost, "/api/projects", string(body))
			if response.Code != http.StatusUnprocessableEntity {
				t.Fatalf("POST with name=%q status = %d, want %d; body: %s", name, response.Code, http.StatusUnprocessableEntity, response.Body.String())
			}
		})
	}
}

// TestUpdateRejectsBlankName is TestCreateRejectsBlankName's mirror for
// PATCH /api/projects/:id.
func TestUpdateRejectsBlankName(t *testing.T) {
	db := testDB(t)
	if err := db.Create(&User{Email: "blank-name-update-owner@example.com", Name: "Owner"}).Error; err != nil {
		t.Fatalf("seed owner: %v", err)
	}
	router := testRouter(t, db)

	create := doJSON(t, router, http.MethodPost, "/api/projects", `{"owner_id":1,"name":"Original","description":"desc"}`)
	if create.Code != http.StatusCreated {
		t.Fatalf("POST /api/projects status = %d, want %d; body: %s", create.Code, http.StatusCreated, create.Body.String())
	}
	var created shared.ProjectData
	decodeData(t, create.Body.Bytes(), &created)

	for _, name := range blankNames {
		t.Run(fmt.Sprintf("name=%q", name), func(t *testing.T) {
			body, err := json.Marshal(map[string]any{"name": name})
			if err != nil {
				t.Fatalf("marshal request body: %v", err)
			}
			response := doJSON(t, router, http.MethodPatch, fmt.Sprintf("/api/projects/%d", created.ID), string(body))
			if response.Code != http.StatusUnprocessableEntity {
				t.Fatalf("PATCH with name=%q status = %d, want %d; body: %s", name, response.Code, http.StatusUnprocessableEntity, response.Body.String())
			}

			// The project must be unchanged -- a rejected update must not
			// have partially applied.
			get := doJSON(t, router, http.MethodGet, fmt.Sprintf("/api/projects/%d", created.ID), "")
			var unchanged shared.ProjectData
			decodeData(t, get.Body.Bytes(), &unchanged)
			if unchanged.Name != "Original" {
				t.Fatalf("project name after rejected PATCH (name=%q) = %q, want unchanged %q", name, unchanged.Name, "Original")
			}
		})
	}
}

// TestListPaginationAndOrdering seeds a known, small set of projects and
// checks the list endpoint's meta shape, deterministic id-DESC ordering,
// and that the owner relationship is actually populated (not just present
// as a zero value) — the fairness properties benchmarks/docs/schema.md
// requires.
func TestListPaginationAndOrdering(t *testing.T) {
	db := testDB(t)
	seedFixture(t, db, 3, 25) // 3 users, 25 projects — enough for two pages at limit=20
	router := testRouter(t, db)

	first := doJSON(t, router, http.MethodGet, "/api/projects?page=1&limit=20", "")
	if first.Code != http.StatusOK {
		t.Fatalf("GET /api/projects page 1 status = %d, want %d", first.Code, http.StatusOK)
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

	second := doJSON(t, router, http.MethodGet, "/api/projects?page=2&limit=20", "")
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

// TestListDoesNotN1 counts the SQL statements the list endpoint issues via
// a counting gorm.Logger and asserts a small constant bound, independent of
// page size — issue #141 §16 requires "the same fixed number of SQL
// queries" and specifically forbids an implementation whose query count
// scales with the number of returned rows. Verified manually against
// Postgres statement logs (docs/plans/BENCH-1-benchmark-suite.md, Phase 3
// notes) before writing this as an automated regression guard: for a
// non-empty page, one count query, one page query, one batched owner
// IN (...) query — 3 total, regardless of whether the page holds 1 row or
// 20. An empty page (no rows to preload owners for) issues only 2 — that's
// pinned by TestListDoesNotN1EmptyPage below, not this test: the invariant
// this repo actually needs is "count independent of N", not "always
// exactly 3 no matter what."
//
// Counts via db.Session(&gorm.Session{Logger: counter}) on the already-open
// connection from testDB, not a second gorm.Open: checked directly
// (gorm.Open, then Ping, against this driver/version) that neither issues
// any statement through the traced logger on its own, so a fresh gorm.Open
// wasn't actually adding spurious queries to the count here — but Session
// on an already-warm connection is the more robust pattern regardless
// (guaranteed no first-connection cost of any kind, present or future
// driver behavior), so it's what's used.
func TestListDoesNotN1(t *testing.T) {
	db := testDB(t)
	seedFixture(t, db, 5, 20)

	counter := &queryCounter{}
	router := testRouter(t, db.Session(&gorm.Session{Logger: counter}))

	response := doJSON(t, router, http.MethodGet, "/api/projects?page=1&limit=20", "")
	if response.Code != http.StatusOK {
		t.Fatalf("GET /api/projects status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if got := counter.Count(); got != 3 {
		t.Fatalf("list issued %d SQL statements, want exactly 3 (count + page + batched owner preload); got query log: %v", got, counter.Queries())
	}
}

// TestListDoesNotN1EmptyPage pins the boundary case TestListDoesNotN1's
// comment used to overclaim away: a page past the end of the data has no
// rows to preload owners for, so it issues 2 statements (count + page), not
// 3. Still O(1) in the number of returned rows (zero rows, zero extra
// queries) — the property that actually matters — just a different small
// constant than the non-empty case.
func TestListDoesNotN1EmptyPage(t *testing.T) {
	db := testDB(t)
	seedFixture(t, db, 5, 20)

	counter := &queryCounter{}
	router := testRouter(t, db.Session(&gorm.Session{Logger: counter}))

	response := doJSON(t, router, http.MethodGet, "/api/projects?page=99&limit=20", "")
	if response.Code != http.StatusOK {
		t.Fatalf("GET /api/projects status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if got := counter.Count(); got != 2 {
		t.Fatalf("empty-page list issued %d SQL statements, want exactly 2 (count + empty page, no owners to preload); got query log: %v", got, counter.Queries())
	}
}

// TestSeedDatabaseNIsIdempotentAndCorrect exercises the real
// truncate-then-seed path (seedDatabaseN, which seedDatabase calls at
// production scale) at a small scale against real PostgreSQL: exact row
// counts, deterministic content for known rows, round-robin ownership, and
// — running it twice — that repeated seeding truncates rather than
// accumulates duplicate data.
func TestSeedDatabaseNIsIdempotentAndCorrect(t *testing.T) {
	db := testDB(t)
	const userCount, projectCount = 7, 23 // deliberately not a multiple, to exercise the round-robin remainder

	for run := 1; run <= 2; run++ {
		if err := seedDatabaseN(context.Background(), db, userCount, projectCount); err != nil {
			t.Fatalf("run %d: seedDatabaseN: %v", run, err)
		}

		var userTotal, projectTotal int64
		if err := db.Model(&User{}).Count(&userTotal).Error; err != nil {
			t.Fatalf("run %d: count users: %v", run, err)
		}
		if err := db.Model(&Project{}).Count(&projectTotal).Error; err != nil {
			t.Fatalf("run %d: count projects: %v", run, err)
		}
		if userTotal != userCount {
			t.Fatalf("run %d: user count = %d, want %d (seed did not truncate before reseeding)", run, userTotal, userCount)
		}
		if projectTotal != projectCount {
			t.Fatalf("run %d: project count = %d, want %d (seed did not truncate before reseeding)", run, projectTotal, projectCount)
		}

		var firstUser User
		if err := db.First(&firstUser, 1).Error; err != nil {
			t.Fatalf("run %d: load user 1: %v", run, err)
		}
		if firstUser.Email != shared.UserEmail(1) || firstUser.Name != shared.UserName(1) {
			t.Fatalf("run %d: user 1 = %+v, want email=%s name=%s", run, firstUser, shared.UserEmail(1), shared.UserName(1))
		}

		// Project userCount+1 (8th project, userCount=7) is the first to
		// wrap back to owner 1 — the round-robin boundary a naive off-by-one
		// would break silently.
		var wrapped Project
		if err := db.First(&wrapped, userCount+1).Error; err != nil {
			t.Fatalf("run %d: load project %d: %v", run, userCount+1, err)
		}
		if wrapped.OwnerID != 1 {
			t.Fatalf("run %d: project %d owner = %d, want 1 (round-robin wrap)", run, userCount+1, wrapped.OwnerID)
		}
		if wrapped.Name != shared.ProjectName(userCount+1) {
			t.Fatalf("run %d: project %d name = %q, want %q", run, userCount+1, wrapped.Name, shared.ProjectName(userCount+1))
		}
	}
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

// seedFixture inserts userCount users and projectCount projects (round-robin
// ownership), independent of seed.go's full 1,000/100,000 dataset so tests
// run in milliseconds, not the minutes the full seed takes.
func seedFixture(t *testing.T, db *gorm.DB, userCount, projectCount int) {
	t.Helper()

	for i := 1; i <= userCount; i++ {
		if err := db.Create(&User{Email: fmt.Sprintf("fixture-%d@example.com", i), Name: fmt.Sprintf("Fixture User %d", i)}).Error; err != nil {
			t.Fatalf("seed fixture user %d: %v", i, err)
		}
	}
	for i := 1; i <= projectCount; i++ {
		if err := db.Create(&Project{OwnerID: shared.ProjectOwnerID(i, userCount), Name: fmt.Sprintf("Fixture Project %d", i), Description: "fixture"}).Error; err != nil {
			t.Fatalf("seed fixture project %d: %v", i, err)
		}
	}
}

// queryCounter is a minimal gorm.Logger that counts SQL statements traced
// via Trace, ignoring GORM's own log-level filtering entirely so the count
// is exact regardless of configured log level. Guarded by a mutex: even
// though today's callers only ever drive it from one goroutine per test,
// GORM does not guarantee Trace is called from the same goroutine that
// issued the query (e.g. connection-pool maintenance, a driver retry), and
// an unguarded field read/write racing with that would be a real data race
// under -race, not just a theoretical one.
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

// Count and Queries are the only reads of queryCounter's state (both from
// test assertions after the handler call under test has returned), but
// still go through the same mutex Trace uses rather than reading the fields
// directly, so nothing about correctness here depends on happens-before
// reasoning about when Trace's last call returns.
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
