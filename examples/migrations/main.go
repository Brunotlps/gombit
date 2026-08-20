package main

import (
	"fmt"

	"github.com/gombit-dev/gombit/examples/migrations/internal/product"
)

func main() {
	fmt.Printf("example migration model: %T\n", product.Product{})
}
