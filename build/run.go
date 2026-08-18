package build

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	errMissingFrontend = "build: missing frontend/package.json; gombit build --embed requires the Vite frontend (backend-only is not supported). Re-run gombit new or add a frontend/package.json"
	errMissingServer   = "build: missing cmd/server; run gombit build from a Gombit application directory"
	errMissingEmbed    = "build: missing internal/web/embed.go; re-run gombit new (the embed hook is scaffolded there)"
)

// ErrEmbedRequired is returned when gombit build is invoked without --embed.
// Split deploy stays the default (C5).
var ErrEmbedRequired = errors.New("build: split deploy is the default (C5). Embedding the Vite frontend is opt-in; pass --embed to collectstatic frontend/dist into internal/web/static and compile a single binary that serves API + static + SPA fallback")

// Run performs the embed production build from an application directory.
func Run(ctx context.Context, opts Options) error {
	if ctx == nil {
		return errors.New("build: nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := opts.normalize(); err != nil {
		return err
	}
	if err := opts.validate(); err != nil {
		return err
	}
	if !opts.Embed {
		return ErrEmbedRequired
	}

	frontendPkg := filepath.Join(opts.WorkDir, filepath.FromSlash(frontendPkgRel))
	if _, err := os.Stat(frontendPkg); err != nil {
		if os.IsNotExist(err) {
			return errors.New(errMissingFrontend)
		}
		return fmt.Errorf("build: stat frontend/package.json: %w", err)
	}
	serverDir := filepath.Join(opts.WorkDir, "cmd", "server")
	if info, err := os.Stat(serverDir); err != nil || !info.IsDir() {
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("build: stat cmd/server: %w", err)
		}
		return errors.New(errMissingServer)
	}
	embedGo := filepath.Join(opts.WorkDir, filepath.FromSlash(embedGoRel))
	if _, err := os.Stat(embedGo); err != nil {
		if os.IsNotExist(err) {
			return errors.New(errMissingEmbed)
		}
		return fmt.Errorf("build: stat embed.go: %w", err)
	}

	frontend, err := planFrontendBuild(opts.WorkDir, opts.LookPath)
	if err != nil {
		return err
	}
	goBin, err := opts.LookPath("go")
	if err != nil {
		return fmt.Errorf("build: go not found in PATH")
	}

	distDir := filepath.Join(opts.WorkDir, filepath.FromSlash(frontendDistRel))
	staticDir := filepath.Join(opts.WorkDir, filepath.FromSlash(staticRel))
	outPath := opts.Out
	if !filepath.IsAbs(outPath) {
		outPath = filepath.Join(opts.WorkDir, outPath)
	}

	if opts.DryRun {
		return printPlan(opts, frontend, goBin, distDir, staticDir, outPath)
	}

	frontendDir := filepath.Join(opts.WorkDir, frontendDirRel)
	if !frontendDepsInstalled(frontendDir) {
		if _, err := fmt.Fprintf(opts.Stdout, "installing frontend dependencies with %s\n", frontend.Manager); err != nil {
			return err
		}
		if err := runCommand(ctx, opts, frontendDir, frontend.Path, []string{"install"}); err != nil {
			return fmt.Errorf("build: %s install: %w", frontend.Manager, err)
		}
	}

	if _, err := fmt.Fprintf(opts.Stdout, "create %s\n", displayRel(opts.WorkDir, distDir)); err != nil {
		return err
	}
	if err := runCommand(ctx, opts, frontendDir, frontend.Path, frontend.Args); err != nil {
		return fmt.Errorf("build: %s run build: %w", frontend.Manager, err)
	}

	if _, err := fmt.Fprintf(opts.Stdout, "copy %s -> %s\n", displayRel(opts.WorkDir, distDir), displayRel(opts.WorkDir, staticDir)); err != nil {
		return err
	}
	if err := CollectStatic(distDir, staticDir); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o750); err != nil {
		return fmt.Errorf("build: mkdir %s: %w", filepath.Dir(outPath), err)
	}
	relOut := displayRel(opts.WorkDir, outPath)
	if _, err := fmt.Fprintf(opts.Stdout, "compile %s\n", relOut); err != nil {
		return err
	}
	if err := runCommand(ctx, opts, opts.WorkDir, goBin, []string{"build", "-o", opts.Out, serverPkg}); err != nil {
		return fmt.Errorf("build: go build: %w", err)
	}
	return nil
}

func printPlan(opts Options, frontend frontendPlan, goBin, distDir, staticDir, outPath string) error {
	var lines []string
	frontendDir := filepath.Join(opts.WorkDir, frontendDirRel)
	if !frontendDepsInstalled(frontendDir) {
		lines = append(lines, fmt.Sprintf("would run: %s install (in %s)", frontend.Manager, frontendDirRel))
	}
	lines = append(lines,
		fmt.Sprintf("would run: %s %s (in %s)", frontend.Manager, joinArgs(frontend.Args), frontendDirRel),
		fmt.Sprintf("would copy: %s -> %s", displayRel(opts.WorkDir, distDir), displayRel(opts.WorkDir, staticDir)),
		fmt.Sprintf("would compile: %s build -o %s %s", filepath.Base(goBin), displayRel(opts.WorkDir, outPath), serverPkg),
	)
	for _, line := range lines {
		if _, err := fmt.Fprintln(opts.Stdout, line); err != nil {
			return err
		}
	}
	return nil
}

func runCommand(ctx context.Context, opts Options, dir, path string, args []string) error {
	cmd := opts.Command(path, args...) //nolint:gosec // path/args come from LookPath and fixed flags
	if cmd == nil {
		return fmt.Errorf("build: nil command for %s", path)
	}
	cmd.Dir = dir
	cmd.Stdout = opts.Stdout
	cmd.Stderr = opts.Stderr
	if err := runCmdContext(ctx, cmd); err != nil {
		return err
	}
	return nil
}

func runCmdContext(ctx context.Context, cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case <-ctx.Done():
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		<-done
		return ctx.Err()
	case err := <-done:
		return err
	}
}

func displayRel(workDir, full string) string {
	rel, err := filepath.Rel(workDir, full)
	if err != nil {
		return filepath.ToSlash(full)
	}
	return filepath.ToSlash(rel)
}

func joinArgs(args []string) string {
	return strings.Join(args, " ")
}
