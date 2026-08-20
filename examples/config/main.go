package main

import (
	"fmt"
	"log"

	"github.com/gombit-dev/gombit/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s listens on %s with API prefix %s (docs enabled=%t)\n", cfg.AppName, cfg.HTTP.Addr, cfg.API.Prefix, cfg.API.DocsEnabled)
}
