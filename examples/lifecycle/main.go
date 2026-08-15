package main

import (
	"log"

	"github.com/LAA-Software-Engineering/gombit/framework"
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
