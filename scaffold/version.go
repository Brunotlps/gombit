package scaffold

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// frameworkModulePath is the module a generated app requires.
const frameworkModulePath = "github.com/LAA-Software-Engineering/gombit"

// FallbackFrameworkVersion is written into a generated go.mod when the CLI
// cannot report a version that the module proxy could resolve — a `go run
// ./cmd/gombit` or a plain source build, which report DevVersion.
//
// It is deliberately unresolvable: a generated app pinned to it does not build
// until the user adds a replace directive pointing at a framework checkout.
// That is the correct outcome for a source build, since the whole point is to
// develop against local framework changes.
const FallbackFrameworkVersion = "v0.0.0"

// semverPattern matches a canonical module version. Go pseudo-versions
// (v0.0.0-20260818193315-9abb3c6ecc8c) are ordinary semver pre-releases and
// match here too, which matters: `go install ...@latest` against an untagged
// repository reports one, and the proxy can resolve it.
// Build metadata (+dirty, +incompatible) is deliberately excluded: it is not
// part of a canonical module version.
var semverPattern = regexp.MustCompile(
	`^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-[0-9A-Za-z.-]+)?$`)

// ResolveFrameworkVersion converts the running CLI's version into the version
// a generated go.mod should require.
//
// A version that the proxy can resolve is used as-is. Anything else — the
// "dev" sentinel, an empty string, or a v2+ version that the framework's
// module path could not satisfy — falls back to FallbackFrameworkVersion, and
// reason explains why for the caller to surface.
func ResolveFrameworkVersion(version string) (resolved string, reason string) {
	if version == "" {
		return FallbackFrameworkVersion, "the gombit binary reports no version"
	}
	// A build from a modified working tree reports a pseudo-version with
	// "+dirty" appended. It describes no published commit, so pinning it would
	// send the user to a version the proxy has never seen.
	if index := strings.IndexByte(version, '+'); index >= 0 {
		return FallbackFrameworkVersion, fmt.Sprintf(
			"%s was built from a modified source tree (%s)", version, version[index:])
	}
	match := semverPattern.FindStringSubmatch(version)
	if match == nil {
		return FallbackFrameworkVersion, fmt.Sprintf("%q is not a module version", version)
	}
	// A v2+ module must carry a /vN path suffix, which the generated import
	// paths do not have. Pinning it would produce a go.mod that can never
	// resolve, so fall back and say so rather than emit something broken.
	if major, err := strconv.Atoi(match[1]); err == nil && major >= 2 {
		return FallbackFrameworkVersion, fmt.Sprintf(
			"%s needs a /v%d module path suffix that generated imports do not use", version, major)
	}
	return version, ""
}
