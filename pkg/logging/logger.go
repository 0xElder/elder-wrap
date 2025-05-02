package logging

import (
	"context"
	"log/slog"
	"os"

	"github.com/golang-cz/devslog"
)

type Logger interface {
	Debug(ctx context.Context, msg string, args ...interface{})
	Info(ctx context.Context, msg string, args ...interface{})
	Warn(ctx context.Context, msg string, args ...interface{})
	Error(ctx context.Context, msg string, args ...interface{})
	// ErrorWithStack(ctx context.Context, msg string, err error, args ...interface{})
	With(args ...interface{}) Logger
}

func DefaultOpts() *slog.HandlerOptions {
	return &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
}

// NewJSONLogger creates a new JSON logger that outputs to stdout.
func NewJSONLogger(opts *slog.HandlerOptions) Logger {
	if opts == nil {
		opts = DefaultOpts()
	}
	handler := slog.NewJSONHandler(os.Stdout, opts)
	return NewLogger(handler)
}

// NewTextLogger creates a new text logger that outputs to stdout.
func NewTextLogger(opts *slog.HandlerOptions) Logger {
	if opts == nil {
		opts = DefaultOpts()
	}
	handler := slog.NewTextHandler(os.Stdout, opts)

	return NewLogger(handler)
}

func NewDevSlogger(opts *slog.HandlerOptions) Logger {
	if opts == nil {
		opts = DefaultOpts()
	}
	return NewLogger(devslog.NewHandler(os.Stdout, &devslog.Options{
		HandlerOptions:  opts,
		NewLineAfterLog: true,
	}))
}
