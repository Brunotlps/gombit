package metadata

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// The required framework_versions / runtime_versions / concurrency fields must
// never serialize as JSON null, even at the CLI's default (no version flags,
// no concurrency): Collect initializes empty maps/slice, so the wire shape is
// {} / [], the honest "collected, nothing to record" value.
func TestWriteJSONNeverNullForRequiredFields(t *testing.T) {
	m := Collect(context.Background(), Options{
		Run: func(context.Context, string, ...string) (string, error) { return "", nil },
	})

	var buf bytes.Buffer
	if err := WriteJSON(&buf, m); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	s := buf.String()
	for _, wantNull := range []string{"framework_versions", "runtime_versions", "concurrency"} {
		if strings.Contains(s, `"`+wantNull+`": null`) {
			t.Errorf("%s serialized as null:\n%s", wantNull, s)
		}
	}
	for _, want := range []string{
		`"framework_versions": {}`,
		`"runtime_versions": {}`,
		`"concurrency": []`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("json missing %q:\n%s", want, s)
		}
	}

	// It must round-trip back to equal values.
	var decoded Metadata
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.FrameworkVersions == nil || decoded.Concurrency == nil {
		t.Errorf("round trip lost empty collections: %+v", decoded)
	}
}
