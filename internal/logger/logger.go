package logger

import (
	"log/slog"
	"os"
	"sync"
)

var (
	defaultLogger *slog.Logger
	once          sync.Once
)

// Init initializes the default logger with a JSON handler writing to os.Stdout.
// It ensures that the logger is initialized only once.
func Init() {
	once.Do(func() {
		defaultLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug, // Default to Debug level, can be made configurable
		}))
		slog.SetDefault(defaultLogger) // Optionally set as the default logger for the slog package
		defaultLogger.Info("Logger initialized")
	})
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
