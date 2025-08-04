package o11y

import (
	"context"

	"github.com/jsnfwlr/o11y/config"
	otelTrace "go.opentelemetry.io/otel/trace"
)

func (o *Observer) Develop(msg string, span otelTrace.Span, ephemeralArgs ...any) {
	if span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		span.SetAttributes(attrs...)
		span.AddEvent(msg)
	}
	o.log(context.Background(), 3, config.LevelDevelop, msg, ephemeralArgs...)
}

func (o *Observer) Debug(msg string, span otelTrace.Span, ephemeralArgs ...any) {
	if span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		span.SetAttributes(attrs...)
		span.AddEvent(msg)
	}
	o.log(context.Background(), 3, config.LevelDebug, msg, ephemeralArgs...)
}

func (o *Observer) Info(msg string, span otelTrace.Span, ephemeralArgs ...any) {
	if span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		span.SetAttributes(attrs...)
		span.AddEvent(msg)
	}
	o.log(context.Background(), 3, config.LevelInfo, msg, ephemeralArgs...)
}

func (o *Observer) Notice(msg string, span otelTrace.Span, ephemeralArgs ...any) {
	if span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		span.SetAttributes(attrs...)
		span.AddEvent(msg)
	}
	o.log(context.Background(), 3, config.LevelNotice, msg, ephemeralArgs...)
}

func (o *Observer) Warning(msg string, span otelTrace.Span, ephemeralArgs ...any) {
	if span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		span.SetAttributes(attrs...)
		span.AddEvent(msg)
	}
	o.log(context.Background(), 3, config.LevelWarning, msg, ephemeralArgs...)
}

func (o *Observer) Warn(msg string, span otelTrace.Span, ephemeralArgs ...any) {
	if span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		span.SetAttributes(attrs...)
		span.AddEvent(msg)
	}
	o.log(context.Background(), 3, config.LevelWarning, msg, ephemeralArgs...)
}

func (o *Observer) Error(err error, span otelTrace.Span, ephemeralArgs ...any) {
	if span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		span.SetAttributes(attrs...)
		span.RecordError(err)
	}
	o.log(context.Background(), 3, config.LevelError, err.Error(), ephemeralArgs...)
}

func (o *Observer) Fatal(err error, span otelTrace.Span, ephemeralArgs ...any) {
	if span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		span.SetAttributes(attrs...)
		span.RecordError(err)
	}
	o.log(context.Background(), 3, config.LevelFatal, err.Error(), ephemeralArgs...)
}
