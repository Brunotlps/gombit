// Command tutorial is the finished application from docs/tutorial.md: one
// Task resource served through Huma, with cookie auth and the runtime admin.
//
//	go run ./examples/tutorial
//	open http://127.0.0.1:8083/docs
//	open http://127.0.0.1:8083/admin/
//
// The tutorial builds this with `gombit new` + `gombit make resource`, which
// produces its own Go module. This copy lives inside the framework module so
// CI compiles it on every push — if the framework changes under the tutorial,
// the build breaks here first.
package main

import (
	"context"
	"errors"
	"log"

	"github.com/gombit-dev/gombit/auth"
	"github.com/gombit-dev/gombit/config"
	"github.com/gombit-dev/gombit/database"
	"github.com/gombit-dev/gombit/examples/tutorial/internal/task"
	"github.com/gombit-dev/gombit/framework"
)

const (
	adminEmail    = "admin@example.com"
	adminPassword = "correct-horse-battery-staple"
)

func main() {
	cfg := config.Default()
	cfg.HTTP.Addr = "127.0.0.1:8083"
	cfg.Auth.JWTSecret = "dev-only-example-jwt-secret-not-for-prod"
	// The admin requires cookie/session auth (ADR-013).
	cfg.Auth.Mode = config.AuthModeCookie
	// Secure=false only because this example serves plain HTTP on 127.0.0.1.
	// config.Validate requires CookieSecure=true in production.
	cfg.Auth.CookieSecure = false

	db, err := database.Open(config.DatabaseConfig{
		Driver: config.DatabaseDriverSQLite,
		DSN:    "file:tutorial-example?mode=memory&cache=shared&_fk=1",
	})
	if err != nil {
		log.Fatal(err)
	}

	app, err := framework.New(
		framework.WithConfig(cfg),
		framework.WithDatabase(db),
	)
	if err != nil {
		_ = db.Close()
		log.Fatal(err)
	}

	// Chapter 4: the API surface.
	task.Register(app)

	// Chapter 9: the same model on the runtime admin.
	if err := task.RegisterAdmin(app); err != nil {
		_ = db.Close()
		log.Fatal(err)
	}

	app.OnStart(func(ctx context.Context) error {
		// A generated app runs `gombit db migrate` instead; AutoMigrate keeps
		// this example self-contained on an in-memory database.
		if err := auth.Migrate(db.DB); err != nil {
			return err
		}
		if err := db.AutoMigrate(&task.Task{}); err != nil {
			return err
		}
		return seedSuperuser(ctx, db, cfg)
	})
	app.OnStop(func(context.Context) error {
		return db.Close()
	})

	log.Printf("tutorial example listening on http://%s", cfg.HTTP.Addr)
	log.Printf("  api    http://%s/api/v1/tasks", cfg.HTTP.Addr)
	log.Printf("  docs   http://%s/docs", cfg.HTTP.Addr)
	log.Printf("  admin  http://%s/admin/", cfg.HTTP.Addr)
	log.Printf("seeded superuser %s / %s", adminEmail, adminPassword)

	if err := framework.Run(app); err != nil {
		log.Fatal(err)
	}
}

// seedSuperuser is the in-process equivalent of `gombit createsuperuser`.
func seedSuperuser(ctx context.Context, db *database.DB, cfg config.Config) error {
	svc, err := auth.NewService(db.DB, cfg)
	if err != nil {
		return err
	}
	if _, err := svc.CreateSuperuser(ctx, adminEmail, adminPassword); err != nil &&
		!errors.Is(err, auth.ErrEmailTaken) {
		return err
	}
	return nil
}
