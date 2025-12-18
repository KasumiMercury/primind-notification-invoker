package config

import (
	"log/slog"
	"os"
	"strings"
)

type Config struct {
	Port              string
	FirebaseProjectID string
	LogLevel          slog.Level
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return &Config{
		Port:              port,
		FirebaseProjectID: os.Getenv("FIREBASE_PROJECT_ID"),
		LogLevel:          parseLogLevel(os.Getenv("LOG_LEVEL")),
	}
}

func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
