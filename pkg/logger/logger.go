// Package logger builds the application's structured slog.Logger.
package logger

import (
	"log/slog"
	"os"
)

// New builds a JSON slog.Logger. debug=true lowers the level and switches
// to a human-readable text handler for local development.
func New(debug bool) *slog.Logger {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if debug {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}
