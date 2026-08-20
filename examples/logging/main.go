package main

import (
	"log"

	"github.com/gombit-dev/gombit/config"
	"github.com/gombit-dev/gombit/framework"
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
