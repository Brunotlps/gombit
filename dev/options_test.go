package dev

import (
	"strings"
	"testing"
	"time"
)

func TestValidateFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		httpAddr     string
		frontendHost string
		frontendPort int
		poll         time.Duration
		clientOut    string
		want         string
	}{
		{
			name:         "empty http",
			httpAddr:     "   ",
			frontendHost: DefaultFrontendHost,
			frontendPort: DefaultFrontendPort,
			poll:         DefaultPollInterval,
			clientOut:    DefaultClientOut,
			want:         "--http",
		},
		{
			name:         "invalid http",
			httpAddr:     "not-an-addr",
			frontendHost: DefaultFrontendHost,
			frontendPort: DefaultFrontendPort,
			poll:         DefaultPollInterval,
			clientOut:    DefaultClientOut,
			want:         "--http",
		},
		{
			name:         "empty frontend host",
			httpAddr:     DefaultHTTPAddr,
			frontendHost: " ",
			frontendPort: DefaultFrontendPort,
			poll:         DefaultPollInterval,
			clientOut:    DefaultClientOut,
			want:         "--frontend-host",
		},
		{
			name:         "zero frontend port",
			httpAddr:     DefaultHTTPAddr,
			frontendHost: DefaultFrontendHost,
			frontendPort: 0,
			poll:         DefaultPollInterval,
			clientOut:    DefaultClientOut,
			want:         "--frontend-port",
		},
		{
			name:         "negative frontend port",
			httpAddr:     DefaultHTTPAddr,
			frontendHost: DefaultFrontendHost,
			frontendPort: -1,
			poll:         DefaultPollInterval,
			clientOut:    DefaultClientOut,
			want:         "--frontend-port",
		},
		{
			name:         "frontend port too large",
			httpAddr:     DefaultHTTPAddr,
			frontendHost: DefaultFrontendHost,
			frontendPort: 70000,
			poll:         DefaultPollInterval,
			clientOut:    DefaultClientOut,
			want:         "--frontend-port",
		},
		{
			name:         "zero poll",
			httpAddr:     DefaultHTTPAddr,
			frontendHost: DefaultFrontendHost,
			frontendPort: DefaultFrontendPort,
			poll:         0,
			clientOut:    DefaultClientOut,
			want:         "--poll",
		},
		{
			name:         "empty client out",
			httpAddr:     DefaultHTTPAddr,
			frontendHost: DefaultFrontendHost,
			frontendPort: DefaultFrontendPort,
			poll:         DefaultPollInterval,
			clientOut:    " ",
			want:         "--client-out",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateFlags(tt.httpAddr, tt.frontendHost, tt.frontendPort, tt.poll, tt.clientOut)
			if err == nil {
				t.Fatal("ValidateFlags() error = nil, want validation error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ValidateFlags() error = %q, want %q", err, tt.want)
			}
		})
	}
}

func TestValidateFlagsOK(t *testing.T) {
	t.Parallel()
	err := ValidateFlags(DefaultHTTPAddr, DefaultFrontendHost, DefaultFrontendPort, DefaultPollInterval, DefaultClientOut)
	if err != nil {
		t.Fatalf("ValidateFlags() error = %v", err)
	}
}
