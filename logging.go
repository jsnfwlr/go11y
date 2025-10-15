package go11y

import (
	"context"
	"os"
)

// Develop records an event on the tracing span if it is available and logs a develop message via the observer (if the observer's log-level allows).
// This is intended for use during development and may be too verbose or could leak secrets in production use, and should be filtered out in such environments.
func (o *Observer) Develop(msg string, ephemeralArgs ...any) {
	logged := o.log(context.Background(), 3, LevelDevelop, msg, ephemeralArgs...)

	if logged && o.span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		o.span.SetAttributes(attrs...)
		o.span.AddEvent(msg)
	}
}

// Debug records an event on the tracing span if it is available and logs a debug message via the observer (if the observer's log-level allows).
func (o *Observer) Debug(msg string, ephemeralArgs ...any) {
	logged := o.log(context.Background(), 3, LevelDebug, msg, ephemeralArgs...)

	if logged && o.span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		o.span.SetAttributes(attrs...)
		o.span.AddEvent(msg)
	}
}

// Info records an event on the tracing span if it is available and logs an information message via the observer (if the observer's log-level allows).
func (o *Observer) Info(msg string, ephemeralArgs ...any) {
	logged := o.log(context.Background(), 3, LevelInfo, msg, ephemeralArgs...)

	if logged && o.span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		o.span.SetAttributes(attrs...)
		o.span.AddEvent(msg)
	}
}

// Notice records an event on the tracing span if it is available and logs a notice message via the observer (if the observer's log-level allows).
func (o *Observer) Notice(msg string, ephemeralArgs ...any) {
	logged := o.log(context.Background(), 3, LevelNotice, msg, ephemeralArgs...)

	if logged && o.span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		o.span.SetAttributes(attrs...)
		o.span.AddEvent(msg)
	}
}

// Warning records an event on the tracing span if it is available and logs a warning message via the observer (if the observer's log-level allows).
func (o *Observer) Warning(msg string, ephemeralArgs ...any) {
	logged := o.log(context.Background(), 3, LevelWarning, msg, ephemeralArgs...)

	if logged && o.span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		o.span.SetAttributes(attrs...)
		o.span.AddEvent(msg)
	}
}

// Warn is an alias for Warning to maintain compatibility with other logging libraries.
func (o *Observer) Warn(msg string, ephemeralArgs ...any) {
	o.Warning(msg, ephemeralArgs...)
}

// Error records an error on the tracing span if it is available and logs an error message via the observer (if the observer's log-level allows), with the
// specified severity level.
func (o *Observer) Error(msg string, err error, severity string, ephemeralArgs ...any) {
	ephemeralArgs = append(ephemeralArgs, "error", err.Error(), "severity", severity)
	logged := o.log(context.Background(), 3, LevelError, msg, ephemeralArgs...)

	if logged && o.span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		o.span.SetAttributes(attrs...)
		o.span.RecordError(err)
	}
}

// Fatal records an error on the tracing span if it is available and logs a fatal error message via the observer with the
// highest severity level and then exits the application with a status code of 1.
func (o *Observer) Fatal(msg string, err error, ephemeralArgs ...any) {
	ephemeralArgs = append(ephemeralArgs, "error", err.Error(), "severity", SeverityHighest)
	logged := o.log(context.Background(), 3, LevelFatal, msg, ephemeralArgs...)

	if logged && o.span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		o.span.SetAttributes(attrs...)
		o.span.RecordError(err)
	}
	os.Exit(1)
}

// Fatal logs a fatal error message with the highest severity level and then exits the application with a status code of 1.
// This is intended to for use in situations where an Observer instance is not available such as in the main function before the observer has been initialised.
func Fatal(msg string, err error, ephemeralArgs ...any) {
	ctx := context.Background()
	cfg := CreateConfig(LevelFatal, "", "", "", nil, nil)
	_, o, _ := Initialise(ctx, cfg, os.Stderr, nil)
	ephemeralArgs = append(ephemeralArgs, "error", err.Error(), "severity", SeverityHighest)
	o.log(context.Background(), 3, LevelFatal, msg, ephemeralArgs...)
	os.Exit(1)
}
