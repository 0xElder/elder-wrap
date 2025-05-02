package logging

import (
	"context"
	"log/slog"
)

// slogLogger implements the Logger interface using slog.
type slogLogger struct {
	logger *slog.Logger
}

// NewLogger creates a new slogLogger with a specified handler.
func NewLogger(handler slog.Handler) Logger {
	return &slogLogger{
		logger: slog.New(handler),
	}
}

// Debug logs a message at Debug level.
func (l *slogLogger) Debug(ctx context.Context, msg string, args ...interface{}) {
	l.logger.DebugContext(ctx, msg, args...)
}

// Info logs a message at Info level.
func (l *slogLogger) Info(ctx context.Context, msg string, args ...interface{}) {
	l.logger.InfoContext(ctx, msg, args...)
}

// Warn logs a message at Warn level.
func (l *slogLogger) Warn(ctx context.Context, msg string, args ...interface{}) {
	l.logger.WarnContext(ctx, msg, args...)
}

// Error logs a message at Error level.
func (l *slogLogger) Error(ctx context.Context, msg string, args ...interface{}) {
	l.logger.ErrorContext(ctx, msg, args...)
}

// With returns a new Logger with additional context.
func (l *slogLogger) With(args ...interface{}) Logger {
	return &slogLogger{
		logger: l.logger.With(args...),
	}
}
