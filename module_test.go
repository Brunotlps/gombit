package gombit

import "testing"

func TestModulePath(t *testing.T) {
	const want = "github.com/LAA-Software-Engineering/gombit"
	if ModulePath != want {
		t.Fatalf("ModulePath = %q, want %q", ModulePath, want)
	}
}
