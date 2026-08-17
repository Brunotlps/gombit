package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/LAA-Software-Engineering/gombit/cli"

	"github.com/example/demo/internal/product"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	root := cli.NewRoot(os.Stdout, os.Stderr)
	product.RegisterCommands(root)
	if err := cli.ExecuteRoot(ctx, root, os.Args[1:]); err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
