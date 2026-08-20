package gombit

import "testing"

func TestModulePath(t *testing.T) {
	const want = "github.com/gombit-dev/gombit"
	if ModulePath != want {
		t.Fatalf("ModulePath = %q, want %q", ModulePath, want)
	}
}
