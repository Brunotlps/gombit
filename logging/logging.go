package logging

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/LAA-Software-Engineering/gombit/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Option customizes logger construction.
type Option func(*options) error

type options struct {
	core zapcore.Core
}

// WithCore supplies a Zap core from an external logging module.
func WithCore(core zapcore.Core) Option {
	return func(opts *options) error {
		if core == nil {
			return errors.New("logging: nil core")
		}
		opts.core = core
		return nil
	}
}

// New builds a Zap logger for cfg.
func New(cfg config.LoggingConfig, opts ...Option) (*zap.Logger, error) {
	if err := config.ValidateLogging(cfg); err != nil {
		return nil, err
	}

	options := options{}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(&options); err != nil {
			return nil, err
		}
	}

	level, err := zapLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	core := options.core
	if core == nil {
		core, err = coreFor(cfg, level)
		if err != nil {
			return nil, err
		}
	}

	return zap.New(
		core,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
		zap.ErrorOutput(zapcore.Lock(os.Stderr)),
	), nil
}

func coreFor(cfg config.LoggingConfig, level zapcore.Level) (zapcore.Core, error) {
	var sink zapcore.WriteSyncer
	switch cfg.Sink {
	case config.LogSinkStderr:
		sink = zapcore.Lock(os.Stderr)
	case config.LogSinkStdout:
		sink = zapcore.Lock(os.Stdout)
	case config.LogSinkMongo:
		return nil, errors.New("logging: mongo sink requires an external zapcore.Core")
	default:
		return nil, fmt.Errorf("logging: unsupported sink %q", cfg.Sink)
	}

	return zapcore.NewCore(encoder(), sink, level), nil
}

func encoder() zapcore.Encoder {
	cfg := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout(time.RFC3339Nano),
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	return zapcore.NewJSONEncoder(cfg)
}

func zapLevel(level config.LogLevel) (zapcore.Level, error) {
	switch level {
	case config.LogLevelDebug:
		return zapcore.DebugLevel, nil
	case config.LogLevelInfo:
		return zapcore.InfoLevel, nil
	case config.LogLevelWarn:
		return zapcore.WarnLevel, nil
	case config.LogLevelError:
		return zapcore.ErrorLevel, nil
	default:
		return zapcore.InfoLevel, fmt.Errorf("logging: unsupported level %q", level)
	}
}
