package go11y

import (
	"context"

	"github.com/jsnfwlr/go11y/config"
)

func (o *Observer) Develop(msg string, ephemeralArgs ...any) {
	if o.span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		o.span.SetAttributes(attrs...)
		o.span.AddEvent(msg)
	}
	o.log(context.Background(), 3, config.LevelDevelop, msg, ephemeralArgs...)
}

func (o *Observer) Debug(msg string, ephemeralArgs ...any) {
	if o.span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		o.span.SetAttributes(attrs...)
		o.span.AddEvent(msg)
	}
	o.log(context.Background(), 3, config.LevelDebug, msg, ephemeralArgs...)
}

func (o *Observer) Info(msg string, ephemeralArgs ...any) {
	if o.span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		o.span.SetAttributes(attrs...)
		o.span.AddEvent(msg)
	}
	o.log(context.Background(), 3, config.LevelInfo, msg, ephemeralArgs...)
}

func (o *Observer) Notice(msg string, ephemeralArgs ...any) {
	if o.span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		o.span.SetAttributes(attrs...)
		o.span.AddEvent(msg)
	}
	o.log(context.Background(), 3, config.LevelNotice, msg, ephemeralArgs...)
}

func (o *Observer) Warning(msg string, ephemeralArgs ...any) {
	if o.span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		o.span.SetAttributes(attrs...)
		o.span.AddEvent(msg)
	}
	o.log(context.Background(), 3, config.LevelWarning, msg, ephemeralArgs...)
}

func (o *Observer) Warn(msg string, ephemeralArgs ...any) {
	if o.span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		o.span.SetAttributes(attrs...)
		o.span.AddEvent(msg)
	}
	o.log(context.Background(), 3, config.LevelWarning, msg, ephemeralArgs...)
}

func (o *Observer) Error(msg string, err error, severity config.Severity, ephemeralArgs ...any) {
	if o.span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		o.span.SetAttributes(attrs...)
		o.span.RecordError(err)
	}
	o.log(context.Background(), 3, config.LevelError, err.Error(), ephemeralArgs...)
}

func (o *Observer) Fatal(msg string, err error, ephemeralArgs ...any) {
	if o.span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		o.span.SetAttributes(attrs...)
		o.span.RecordError(err)
	}
	o.log(context.Background(), 3, config.LevelFatal, err.Error(), ephemeralArgs...)
}
