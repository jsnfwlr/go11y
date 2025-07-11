package o11y

import (
	"context"
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

func (o *Observer) Develop(msg string, ephemeralArgs ...any) {
	if o.span != nil {
		o.event(o.span, msg, ephemeralArgs...)
	}
	o.log(context.Background(), 3, LevelDevelop, msg, ephemeralArgs...)
}

func (o *Observer) Debug(msg string, ephemeralArgs ...any) {
	if o.span != nil {
		o.event(o.span, msg, ephemeralArgs...)
	}
	o.log(context.Background(), 3, LevelDebug, msg, ephemeralArgs...)
}

func (o *Observer) Info(msg string, ephemeralArgs ...any) {
	if o.span != nil {
		o.event(o.span, msg, ephemeralArgs...)
	}
	o.log(context.Background(), 3, LevelInfo, msg, ephemeralArgs...)
}

func (o *Observer) Notice(msg string, ephemeralArgs ...any) {
	if o.span != nil {
		o.event(o.span, msg, ephemeralArgs...)
	}
	o.log(context.Background(), 3, LevelNotice, msg, ephemeralArgs...)
}

func (o *Observer) Warning(msg string, ephemeralArgs ...any) {
	if o.span != nil {
		o.event(o.span, msg, ephemeralArgs...)
	}
	o.log(context.Background(), 3, LevelWarning, msg, ephemeralArgs...)
}

func (o *Observer) Warn(msg string, ephemeralArgs ...any) {
	if o.span != nil {
		o.event(o.span, msg, ephemeralArgs...)
	}
	o.log(context.Background(), 3, LevelWarning, msg, ephemeralArgs...)
}

func (o *Observer) Error(err error, ephemeralArgs ...any) {
	if o.span != nil {
		o.error(o.span, err)
	}
	o.log(context.Background(), 3, LevelError, err.Error(), ephemeralArgs...)
}

func (o *Observer) Fatal(err error, ephemeralArgs ...any) {
	if o.span != nil {
		o.error(o.span, err)
	}
	o.log(context.Background(), 3, LevelFatal, err.Error(), ephemeralArgs...)
}
