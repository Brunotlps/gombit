package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/LAA-Software-Engineering/gombit/contract"
	"github.com/danielgtaylor/huma/v2"
)

//go:generate go run ../cmd/gombit client check --write --spec ../examples/client/openapi.json --out ../examples/client/frontend/src/api/generated

const (
	// SampleSpecPath is the committed OpenAPI fixture for the sample widget API.
	SampleSpecPath = "examples/client/openapi.json"
	// SampleOutDir is the committed TypeScript client generated from SampleSpecPath.
	SampleOutDir = "examples/client/frontend/src/api/generated"
)

var sampleClientFiles = []string{"schema.ts", "client.ts", "error.ts"}

// DriftOptions configures a contract drift check or fixture rewrite.
type DriftOptions struct {
	// WorkDir is the repository root. Empty means the current working directory.
	WorkDir string
	// SpecPath is the committed OpenAPI fixture, relative to WorkDir unless absolute.
	// Default: SampleSpecPath.
	SpecPath string
	// OutDir is the committed generated-client directory, relative to WorkDir unless absolute.
	// Default: SampleOutDir.
	OutDir string
	// API is the Huma API to emit. Nil means SampleApp().
	API huma.API
	// Write regenerates committed fixtures instead of comparing them.
	Write bool
	// NPXBinary is the npx executable used by Generate. Default: npx.
	NPXBinary string
	Stdout    io.Writer
	Stderr    io.Writer
}

// CheckDrift regenerates the sample OpenAPI document and TypeScript client and
// reports whether committed artifacts would change.
//
// The spec is compared semantically via encoding/json, so whitespace-only
// differences are not drift. Generated TypeScript is compared byte-for-byte.
// Generation is in-process from SampleApp (or opts.API) and does not fetch a
// live /openapi.json URL.
//
// When Write is true, fixtures are rewritten in place with contract.WriteOpenAPI
// and Generate.
func CheckDrift(ctx context.Context, opts DriftOptions) error {
	if ctx == nil {
		return errors.New("client: nil context")
	}
	opts = normalizeDriftOptions(opts)

	api := opts.API
	if api == nil {
		app, err := SampleApp()
		if err != nil {
			return fmt.Errorf("client: sample app: %w", err)
		}
		api = app.API()
	}

	if opts.Write {
		return writeSampleFixtures(ctx, opts, api)
	}
	return compareSampleFixtures(ctx, opts, api)
}

func normalizeDriftOptions(opts DriftOptions) DriftOptions {
	if opts.WorkDir == "" {
		opts.WorkDir = "."
	}
	if opts.SpecPath == "" {
		opts.SpecPath = SampleSpecPath
	}
	if opts.OutDir == "" {
		opts.OutDir = SampleOutDir
	}
	if opts.Stdout == nil {
		opts.Stdout = io.Discard
	}
	if opts.Stderr == nil {
		opts.Stderr = io.Discard
	}
	if opts.NPXBinary == "" {
		opts.NPXBinary = "npx"
	}
	return opts
}

func resolvePath(workDir, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(workDir, path)
}

func writeSampleFixtures(ctx context.Context, opts DriftOptions, api huma.API) error {
	specPath := resolvePath(opts.WorkDir, opts.SpecPath)
	if err := contract.WriteOpenAPI(specPath, api); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(opts.Stdout, "wrote %s\n", displayPath(opts.WorkDir, specPath)); err != nil {
		return err
	}
	return Generate(ctx, Options{
		WorkDir:   opts.WorkDir,
		SpecPath:  specPath,
		OutDir:    resolvePath(opts.WorkDir, opts.OutDir),
		NPXBinary: opts.NPXBinary,
		Stdout:    opts.Stdout,
		Stderr:    opts.Stderr,
	})
}

func compareSampleFixtures(ctx context.Context, opts DriftOptions, api huma.API) error {
	tmpDir, err := os.MkdirTemp("", "gombit-drift-*")
	if err != nil {
		return fmt.Errorf("client: temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	generatedSpecPath := filepath.Join(tmpDir, "openapi.json")
	if err := contract.WriteOpenAPI(generatedSpecPath, api); err != nil {
		return err
	}
	// #nosec G304 -- generatedSpecPath is created in this function
	generatedSpec, err := os.ReadFile(generatedSpecPath)
	if err != nil {
		return fmt.Errorf("client: read generated spec: %w", err)
	}

	specPath := resolvePath(opts.WorkDir, opts.SpecPath)
	// #nosec G304 -- spec path is a user-supplied committed fixture
	committedSpec, err := os.ReadFile(specPath)
	if err != nil {
		return fmt.Errorf("client: read committed spec: %w", err)
	}

	var drifted []string
	equal, err := jsonEqual(committedSpec, generatedSpec)
	if err != nil {
		return err
	}
	if !equal {
		drifted = append(drifted, displayPath(opts.WorkDir, specPath))
	}

	generatedOut := filepath.Join(tmpDir, "generated")
	if err := Generate(ctx, Options{
		WorkDir:   tmpDir,
		SpecPath:  generatedSpecPath,
		OutDir:    generatedOut,
		NPXBinary: opts.NPXBinary,
		Stdout:    io.Discard,
		Stderr:    opts.Stderr,
	}); err != nil {
		return err
	}

	committedOut := resolvePath(opts.WorkDir, opts.OutDir)
	for _, name := range sampleClientFiles {
		committedPath := filepath.Join(committedOut, name)
		generatedPath := filepath.Join(generatedOut, name)
		// #nosec G304 -- committedPath is the checked-in generated client file
		committed, readErr := os.ReadFile(committedPath)
		if readErr != nil {
			return fmt.Errorf("client: read committed %s: %w", displayPath(opts.WorkDir, committedPath), readErr)
		}
		// #nosec G304 -- generatedPath is created in this function
		generated, readErr := os.ReadFile(generatedPath)
		if readErr != nil {
			return fmt.Errorf("client: read generated %s: %w", name, readErr)
		}
		if !bytes.Equal(committed, generated) {
			drifted = append(drifted, displayPath(opts.WorkDir, committedPath))
		}
	}

	if len(drifted) > 0 {
		return fmt.Errorf("client: contract drift in %s; regenerate with gombit client check --write", strings.Join(drifted, ", "))
	}
	_, err = fmt.Fprintln(opts.Stdout, "no contract drift")
	return err
}

func jsonEqual(left, right []byte) (bool, error) {
	var leftValue any
	var rightValue any
	if err := json.Unmarshal(left, &leftValue); err != nil {
		return false, fmt.Errorf("client: parse committed spec: %w", err)
	}
	if err := json.Unmarshal(right, &rightValue); err != nil {
		return false, fmt.Errorf("client: parse generated spec: %w", err)
	}
	leftBytes, err := json.Marshal(leftValue)
	if err != nil {
		return false, fmt.Errorf("client: marshal committed spec: %w", err)
	}
	rightBytes, err := json.Marshal(rightValue)
	if err != nil {
		return false, fmt.Errorf("client: marshal generated spec: %w", err)
	}
	return string(leftBytes) == string(rightBytes), nil
}
