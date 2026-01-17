package logger

import (
	"log/slog"
	"os"
	"strings"

	"github.com/Marlliton/slogpretty"
)

func init() {
	Setup("info")
}

// Setup initializes the global slog logger with the given level.
// level can be "debug", "info", "warn", or "error".
func Setup(level string) {
	var slogLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		slogLevel = slog.LevelDebug
	case "info":
		slogLevel = slog.LevelInfo
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	logger := slog.New(slogpretty.New(os.Stdout, &slogpretty.Options{
		Level:      slogLevel,
		TimeFormat: "2006-01-02 15:04:05",
	}))
	slog.SetDefault(logger)
}
