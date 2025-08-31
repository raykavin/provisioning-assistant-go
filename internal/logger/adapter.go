package logger

import (
	"context"
	"fmt"
	"provisioning-assistant/internal/domain"
	"time"
)

type ZLogXAdapter struct {
	*ZLogX
}

// Ensure it implements both interfaces
var (
	_ domain.Logger        = (*ZLogXAdapter)(nil)
	_ domain.Observability = (*ZLogXAdapter)(nil)
)

// Print implements Logging.
func (s *ZLogXAdapter) Print(args ...any) {
	s.Logger.Print(args...)
}

// Debug implements Logging.
func (s *ZLogXAdapter) Debug(args ...any) {
	s.Logger.Debug().Msg(fmt.Sprint(args...))
}

// Info implements Logging.
func (s *ZLogXAdapter) Info(args ...any) {
	s.Logger.Info().Msg(fmt.Sprint(args...))
}

// Warn implements Logging.
func (s *ZLogXAdapter) Warn(args ...any) {
	s.Logger.Warn().Msg(fmt.Sprint(args...))
}

// Error implements Logging.
func (s *ZLogXAdapter) Error(args ...any) {
	s.Logger.Error().Msg(fmt.Sprint(args...))
}

// Fatal implements Logging.
func (s *ZLogXAdapter) Fatal(args ...any) {
	s.Logger.Fatal().Msg(fmt.Sprint(args...))
}

// Panic implements Logging.
func (s *ZLogXAdapter) Panic(args ...any) {
	s.Logger.Panic().Msg(fmt.Sprint(args...))
}

// Printf implements Logging.
func (s *ZLogXAdapter) Printf(format string, args ...any) {
	s.Logger.Printf(format, args...)
}

// Debugf implements Logging.
func (s *ZLogXAdapter) Debugf(format string, args ...any) {
	s.Logger.Debug().Msgf(format, args...)
}

// Infof implements Logging.
func (s *ZLogXAdapter) Infof(format string, args ...any) {
	s.Logger.Info().Msgf(format, args...)
}

// Warnf implements Logging.
func (s *ZLogXAdapter) Warnf(format string, args ...any) {
	s.Logger.Warn().Msgf(format, args...)
}

// Errorf implements Logging.
func (s *ZLogXAdapter) Errorf(format string, args ...any) {
	s.Logger.Error().Msgf(format, args...)
}

// Fatalf implements Logging.
func (s *ZLogXAdapter) Fatalf(format string, args ...any) {
	s.Logger.Fatal().Msgf(format, args...)
}

// Panicf implements Logging.
func (s *ZLogXAdapter) Panicf(format string, args ...any) {
	s.Logger.Panic().Msgf(format, args...)
}

// WithError implements Logging.
func (s *ZLogXAdapter) WithError(err error) domain.Logger {
	newLogger := s.With().Err(err).Logger()
	return &ZLogXAdapter{&ZLogX{Logger: &newLogger}}
}

// WithField implements Logging.
func (s *ZLogXAdapter) WithField(key string, value any) domain.Logger {
	newLogger := s.With().Interface(key, value).Logger()
	return &ZLogXAdapter{&ZLogX{Logger: &newLogger}}
}

// WithFields implements Logging.
func (s *ZLogXAdapter) WithFields(fields map[string]any) domain.Logger {
	newLogger := s.With().Fields(fields).Logger()
	return &ZLogXAdapter{&ZLogX{Logger: &newLogger}}
}

// Success implements SmartLogging.
func (s *ZLogXAdapter) Success(msg string) {
	s.ZLogX.Success(msg)
}

// Failure implements SmartLogging.
func (s *ZLogXAdapter) Failure(msg string) {
	s.ZLogX.Failure(msg)
}

// Progress implements SmartLogging.
func (s *ZLogXAdapter) Progress(msg string, current, total int) {
	s.ZLogX.Progress(msg, current, total)
}

// Benchmark implements SmartLogging.
func (s *ZLogXAdapter) Benchmark(name string, duration time.Duration) {
	s.ZLogX.Benchmark(name, duration)
}

// API implements SmartLogging.
func (s *ZLogXAdapter) API(method, path, remoteAddr string, statusCode int, duration time.Duration) {
	s.ZLogX.API(method, path, remoteAddr, statusCode, duration)
}

// WithContext implements SmartLogging.
func (s *ZLogXAdapter) WithContext(ctx context.Context) domain.Observability {
	s.Logger.WithContext(ctx)
	return &ZLogXAdapter{ZLogX: s.ZLogX}
}
