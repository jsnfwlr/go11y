package config

import (
	"log/slog"
	"strings"
)

const (
	LevelDevelop = slog.Level(-8) // Custom level for development only logging, will be disabled in staging and production
	LevelDebug   = slog.Level(-4)
	LevelInfo    = slog.Level(0)
	LevelNotice  = slog.Level(2)
	LevelWarning = slog.Level(4)
	LevelError   = slog.Level(8)
	LevelFatal   = slog.Level(12)
)

func StringToLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "develop":
		return LevelDevelop // Custom level for development, not used in production
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "notice":
		return LevelNotice
	case "warning", "warn":
		return LevelWarning
	case "error":
		return LevelError
	case "fatal":
		return LevelFatal
	default:
		return LevelDebug // default to debug if unknown level
	}
}
