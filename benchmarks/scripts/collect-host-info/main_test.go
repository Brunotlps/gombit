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
	m := parseKeyVals("")
	if m == nil {
		t.Fatal("parseKeyVals(\"\") = nil; want an empty (non-nil) map")
	}
	if len(m) != 0 {
		t.Errorf("parseKeyVals(\"\") = %v, want empty", m)
	}

	got := parseKeyVals("gombit=v0.1.0, gin-gorm=v1.11.0 ,malformed,go=1.25.7")
	if got["gombit"] != "v0.1.0" || got["gin-gorm"] != "v1.11.0" || got["go"] != "1.25.7" {
		t.Errorf("parseKeyVals = %v", got)
	}
	if _, ok := got["malformed"]; ok {
		t.Errorf("parseKeyVals kept a token with no '=': %v", got)
	}
}
