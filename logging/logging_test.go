package logging

import (
	"strings"
	"testing"

	"github.com/LAA-Software-Engineering/gombit/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestNewBuildsDefaultLoggerWithoutMongo(t *testing.T) {
	logger, err := New(config.Default().Logging)
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}
	if logger == nil {
		t.Fatal("New() logger = nil, want logger")
	}
}

func TestNewUsesCustomCoreForExternalSink(t *testing.T) {
	core, observed := observer.New(zapcore.DebugLevel)

	logger, err := New(config.LoggingConfig{
		Level: config.LogLevelDebug,
		Sink:  config.LogSinkMongo,
	}, WithCore(core))
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}

	logger.Debug("captured", zap.String("sink", "mongo"))
	if got := observed.Len(); got != 1 {
		t.Fatalf("observed log count = %d, want 1", got)
	}
	if got := observed.All()[0].ContextMap()["sink"]; got != "mongo" {
		t.Fatalf("observed sink field = %#v, want %q", got, "mongo")
	}
}

func TestNewRequiresExternalCoreForMongoSink(t *testing.T) {
	_, err := New(config.LoggingConfig{
		Level: config.LogLevelInfo,
		Sink:  config.LogSinkMongo,
	})
	if err == nil {
		t.Fatal("New() error = nil, want mongo external core error")
	}
	if !strings.Contains(err.Error(), "mongo sink requires an external zapcore.Core") {
		t.Fatalf("New() error = %q, want mongo external core message", err)
	}
}

func TestWithCoreRejectsNilCore(t *testing.T) {
	_, err := New(config.Default().Logging, WithCore(nil))
	if err == nil {
		t.Fatal("New(..., WithCore(nil)) error = nil, want error")
	}
}
