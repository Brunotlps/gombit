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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gombit-dev/gombit/contract"
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

func TestSanitizeJSONBodyRejectsOversized(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	body := bytes.Repeat([]byte("x"), int(maxJSONBodyBytes)+1)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	sanitizeJSONBody(c)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
	if !c.IsAborted() {
		t.Fatal("expected XSS sanitizer to abort an oversized JSON body")
	}
	assertXSSPayloadTooLarge(t, rec)
}

type infiniteJSONReader struct{}

func (infiniteJSONReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'x'
	}
	return len(p), nil
}

func TestSanitizeJSONBodyDoesNotBlockOnInfiniteBody(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", infiniteJSONReader{})
	c.Request.Header.Set("Content-Type", "application/json")

	done := make(chan struct{})
	go func() {
		sanitizeJSONBody(c)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("sanitizeJSONBody blocked on unbounded ReadAll")
	}
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
	if !c.IsAborted() {
		t.Fatal("expected XSS sanitizer to abort an unbounded JSON body")
	}
	assertXSSPayloadTooLarge(t, rec)
}

type errJSONReader struct{}

func (errJSONReader) Read([]byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

func TestSanitizeJSONBodyReadErrorLeavesEmptyBody(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", errJSONReader{})
	c.Request.Header.Set("Content-Type", "application/json")

	sanitizeJSONBody(c)

	if c.IsAborted() {
		t.Fatal("read error should not abort; Huma/Gin emit D10 for the empty body")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want recorder default %d (no abort)", rec.Code, http.StatusOK)
	}
	got, err := io.ReadAll(c.Request.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("body = %q, want empty so handlers can reject it", got)
	}
}

func assertXSSPayloadTooLarge(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	var env contract.ErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode D10 envelope: %v; body: %s", err, rec.Body.String())
	}
	if env.Body.Code != contract.CodePayloadTooLarge {
		t.Fatalf("error.code = %q, want %q; body: %s", env.Body.Code, contract.CodePayloadTooLarge, rec.Body.String())
	}
	if env.Body.Message == "" {
		t.Fatalf("error.message empty; body: %s", rec.Body.String())
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
