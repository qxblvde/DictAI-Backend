package logger

import (
	"log/slog"
	"os"
	"strings"
)

func New(service string) *slog.Logger {
	level := parseLevel(os.Getenv("LOG_LEVEL"))
	handler := newHandler(level)
	return slog.New(handler).With("service", service)
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func newHandler(level slog.Level) slog.Handler {
	opts := &slog.HandlerOptions{Level: level}
	if strings.ToLower(strings.TrimSpace(os.Getenv("LOG_FORMAT"))) == "text" {
		return slog.NewTextHandler(os.Stdout, opts)
	}
	return slog.NewJSONHandler(os.Stdout, opts)
}
