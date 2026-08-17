package main

import (
	"context"
	"log"

	"github.com/LAA-Software-Engineering/gombit/config"
	"github.com/LAA-Software-Engineering/gombit/framework"

	"github.com/example/demo/internal/platform"
	"github.com/example/demo/internal/product"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	db, err := platform.OpenDatabase(cfg.Database)
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

	app.OnStart(func(ctx context.Context) error {
		return platform.AutoMigrate(db)
	})
	app.OnStop(func(context.Context) error {
		return db.Close()
	})

	product.Register(app)

	if err := framework.Run(app); err != nil {
		log.Fatal(err)
	}
}
