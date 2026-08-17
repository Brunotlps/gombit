package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/LAA-Software-Engineering/gombit/contract"
	"github.com/pb33f/libopenapi"
	openapivalidator "github.com/pb33f/libopenapi-validator"
)

const defaultOpenAPIURL = "http://127.0.0.1:8080/openapi.json"

var openAPIHTTPClient = &http.Client{Timeout: 30 * time.Second}

func runOpenAPI(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 {
		openapiUsage(stderr)
		return errors.New("gombit openapi: subcommand is required")
	}
	if args[0] != "generate" {
		openapiUsage(stderr)
		return fmt.Errorf("gombit openapi: unknown subcommand %q", args[0])
	}
	return runOpenAPIGenerate(ctx, args[1:], stdout, stderr)
}

func runOpenAPIGenerate(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) error {
	flags := flag.NewFlagSet("gombit openapi generate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	out := flags.String("out", "openapi.json", "output path for the OpenAPI 3.1 document")
	rawURL := flags.String("url", defaultOpenAPIURL, "URL of the live /openapi.json document")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		flags.Usage()
		return fmt.Errorf("gombit openapi generate: unexpected argument %q", flags.Arg(0))
	}

	spec, err := fetchOpenAPI(ctx, *rawURL)
	if err != nil {
		return err
	}
	if err := validateOpenAPIDocument(spec); err != nil {
		return fmt.Errorf("gombit openapi generate: %w", err)
	}
	if err := contract.WriteOpenAPIFile(*out, spec); err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "wrote OpenAPI document to %s\n", *out)
	return err
}

func fetchOpenAPI(ctx context.Context, rawURL string) ([]byte, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, fmt.Errorf("gombit openapi generate: parse url: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("gombit openapi generate: url scheme must be http or https")
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("gombit openapi generate: url host is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("gombit openapi generate: build request: %w", err)
	}
	resp, err := openAPIHTTPClient.Do(req) //nolint:gosec // URL is a user-supplied CLI flag for the local/live spec
	if err != nil {
		return nil, fmt.Errorf("gombit openapi generate: fetch %s: %w", parsed.String(), err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("gombit openapi generate: read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gombit openapi generate: GET %s: status %d", parsed.String(), resp.StatusCode)
	}
	return body, nil
}

func validateOpenAPIDocument(data []byte) error {
	document, err := libopenapi.NewDocument(data)
	if err != nil {
		return fmt.Errorf("parse OpenAPI document: %w", err)
	}
	validator, validatorErrs := openapivalidator.NewValidator(document)
	if len(validatorErrs) > 0 {
		return fmt.Errorf("create OpenAPI validator: %v", validatorErrs)
	}
	valid, documentErrs := validator.ValidateDocument()
	if !valid || len(documentErrs) > 0 {
		return fmt.Errorf("invalid OpenAPI 3.1 document: %v", documentErrs)
	}
	return nil
}

func openapiUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "available openapi subcommands:")
	_, _ = fmt.Fprintln(w, "  generate [--out openapi.json] [--url http://127.0.0.1:8080/openapi.json]")
}
