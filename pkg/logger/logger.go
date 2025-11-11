package logger

import (
	"log/slog"
	"os"
)

// New creates and returns a new structured logger (slog).
func New(level slog.Level) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
}
