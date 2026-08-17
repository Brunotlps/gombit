package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/LAA-Software-Engineering/gombit/contract"
	"github.com/pb33f/libopenapi"
	openapivalidator "github.com/pb33f/libopenapi-validator"
	"github.com/spf13/cobra"
)

const (
	defaultOpenAPIURL     = "http://127.0.0.1:8080/openapi.json"
	defaultMaxOpenAPISize = 8 << 20
)

var (
	openAPIHTTPClient = &http.Client{Timeout: 30 * time.Second}
	// MaxOpenAPISize is the fetch cap for gombit openapi generate.
	MaxOpenAPISize int64 = defaultMaxOpenAPISize
)

func newOpenAPICommand(stdout io.Writer, stderr io.Writer) *cobra.Command {
	cmd := silence(&cobra.Command{
		Use:   "openapi",
		Short: "OpenAPI document commands",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			openapiUsage(stderr)
			if len(args) == 0 {
				return errors.New("gombit openapi: subcommand is required")
			}
			return fmt.Errorf("gombit openapi: unknown subcommand %q", args[0])
		},
	})
	cmd.AddCommand(newOpenAPIGenerateCommand(stdout, stderr))
	return cmd
}

func newOpenAPIGenerateCommand(stdout io.Writer, stderr io.Writer) *cobra.Command {
	cmd := silence(&cobra.Command{
		Use:   "generate",
		Short: "Fetch and write the live OpenAPI 3.1 document",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("gombit openapi generate: unexpected argument %q", args[0])
			}
			out, err := cmd.Flags().GetString("out")
			if err != nil {
				return err
			}
			rawURL, err := cmd.Flags().GetString("url")
			if err != nil {
				return err
			}
			spec, err := fetchOpenAPI(cmd.Context(), rawURL)
			if err != nil {
				return err
			}
			if err := validateOpenAPIDocument(spec); err != nil {
				return fmt.Errorf("gombit openapi generate: %w", err)
			}
			if err := contract.WriteOpenAPIFile(out, spec); err != nil {
				return err
			}
			_, err = fmt.Fprintf(stdout, "wrote OpenAPI document to %s\n", out)
			return err
		},
	})
	cmd.Flags().String("out", "openapi.json", "output path for the OpenAPI 3.1 document")
	cmd.Flags().String("url", defaultOpenAPIURL, "URL of the live /openapi.json document")
	return cmd
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
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxOpenAPISize+1))
	if err != nil {
		return nil, fmt.Errorf("gombit openapi generate: read response: %w", err)
	}
	if int64(len(body)) > MaxOpenAPISize {
		return nil, fmt.Errorf("gombit openapi generate: spec exceeds 8MiB")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gombit openapi generate: GET %s: status %d", parsed.String(), resp.StatusCode)
	}
	return body, nil
}

func validateOpenAPIDocument(data []byte) error {
	var probe map[string]any
	if err := json.Unmarshal(data, &probe); err != nil {
		return fmt.Errorf("parse OpenAPI document: %w", err)
	}
	version, _ := probe["openapi"].(string)
	if !strings.HasPrefix(version, "3.1.") && version != "3.1" {
		if version == "" {
			return errors.New("parse OpenAPI document: missing openapi 3.1 version")
		}
		return fmt.Errorf("parse OpenAPI document: openapi version %q is not 3.1", version)
	}

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
