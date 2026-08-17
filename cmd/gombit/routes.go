package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/LAA-Software-Engineering/gombit/config"
	"github.com/LAA-Software-Engineering/gombit/framework"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func newRoutesCommand(stdout io.Writer, stderr io.Writer) *cobra.Command {
	cmd := silence(&cobra.Command{
		Use:   "routes",
		Short: "Print HTTP routes",
		Long: `Print a table of HTTP method and path.

By default this constructs an in-process framework.App (memory cache,
docs enabled) and lists framework-owned routes: probes, metrics,
OpenAPI, and /docs. Application feature routes are registered by the
app itself; list the live OpenAPI contract with --url against a
running server (for example gombit routes --url http://127.0.0.1:8080).`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rawURL, err := cmd.Flags().GetString("url")
			if err != nil {
				return err
			}
			rows, note, err := collectRoutes(cmd.Context(), rawURL)
			if err != nil {
				return fmt.Errorf("gombit routes: %w", err)
			}
			if note != "" {
				_, _ = fmt.Fprintln(stderr, note)
			}
			return writeRouteTable(stdout, rows)
		},
	})
	cmd.Flags().String("url", "", "list OpenAPI paths from a running server (http or https origin or /openapi.json URL)")
	return cmd
}

type routeRow struct {
	Method string
	Path   string
}

func collectRoutes(ctx context.Context, rawURL string) ([]routeRow, string, error) {
	if strings.TrimSpace(rawURL) != "" {
		rows, err := routesFromURL(ctx, rawURL)
		if err != nil {
			return nil, "", err
		}
		return rows, "OpenAPI contract paths from --url. Probe/metrics routes are raw Gin and are not in the spec.", nil
	}
	rows, err := frameworkOwnedRoutes()
	if err != nil {
		return nil, "", err
	}
	return rows, "Framework-owned routes from an in-process framework.App. Application feature routes require the running app: gombit routes --url http://127.0.0.1:8080", nil
}

func frameworkOwnedRoutes() ([]routeRow, error) {
	gin.SetMode(gin.ReleaseMode)
	cfg := config.DefaultFor(config.EnvironmentTest)
	cfg.HTTP.Addr = "127.0.0.1:0"
	cfg.API.DocsEnabled = true
	cfg.Cache.Driver = config.CacheDriverMemory
	app, err := framework.New(framework.WithConfig(cfg), framework.WithLogger(zap.NewNop()))
	if err != nil {
		return nil, err
	}
	ginRoutes := app.Router().Routes()
	rows := make([]routeRow, 0, len(ginRoutes))
	for _, route := range ginRoutes {
		rows = append(rows, routeRow{Method: route.Method, Path: route.Path})
	}
	sortRouteRows(rows)
	return rows, nil
}

func routesFromURL(ctx context.Context, rawURL string) ([]routeRow, error) {
	specURL, err := openAPIURLFromRoutesFlag(rawURL)
	if err != nil {
		return nil, err
	}
	spec, err := fetchOpenAPI(ctx, specURL)
	if err != nil {
		return nil, err
	}
	rows, err := routesFromOpenAPI(spec)
	if err != nil {
		return nil, err
	}
	sortRouteRows(rows)
	return rows, nil
}

func openAPIURLFromRoutesFlag(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse url: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("url scheme must be http or https")
	}
	if strings.HasSuffix(parsed.Path, "/openapi.json") || parsed.Path == "openapi.json" {
		return parsed.String(), nil
	}
	parsed.Path = strings.TrimSuffix(parsed.Path, "/") + "/openapi.json"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func routesFromOpenAPI(spec []byte) ([]routeRow, error) {
	var doc struct {
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(spec, &doc); err != nil {
		return nil, fmt.Errorf("parse OpenAPI: %w", err)
	}
	methods := []string{"get", "post", "put", "patch", "delete", "head", "options", "trace"}
	rows := make([]routeRow, 0)
	for path, item := range doc.Paths {
		for _, method := range methods {
			if _, ok := item[method]; ok {
				rows = append(rows, routeRow{
					Method: strings.ToUpper(method),
					Path:   path,
				})
			}
		}
	}
	return rows, nil
}

func sortRouteRows(rows []routeRow) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Path == rows[j].Path {
			return rows[i].Method < rows[j].Method
		}
		return rows[i].Path < rows[j].Path
	})
}

func writeRouteTable(w io.Writer, rows []routeRow) error {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "METHOD\tPATH"); err != nil {
		return err
	}
	for _, row := range rows {
		if _, err := fmt.Fprintf(tw, "%s\t%s\n", row.Method, row.Path); err != nil {
			return err
		}
	}
	return tw.Flush()
}
