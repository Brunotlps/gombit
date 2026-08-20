package main

import (
	"fmt"
	"os"

	"github.com/gombit-dev/gombit/internal/contractspike"
)

func main() {
	path := "openapi.json"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}

	if err := contractspike.WriteOpenAPI(path); err != nil {
		fmt.Fprintf(os.Stderr, "write OpenAPI document: %v\n", err)
		os.Exit(1)
	}
}
