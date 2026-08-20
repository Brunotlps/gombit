//go:build integration

package admin_test

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"testing"

	"github.com/gombit-dev/gombit/auth"
	"github.com/gombit-dev/gombit/config"
	"github.com/gombit-dev/gombit/database"
	"github.com/gin-gonic/gin"
)

var (
	postgresDSN = flag.String("admin.postgres-dsn", "", "PostgreSQL DSN for admin integration tests")
	mysqlDSN    = flag.String("admin.mysql-dsn", "", "MySQL DSN for admin integration tests")
)

func TestResourcePostgres(t *testing.T) {
	if *postgresDSN == "" {
		t.Skip("set -admin.postgres-dsn to run Postgres admin integration tests")
	}
	runResourceDriver(t, config.DatabaseDriverPostgres, *postgresDSN)
}

func TestResourceMySQL(t *testing.T) {
	if *mysqlDSN == "" {
		t.Skip("set -admin.mysql-dsn to run MySQL admin integration tests")
	}
	runResourceDriver(t, config.DatabaseDriverMySQL, *mysqlDSN)
}

func runResourceDriver(t *testing.T, driver config.DatabaseDriver, dsn string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := openAdminDriver(t, driver, dsn)
	app := newCookieAppWithDB(t, db)
	registerWidgets(t, app)
	jar := loginSuperuser(t, app)

	create := doRequest(app, jar, http.MethodPost, "/api/v1/admin/resources/widgets", `{"name":"Bolt","sku":"b-1","price":5}`)
	if create.Code != http.StatusOK {
		t.Fatalf("create status = %d; body: %s", create.Code, create.Body.String())
	}
	var created rowEnvelope
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	id := fmt.Sprint(asInt(created.Data["id"]))

	list := doRequest(app, jar, http.MethodGet, "/api/v1/admin/resources/widgets?search=Bolt&ordering=name", "")
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d; body: %s", list.Code, list.Body.String())
	}
	del := doRequest(app, jar, http.MethodDelete, "/api/v1/admin/resources/widgets/"+id, "")
	if del.Code != http.StatusOK {
		t.Fatalf("delete status = %d; body: %s", del.Code, del.Body.String())
	}

	email := "integration-viewer@example.com"
	viewerJar := loginUser(t, app, email, testPassword)
	grantGroupPermission(t, app, email, "admin.widgets.view")
	viewOnly := doRequest(app, viewerJar, http.MethodGet, "/api/v1/admin/resources/widgets", "")
	if viewOnly.Code != http.StatusOK {
		t.Fatalf("view-only list status = %d; body: %s", viewOnly.Code, viewOnly.Body.String())
	}
	denied := doRequest(app, viewerJar, http.MethodPost, "/api/v1/admin/resources/widgets", `{"name":"Denied"}`)
	assertError(t, denied, http.StatusForbidden, "authorization")
}

func openAdminDriver(t *testing.T, driver config.DatabaseDriver, dsn string) *database.DB {
	t.Helper()
	db, err := database.Open(config.DatabaseConfig{Driver: driver, DSN: dsn})
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Migrator().DropTable(&Widget{})
		_ = db.Migrator().DropTable("auth_user_permissions", "auth_user_groups", "auth_group_permissions")
		_ = db.Migrator().DropTable(auth.Models()...)
		_ = db.Close()
	})
	return db
}
