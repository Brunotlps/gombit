package main

import (
	"log"

	"github.com/LAA-Software-Engineering/gombit/config"
	"github.com/LAA-Software-Engineering/gombit/framework"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Default()
	cfg.AppName = "logging-example"

	app, err := framework.New(framework.WithConfig(cfg))
	if err != nil {
		log.Fatal(err)
	}

	app.Logger().Info("logger ready", zap.String("sink", string(cfg.Logging.Sink)))
}
