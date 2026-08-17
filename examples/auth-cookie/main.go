package main

import (
	"context"
	"log"

	"github.com/LAA-Software-Engineering/gombit/auth"
	"github.com/LAA-Software-Engineering/gombit/config"
	"github.com/LAA-Software-Engineering/gombit/database"
	"github.com/LAA-Software-Engineering/gombit/framework"
)

func main() {
	cfg := config.Default()
	cfg.HTTP.Addr = "127.0.0.1:8081"
	cfg.Auth.JWTSecret = "dev-only-example-jwt-secret-not-for-prod"
	cfg.Auth.Mode = config.AuthModeCookie
	// Secure=false only because this example serves plain HTTP on
	// 127.0.0.1. config.Validate requires CookieSecure=true in production
	// (Appendix C); see docs/auth-cookie.md.
	cfg.Auth.CookieSecure = false

	db, err := database.Open(config.DatabaseConfig{
		Driver: config.DatabaseDriverSQLite,
		DSN:    "file:auth-cookie-example?mode=memory&cache=shared&_fk=1",
	})
	if err != nil {
		log.Fatal(err)
	}

	app, err := framework.New(framework.WithConfig(cfg), framework.WithDatabase(db))
	if err != nil {
		_ = db.Close()
		log.Fatal(err)
	}
	app.OnStart(func(context.Context) error {
		return auth.Migrate(db.DB)
	})
	app.OnStop(func(context.Context) error {
		return db.Close()
	})

	log.Printf("auth-cookie example listening on http://%s (docs http://127.0.0.1:8081/docs)", cfg.HTTP.Addr)
	if err := framework.Run(app); err != nil {
		log.Fatal(err)
	}
}
