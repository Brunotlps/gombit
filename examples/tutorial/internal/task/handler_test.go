package task_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gombit-dev/gombit/config"
	"github.com/gombit-dev/gombit/contract"
	"github.com/gombit-dev/gombit/database"
	"github.com/gombit-dev/gombit/examples/tutorial/internal/task"
	"github.com/gombit-dev/gombit/framework"
	"go.uber.org/zap"
)

type listEnvelope struct {
	Data []map[string]any   `json:"data"`
	Meta *contract.PageMeta `json:"meta"`
}

func TestListHonorsPageAndPerPage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := newTaskApp(t)
	const total = 25
	for i := 0; i < total; i++ {
		createTask(t, app, fmt.Sprintf("task-%02d", i+1))
	}

	first := getList(t, app, "/api/v1/tasks")
	if first.Meta == nil {
		t.Fatal("default list missing meta")
	}
	if first.Meta.Page != 1 || first.Meta.PerPage != contract.DefaultPerPage || first.Meta.Total != total {
		t.Fatalf("default meta = %+v, want page=1 per_page=%d total=%d", first.Meta, contract.DefaultPerPage, total)
	}
	if len(first.Data) != contract.DefaultPerPage {
		t.Fatalf("default data len = %d, want %d (got more rows than advertised per_page)", len(first.Data), contract.DefaultPerPage)
	}

	page2 := getList(t, app, "/api/v1/tasks?page=2")
	if page2.Meta == nil || page2.Meta.Page != 2 || page2.Meta.PerPage != contract.DefaultPerPage || page2.Meta.Total != total {
		t.Fatalf("page=2 meta = %+v", page2.Meta)
	}
	if got, want := len(page2.Data), total-contract.DefaultPerPage; got != want {
		t.Fatalf("page=2 data len = %d, want %d", got, want)
	}
	seen := map[any]struct{}{}
	for _, row := range append(append([]map[string]any{}, first.Data...), page2.Data...) {
		id := row["id"]
		if _, ok := seen[id]; ok {
			t.Fatalf("pages overlap on id %#v", id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != total {
		t.Fatalf("page1+page2 unique ids = %d, want %d", len(seen), total)
	}

	small := getList(t, app, "/api/v1/tasks?page=1&per_page=5")
	if small.Meta == nil || small.Meta.Page != 1 || small.Meta.PerPage != 5 || small.Meta.Total != total {
		t.Fatalf("per_page=5 meta = %+v", small.Meta)
	}
	if len(small.Data) != 5 {
		t.Fatalf("per_page=5 data len = %d, want 5", len(small.Data))
	}

	clamped := getList(t, app, "/api/v1/tasks?page=0&per_page=1000")
	if clamped.Meta == nil || clamped.Meta.Page != contract.DefaultPage || clamped.Meta.PerPage != contract.MaxPerPage || clamped.Meta.Total != total {
		t.Fatalf("clamped meta = %+v, want page=%d per_page=%d total=%d", clamped.Meta, contract.DefaultPage, contract.MaxPerPage, total)
	}
	if len(clamped.Data) != total {
		t.Fatalf("clamped data len = %d, want %d (all rows fit under max per_page)", len(clamped.Data), total)
	}
}

func TestListOpenAPIExposesPageQueryParams(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := newTaskApp(t)

	rec := httptest.NewRecorder()
	app.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("openapi status = %d; body: %s", rec.Code, rec.Body.String())
	}
	spec := rec.Body.String()
	if !strings.Contains(spec, `"name":"page"`) || !strings.Contains(spec, `"name":"per_page"`) {
		t.Fatalf("OpenAPI missing page/per_page query params; body: %s", spec)
	}
}

func newTaskApp(t *testing.T) *framework.App {
	t.Helper()
	db, err := database.Open(config.DatabaseConfig{
		Driver: config.DatabaseDriverSQLite,
		DSN:    "file:" + filepath.Join(t.TempDir(), "task.db") + "?cache=shared&_fk=1",
	})
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.AutoMigrate(&task.Task{}); err != nil {
		t.Fatalf("AutoMigrate(Task) error = %v", err)
	}

	cfg := config.DefaultFor(config.EnvironmentTest)
	cfg.HTTP.Addr = "127.0.0.1:0"
	app, err := framework.New(
		framework.WithConfig(cfg),
		framework.WithDatabase(db),
		framework.WithLogger(zap.NewNop()),
	)
	if err != nil {
		t.Fatalf("framework.New() error = %v", err)
	}
	task.Register(app)
	return app
}

func createTask(t *testing.T, app *framework.App, title string) {
	t.Helper()
	body := fmt.Sprintf(`{"title":%q,"done":false}`, title)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	app.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create %q status = %d; body: %s", title, rec.Code, rec.Body.String())
	}
}

func getList(t *testing.T, app *framework.App, path string) listEnvelope {
	t.Helper()
	rec := httptest.NewRecorder()
	app.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d; body: %s", path, rec.Code, rec.Body.String())
	}
	var env listEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode GET %s: %v; body: %s", path, err, rec.Body.String())
	}
	return env
}
