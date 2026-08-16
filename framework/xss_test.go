package framework

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestStripHTML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain", in: "hello", want: "hello"},
		{name: "script discarded", in: `<script>alert(1)</script>hi`, want: "hi"},
		{name: "bold stripped", in: `<b>hi</b>`, want: "hi"},
		{name: "img stripped", in: `x<img src=x onerror=alert(0)>y`, want: "xy"},
		{name: "unclosed script fail-closed", in: `<script src=x>alert(1)`, want: ""},
		{name: "nested markup", in: `<div><b>a</b><i>b</i></div>`, want: "ab"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := stripHTML(tt.in); got != tt.want {
				t.Fatalf("stripHTML(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSanitizeJSONValueNestedArray(t *testing.T) {
	t.Parallel()

	payload := map[string]any{
		"items": []any{
			map[string]any{"name": `<b>one</b>`},
			`<script>x</script>two`,
		},
	}
	sanitizeJSONValue(payload, "")

	items, ok := payload["items"].([]any)
	if !ok || len(items) != 2 {
		t.Fatalf("items = %#v", payload["items"])
	}
	first, ok := items[0].(map[string]any)
	if !ok || first["name"] != "one" {
		t.Fatalf("items[0] = %#v, want name=one", items[0])
	}
	if items[1] != "two" {
		t.Fatalf("items[1] = %#v, want two", items[1])
	}
}

func TestSanitizeJSONBodyInvalidJSONPassthrough(t *testing.T) {
	t.Parallel()

	raw := []byte(`{"comment":`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(raw))
	c.Request.Header.Set("Content-Type", "application/json")

	sanitizeJSONBody(c)

	got, err := io.ReadAll(c.Request.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !bytes.Equal(got, raw) {
		t.Fatalf("body = %q, want original invalid JSON %q", got, raw)
	}
}

func TestSanitizeJSONBodyEmptyBodyNoop(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader("   "))
	c.Request.Header.Set("Content-Type", "application/json")

	sanitizeJSONBody(c)

	got, err := io.ReadAll(c.Request.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(got) != "   " {
		t.Fatalf("body = %q, want unchanged whitespace", got)
	}
}

func TestSanitizeJSONBodyUpdatesContentLengthHeader(t *testing.T) {
	t.Parallel()

	raw := []byte(`{"comment":"<b>hi</b>"}`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(raw))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("Content-Length", "999")

	sanitizeJSONBody(c)

	got, err := io.ReadAll(c.Request.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var payload map[string]string
	if err := json.Unmarshal(got, &payload); err != nil {
		t.Fatalf("unmarshal sanitized body: %v (%s)", err, got)
	}
	if payload["comment"] != "hi" {
		t.Fatalf("comment = %q, want hi", payload["comment"])
	}
	if c.Request.ContentLength != int64(len(got)) {
		t.Fatalf("ContentLength = %d, want %d", c.Request.ContentLength, len(got))
	}
	if gotHeader := c.Request.Header.Get("Content-Length"); gotHeader != strconv.Itoa(len(got)) {
		t.Fatalf("Content-Length header = %q, want %d", gotHeader, len(got))
	}
}
