package logger

import (
	"log/slog"
	"os"
	"strings"
	"sync"
)

var (
	defaultLogger *slog.Logger
	once          sync.Once
)

// Init initializes the default logger with a JSON handler writing to os.Stderr,
// keeping stdout clean for command output (digests are printed/piped from stdout).
// The level comes from BRIEFLY_LOG_LEVEL (debug|info|warn|error), defaulting to warn
// so routine runs aren't interleaved with log noise.
func Init() {
	once.Do(func() {
		defaultLogger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: levelFromEnv(),
		}))
		slog.SetDefault(defaultLogger)
	})
}

func levelFromEnv() slog.Level {
	switch strings.ToLower(os.Getenv("BRIEFLY_LOG_LEVEL")) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "error":
		return slog.LevelError
	case "warn", "":
		return slog.LevelWarn
	default:
		return slog.LevelWarn
	}
}

// Get returns the initialized default logger.
// It calls Init() to ensure the logger is ready before returning it.
func Get() *slog.Logger {
	Init() // Ensures logger is initialized
	return defaultLogger
}

// Info logs an informational message using the default logger.
func Info(msg string, args ...any) {
	Get().Info(msg, args...)
}

// Warn logs a warning message using the default logger.
func Warn(msg string, args ...any) {
	Get().Warn(msg, args...)
}

// Error logs an error message using the default logger.
func Error(msg string, err error, args ...any) {
	if err != nil {
		args = append(args, "error", err.Error())
	}
	Get().Error(msg, args...)
}

// Debug logs a debug message using the default logger.
func Debug(msg string, args ...any) {
	Get().Debug(msg, args...)
}
