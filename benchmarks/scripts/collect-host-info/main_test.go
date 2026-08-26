package main

import "testing"

func TestParseIntListRejectsInvalidToken(t *testing.T) {
	// A malformed token is an error, not a silently dropped element — the
	// recorded sweep must equal the requested one or fail.
	for _, in := range []string{"1,10,abc,100", "1,,10", "100o", "1, 2, x"} {
		if got, err := parseIntList(in); err == nil {
			t.Errorf("parseIntList(%q) = %v, nil; want an error", in, got)
		}
	}
}

func TestParseIntListValid(t *testing.T) {
	got, err := parseIntList("1, 10 , 100,500,1000")
	if err != nil {
		t.Fatalf("parseIntList: %v", err)
	}
	want := []int{1, 10, 100, 500, 1000}
	if len(got) != len(want) {
		t.Fatalf("parseIntList len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("parseIntList[%d] = %d, want %d", i, got[i], want[i])
		}
	}

	if got, err := parseIntList(""); err != nil || got != nil {
		t.Errorf("parseIntList(\"\") = %v, %v; want nil, nil", got, err)
	}
}

func TestParseKeyValsEmptyIsNonNilMap(t *testing.T) {
	m, err := parseKeyVals("")
	if err != nil {
		t.Fatalf("parseKeyVals(\"\"): %v", err)
	}
	if m == nil {
		t.Fatal("parseKeyVals(\"\") = nil; want an empty (non-nil) map")
	}
	if len(m) != 0 {
		t.Errorf("parseKeyVals(\"\") = %v, want empty", m)
	}
}

func TestParseKeyValsValid(t *testing.T) {
	got, err := parseKeyVals("gombit=v0.1.0, gin-gorm=v1.11.0 ,go=1.25.7,empty=")
	if err != nil {
		t.Fatalf("parseKeyVals: %v", err)
	}
	if got["gombit"] != "v0.1.0" || got["gin-gorm"] != "v1.11.0" || got["go"] != "1.25.7" {
		t.Errorf("parseKeyVals = %v", got)
	}
	// An explicit empty value (django=) is a recorded fact, kept; a missing
	// '=' (django) is not — see TestParseKeyValsFailsClosed.
	if v, ok := got["empty"]; !ok || v != "" {
		t.Errorf("parseKeyVals should keep an explicit empty value: %v", got)
	}
}

// The sibling of parseIntList must fail closed too: a token that can't be a
// key=value pair is an error, never a silently dropped version. Otherwise the
// document claims a framework/runtime set that isn't what ran.
func TestParseKeyValsFailsClosed(t *testing.T) {
	for _, in := range []string{
		"gombit=v0.1.0,django",        // 'django' has no '='
		"go=1.25.7,python",            // 'python' has no '='
		"gombit=v0.1.0,",              // trailing comma
		"=v0.1.0",                     // empty key
		"gombit=v0.1.0,gombit=v0.2.0", // duplicate key
	} {
		if got, err := parseKeyVals(in); err == nil {
			t.Errorf("parseKeyVals(%q) = %v, nil; want an error", in, got)
		}
	}
}
