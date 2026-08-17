package dev

import (
	"strings"
	"testing"
)

func TestFormatServiceTableIncludesDocsAndOpenAPI(t *testing.T) {
	t.Parallel()

	got := FormatServiceTable(Services{
		Backend:  "http://127.0.0.1:8080",
		Frontend: "http://127.0.0.1:5173",
	})
	wants := []string{
		"Backend",
		"http://127.0.0.1:8080",
		"Frontend",
		"http://127.0.0.1:5173",
		"/openapi.json",
		"/docs",
		"API docs",
		"OpenAPI",
	}
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Fatalf("service table missing %q:\n%s", want, got)
		}
	}
	if !strings.Contains(got, "http://127.0.0.1:8080/docs") {
		t.Fatalf("service table missing API docs URL:\n%s", got)
	}
	if !strings.Contains(got, "http://127.0.0.1:8080/openapi.json") {
		t.Fatalf("service table missing OpenAPI URL:\n%s", got)
	}
}

func TestOriginFromAddr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		addr string
		want string
	}{
		{addr: ":8080", want: "http://127.0.0.1:8080"},
		{addr: "127.0.0.1:8080", want: "http://127.0.0.1:8080"},
		{addr: "0.0.0.0:9000", want: "http://127.0.0.1:9000"},
		{addr: "8080", want: "http://127.0.0.1:8080"},
	}
	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			t.Parallel()
			if got := originFromAddr(tt.addr); got != tt.want {
				t.Fatalf("originFromAddr(%q) = %q, want %q", tt.addr, got, tt.want)
			}
		})
	}
}
