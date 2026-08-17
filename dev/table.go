package dev

import (
	"fmt"
	"strings"
)

// Services holds the URLs printed in the startup service table.
type Services struct {
	Backend  string
	Frontend string
}

// FormatServiceTable renders the Backend / Frontend / OpenAPI / API docs table.
func FormatServiceTable(services Services) string {
	rows := [][2]string{
		{"Backend", services.Backend},
		{"Frontend", services.Frontend},
		{"OpenAPI", services.Backend + "/openapi.json"},
		{"API docs", services.Backend + "/docs"},
	}
	width := 0
	for _, row := range rows {
		if n := len(row[0]); n > width {
			width = n
		}
	}
	var b strings.Builder
	for _, row := range rows {
		fmt.Fprintf(&b, "%-*s  %s\n", width, row[0], row[1])
	}
	return b.String()
}
