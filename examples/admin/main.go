package main

import (
	"context"
	"errors"
	"log"

	"github.com/LAA-Software-Engineering/gombit/auth"
	"github.com/LAA-Software-Engineering/gombit/config"
	"github.com/LAA-Software-Engineering/gombit/database"
	"github.com/LAA-Software-Engineering/gombit/examples/admin/internal/widget"
	"github.com/LAA-Software-Engineering/gombit/framework"
)

func main() {
	cfg := config.Default()
	cfg.HTTP.Addr = "127.0.0.1:8082"
	cfg.Auth.JWTSecret = "dev-only-example-jwt-secret-not-for-prod"
	cfg.Auth.Mode = config.AuthModeCookie
	// Secure=false only because this example serves plain HTTP on
	// 127.0.0.1. config.Validate requires CookieSecure=true in production.
	cfg.Auth.CookieSecure = false

	db, err := database.Open(config.DatabaseConfig{
		Driver: config.DatabaseDriverSQLite,
		DSN:    "file:admin-example?mode=memory&cache=shared&_fk=1",
	})
	if err != nil {
		log.Fatal(err)
	}

	app, err := framework.New(framework.WithConfig(cfg), framework.WithDatabase(db))
	if err != nil {
		_ = db.Close()
		log.Fatal(err)
	}
	if err := widget.RegisterAdmin(app); err != nil {
		_ = db.Close()
		log.Fatal(err)
	}

	app.OnStart(func(ctx context.Context) error {
		if err := auth.Migrate(db.DB); err != nil {
			return err
		}
		if err := db.AutoMigrate(&widget.Widget{}); err != nil {
			return err
		}
		svc, err := auth.NewService(db.DB, cfg)
		if err != nil {
			return err
		}
		_, err = svc.CreateSuperuser(ctx, "admin@example.com", "correct-horse-battery-staple")
		if err != nil && !errors.Is(err, auth.ErrEmailTaken) {
			return err
		}
		return nil
	})
	app.OnStop(func(context.Context) error {
		return db.Close()
	})

	log.Printf("admin example listening on http://%s (admin http://127.0.0.1:8082/admin/ docs http://127.0.0.1:8082/docs)", cfg.HTTP.Addr)
	log.Printf("seeded superuser admin@example.com / correct-horse-battery-staple")
	if err := framework.Run(app); err != nil {
		log.Fatal(err)
	}
}
