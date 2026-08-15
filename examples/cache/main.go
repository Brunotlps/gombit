package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/LAA-Software-Engineering/gombit/framework"
)

type widget struct {
	ID   int
	Name string
}

func main() {
	app, err := framework.New()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	if err := app.Cache().Set(ctx, "widgets:1", widget{ID: 1, Name: "cached"}, time.Minute); err != nil {
		log.Fatal(err)
	}

	var got widget
	found, err := app.Cache().Get(ctx, "widgets:1", &got)
	if err != nil {
		log.Fatal(err)
	}

	count, err := app.Cache().Increment(ctx, "rate:example", 1)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("found=%t widget=%s count=%d redis=%t\n", found, got.Name, count, app.Redis() != nil)
}
