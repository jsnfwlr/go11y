package go11y

import (
	"context"
)

func (o *Observer) Develop(msg string, ephemeralArgs ...any) {
	if o.span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		o.span.SetAttributes(attrs...)
		o.span.AddEvent(msg)
	}
	o.log(context.Background(), 3, LevelDevelop, msg, ephemeralArgs...)
}

func (o *Observer) Debug(msg string, ephemeralArgs ...any) {
	if o.span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		o.span.SetAttributes(attrs...)
		o.span.AddEvent(msg)
	}
	o.log(context.Background(), 3, LevelDebug, msg, ephemeralArgs...)
}

func (o *Observer) Info(msg string, ephemeralArgs ...any) {
	if o.span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		o.span.SetAttributes(attrs...)
		o.span.AddEvent(msg)
	}
	o.log(context.Background(), 3, LevelInfo, msg, ephemeralArgs...)
}

func (o *Observer) Notice(msg string, ephemeralArgs ...any) {
	if o.span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		o.span.SetAttributes(attrs...)
		o.span.AddEvent(msg)
	}
	o.log(context.Background(), 3, LevelNotice, msg, ephemeralArgs...)
}

func (o *Observer) Warning(msg string, ephemeralArgs ...any) {
	if o.span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		o.span.SetAttributes(attrs...)
		o.span.AddEvent(msg)
	}
	o.log(context.Background(), 3, LevelWarning, msg, ephemeralArgs...)
}

func (o *Observer) Warn(msg string, ephemeralArgs ...any) {
	if o.span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		o.span.SetAttributes(attrs...)
		o.span.AddEvent(msg)
	}
	o.log(context.Background(), 3, LevelWarning, msg, ephemeralArgs...)
}

func (o *Observer) Error(msg string, err error, severity string, ephemeralArgs ...any) {
	if o.span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		o.span.SetAttributes(attrs...)
		o.span.RecordError(err)
	}
	ephemeralArgs = append(ephemeralArgs, "error", err.Error(), "severity", severity)
	o.log(context.Background(), 3, LevelError, msg, ephemeralArgs...)
}

func (o *Observer) Fatal(msg string, err error, ephemeralArgs ...any) {
	if o.span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		o.span.SetAttributes(attrs...)
		o.span.RecordError(err)
	}
	ephemeralArgs = append(ephemeralArgs, "error", err.Error(), "severity", SeverityHighest)
	o.log(context.Background(), 3, LevelFatal, msg, ephemeralArgs...)
}
