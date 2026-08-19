package config

import (
	"bufio"
	"os"
	"strings"
)

// dotEnvFile is the file Load looks for in the current working directory.
// gombit new writes this with a per-project random GOMBIT_JWT_SECRET; it is
// gitignored and never read by LoadFromEnv directly, only by Load.
const dotEnvFile = ".env"

// loadDotEnv reads KEY=VALUE pairs from .env in the current working
// directory and applies them to the process environment, without
// overwriting a variable that is already set. It is a silent no-op when the
// file does not exist, so it is safe to call unconditionally: a real
// deployment sets variables through its own environment and never ships a
// .env file (it is gitignored by every gombit new scaffold).
func loadDotEnv() {
	f, err := os.Open(dotEnvFile)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		key, value, ok := parseDotEnvLine(scanner.Text())
		if !ok {
			continue
		}
		if _, set := os.LookupEnv(key); set {
			continue
		}
		_ = os.Setenv(key, value)
	}
}

// parseDotEnvLine parses a single KEY=VALUE line, skipping blanks and `#`
// comments. It strips one layer of matching single or double quotes from
// the value, the same convention every gombit-generated .env file follows.
func parseDotEnvLine(line string) (key, value string, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false
	}
	k, v, found := strings.Cut(line, "=")
	if !found {
		return "", "", false
	}
	k = strings.TrimSpace(k)
	if k == "" {
		return "", "", false
	}
	v = strings.TrimSpace(v)
	if len(v) >= 2 {
		if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
			v = v[1 : len(v)-1]
		}
	}
	return k, v, true
}
