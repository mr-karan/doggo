package utils

import (
	"log/slog"
	"os"
)

// InitLogger initializes logger.
func InitLogger(debug bool) *slog.Logger {
	lvl := slog.LevelInfo
	if debug {
		lvl = slog.LevelDebug
	}

	lgrOpts := &slog.HandlerOptions{
		Level: lvl,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, lgrOpts))
	return logger
}
