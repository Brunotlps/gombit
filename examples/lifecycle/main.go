package main

import (
	"log"

	"github.com/gombit-dev/gombit/framework"
)

func main() {
	app, err := framework.New()
	if err != nil {
		log.Fatal(err)
	}

	if err := framework.Run(app); err != nil {
		log.Fatal(err)
	}
}
