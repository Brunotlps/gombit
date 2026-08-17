package main

import (
	"fmt"
	"io"
	"time"

	"github.com/LAA-Software-Engineering/gombit/dev"
	"github.com/spf13/cobra"
)

func newDevCommand(stdout io.Writer, stderr io.Writer) *cobra.Command {
	var (
		httpAddr     string
		frontendHost string
		frontendPort int
		clientOut    string
		poll         time.Duration
	)
	cmd := silence(&cobra.Command{
		Use:   "dev",
		Short: "Run the API and Vite frontend together",
		Long: `Run the Go API and Vite frontend with HMR from an application directory.

Backend reload uses air when it is on PATH, then watchexec, then falls back
to go run ./cmd/server (with a hint). The frontend uses pnpm when available
(or when frontend/pnpm-lock.yaml exists), otherwise npm.

Vite proxies /api, /openapi.json, and /docs to the Go origin. When the live
OpenAPI document changes, gombit regenerates frontend/src/api/generated
(the same output as gombit client generate).

A service table is printed at startup, including the API docs URL (/docs).

SIGINT/SIGTERM stops the child processes. air/watchexec and Node.js are
optional for reload/HMR; a missing frontend/package.json is an error
(backend-only is not supported).

The Vite stub from gombit new is enough to start HMR. The full React
skeleton (router, React Hook Form, auth pages) is M5-1.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := dev.ValidateFlags(dev.DefaultHTTPAddr, frontendHost, frontendPort, poll, clientOut); err != nil {
				return fmt.Errorf("gombit %w", err)
			}
			if !cmd.Flags().Changed("http") {
				cfg, err := loadConfig()
				if err != nil {
					return fmt.Errorf("gombit dev: %w", err)
				}
				httpAddr = dev.HTTPAddrFromConfig(cfg)
			}
			if err := dev.ValidateFlags(httpAddr, frontendHost, frontendPort, poll, clientOut); err != nil {
				return fmt.Errorf("gombit %w", err)
			}
			err := runDev(cmd.Context(), dev.Options{
				WorkDir:      ".",
				HTTPAddr:     httpAddr,
				FrontendHost: frontendHost,
				FrontendPort: frontendPort,
				ClientOut:    clientOut,
				PollInterval: poll,
				Stdout:       stdout,
				Stderr:       stderr,
			})
			if err != nil {
				return fmt.Errorf("gombit %w", err)
			}
			return nil
		},
	})
	cmd.Flags().StringVar(&httpAddr, "http", dev.DefaultHTTPAddr, "Go API listen address (default GOMBIT_HTTP_ADDR or :8080)")
	cmd.Flags().StringVar(&frontendHost, "frontend-host", dev.DefaultFrontendHost, "Vite bind host")
	cmd.Flags().IntVar(&frontendPort, "frontend-port", dev.DefaultFrontendPort, "Vite port")
	cmd.Flags().StringVar(&clientOut, "client-out", dev.DefaultClientOut, "TypeScript client output directory")
	cmd.Flags().DurationVar(&poll, "poll", dev.DefaultPollInterval, "OpenAPI poll interval")
	return cmd
}

var runDev = dev.Run
