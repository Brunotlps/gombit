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
	if !updated.UpdatedAt.After(created.UpdatedAt) {
		t.Fatalf("updated.UpdatedAt = %v, want after created.UpdatedAt = %v", updated.UpdatedAt, created.UpdatedAt)
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

// TestListDoesNotN1s counts the SQL statements the list endpoint issues via
// a counting gorm.Logger and asserts a small constant bound, independent of
// page size — issue #141 §16 requires "the same fixed number of SQL
// queries" and specifically forbids an implementation whose query count
// scales with the number of returned rows. Verified manually against
// Postgres statement logs (docs/plans/BENCH-1-benchmark-suite.md, Phase 3
// notes) before writing this as an automated regression guard: one count
// query, one page query, one batched owner IN (...) query — 3 total,
// regardless of whether the page holds 1 row or 20.
func TestListDoesNotN1(t *testing.T) {
	db := testDB(t)
	seedFixture(t, db, 5, 20)

	counter := &queryCounter{}
	countingDB, err := gorm.Open(postgres.Open(*databaseDSN), &gorm.Config{Logger: counter})
	if err != nil {
		t.Fatalf("open counting db: %v", err)
	}
	router := testRouter(t, countingDB)

	response := doJSON(t, router, http.MethodGet, "/api/projects?page=1&limit=20", "")
	if response.Code != http.StatusOK {
		t.Fatalf("GET /api/projects status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if counter.count != 3 {
		t.Fatalf("list issued %d SQL statements, want exactly 3 (count + page + batched owner preload); got query log: %v", counter.count, counter.queries)
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
		ownerID := uint((i-1)%userCount + 1)
		if err := db.Create(&Project{OwnerID: ownerID, Name: fmt.Sprintf("Fixture Project %d", i), Description: "fixture"}).Error; err != nil {
			t.Fatalf("seed fixture project %d: %v", i, err)
		}
	}
}

// queryCounter is a minimal gorm.Logger that counts SQL statements traced
// via Trace, ignoring GORM's own log-level filtering entirely so the count
// is exact regardless of configured log level.
type queryCounter struct {
	count   int
	queries []string
}

func (q *queryCounter) LogMode(logger.LogLevel) logger.Interface      { return q }
func (q *queryCounter) Info(context.Context, string, ...interface{})  {}
func (q *queryCounter) Warn(context.Context, string, ...interface{})  {}
func (q *queryCounter) Error(context.Context, string, ...interface{}) {}
func (q *queryCounter) Trace(_ context.Context, _ time.Time, fc func() (string, int64), _ error) {
	sql, _ := fc()
	q.count++
	q.queries = append(q.queries, sql)
}
