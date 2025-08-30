package domain

import (
	"context"
	"time"
)

type Logger interface {
	// Context methods returns a logger based off the root
	// logger and decorates it with the given context and arguments.
	WithField(key string, value any) Logger
	WithFields(fields map[string]any) Logger
	WithError(err error) Logger

	// Standard log functions
	Print(args ...any)
	Debug(args ...any)
	Info(args ...any)
	Warn(args ...any)
	Error(args ...any)
	Fatal(args ...any)
	Panic(args ...any)

	// Formatted log functions
	Printf(format string, args ...any)
	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
	Panicf(format string, args ...any)
}

// Observability interface for observability logging features
type Observability interface {
	// Base interface
	Logger

	// Enhanced logging methods
	Success(msg string)
	Failure(msg string)
	Progress(msg string, current, total int)
	Benchmark(name string, duration time.Duration)
	API(method, path, ipAddress string, statusCode int, duration time.Duration)

	// With context
	WithContext(ctx context.Context) Observability
}
