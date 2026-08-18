package build

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

const indexHTML = "index.html"

// CollectStatic copies srcDir (frontend/dist) into dstDir (internal/web/static),
// replacing previous assets but leaving .keep and never touching embed.go.
func CollectStatic(srcDir, dstDir string) error {
	if _, err := os.Stat(filepath.Join(srcDir, indexHTML)); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("build: missing %s after the frontend build", filepath.ToSlash(filepath.Join(srcDir, indexHTML)))
		}
		return fmt.Errorf("build: stat index.html: %w", err)
	}
	if err := os.MkdirAll(dstDir, 0o750); err != nil {
		return fmt.Errorf("build: mkdir %s: %w", dstDir, err)
	}
	if err := clearStaticAssets(dstDir); err != nil {
		return err
	}
	return copyTree(srcDir, dstDir)
}

func clearStaticAssets(dstDir string) error {
	entries, err := os.ReadDir(dstDir)
	if err != nil {
		return fmt.Errorf("build: read %s: %w", dstDir, err)
	}
	for _, entry := range entries {
		if entry.Name() == keepName {
			continue
		}
		if err := os.RemoveAll(filepath.Join(dstDir, entry.Name())); err != nil {
			return fmt.Errorf("build: replace %s: %w", entry.Name(), err)
		}
	}
	return nil
}

func copyTree(srcDir, dstDir string) error {
	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return fmt.Errorf("build: relative path: %w", err)
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dstDir, rel)
		if d.IsDir() {
			if err := os.MkdirAll(target, 0o750); err != nil {
				return fmt.Errorf("build: mkdir %s: %w", target, err)
			}
			return nil
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return fmt.Errorf("build: mkdir %s: %w", filepath.Dir(dst), err)
	}
	// #nosec G304 -- src and dst are rooted application paths (frontend/dist → internal/web/static)
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("build: open %s: %w", src, err)
	}
	defer func() { _ = in.Close() }()

	// #nosec G304 -- dst is under internal/web/static in the application work dir
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("build: write %s: %w", dst, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return fmt.Errorf("build: copy %s: %w", dst, err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("build: close %s: %w", dst, err)
	}
	return nil
}
