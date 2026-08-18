package scaffold

import (
	"strings"
	"testing"
)

func TestResolveFrameworkVersion(t *testing.T) {
	tests := map[string]struct {
		version      string
		wantResolved string
		wantFallback bool
		reasonHas    string
	}{
		"release tag": {
			version:      "v0.1.0",
			wantResolved: "v0.1.0",
		},
		"pre-release tag": {
			version:      "v0.2.0-rc.1",
			wantResolved: "v0.2.0-rc.1",
		},
		// `go install ...@latest` against an untagged repo reports one of
		// these, and the proxy resolves it — so it must not be rejected.
		"pseudo-version": {
			version:      "v0.0.0-20260818193315-9abb3c6ecc8c",
			wantResolved: "v0.0.0-20260818193315-9abb3c6ecc8c",
		},
		"v1 release": {
			version:      "v1.4.2",
			wantResolved: "v1.4.2",
		},
		// `go build` from a modified checkout reports this; it describes no
		// published commit and is not a canonical module version.
		"dirty pseudo-version": {
			version:      "v0.0.0-20260818195347-ca19c17754ca+dirty",
			wantResolved: FallbackFrameworkVersion,
			wantFallback: true,
			reasonHas:    "modified source tree",
		},
		"incompatible suffix": {
			version:      "v1.4.2+incompatible",
			wantResolved: FallbackFrameworkVersion,
			wantFallback: true,
			reasonHas:    "modified source tree",
		},
		"dev sentinel": {
			version:      "dev",
			wantResolved: FallbackFrameworkVersion,
			wantFallback: true,
			reasonHas:    `"dev" is not a module version`,
		},
		"empty": {
			version:      "",
			wantResolved: FallbackFrameworkVersion,
			wantFallback: true,
			reasonHas:    "reports no version",
		},
		"go devel marker": {
			version:      "(devel)",
			wantResolved: FallbackFrameworkVersion,
			wantFallback: true,
			reasonHas:    "not a module version",
		},
		"missing v prefix": {
			version:      "0.1.0",
			wantResolved: FallbackFrameworkVersion,
			wantFallback: true,
			reasonHas:    "not a module version",
		},
		"not semver": {
			version:      "v1.2",
			wantResolved: FallbackFrameworkVersion,
			wantFallback: true,
			reasonHas:    "not a module version",
		},
		// v2+ needs a /vN module path suffix that generated imports lack, so
		// pinning it would emit a go.mod that can never resolve.
		"v2 needs a path suffix": {
			version:      "v2.0.0",
			wantResolved: FallbackFrameworkVersion,
			wantFallback: true,
			reasonHas:    "/v2 module path suffix",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			resolved, reason := ResolveFrameworkVersion(tc.version)
			if resolved != tc.wantResolved {
				t.Errorf("ResolveFrameworkVersion(%q) resolved = %q, want %q",
					tc.version, resolved, tc.wantResolved)
			}
			if gotFallback := reason != ""; gotFallback != tc.wantFallback {
				t.Errorf("ResolveFrameworkVersion(%q) reason = %q, want fallback = %v",
					tc.version, reason, tc.wantFallback)
			}
			if tc.reasonHas != "" && !strings.Contains(reason, tc.reasonHas) {
				t.Errorf("ResolveFrameworkVersion(%q) reason = %q, want it to contain %q",
					tc.version, reason, tc.reasonHas)
			}
		})
	}
}
