package go11y

import (
	"context"
	"fmt"
	"os"
	"slices"
)

// Develop logs a development-only message and adds an event to the span if available.
// $msg is the message to log
// $ephemeralArgs are any additional key-value pairs to include in the log and span attributes.
func (o *Observer) Develop(msg string, ephemeralArgs ...any) {
	logged := o.log(o.skipCallers, LevelDevelop, msg, ephemeralArgs...)
	if logged && o.span != nil {
		attrs := argsToAttributes(append(o.stableArgs, ephemeralArgs)...)
		o.span.SetAttributes(attrs...)
		o.span.AddEvent(msg)
	}
}

// Debug logs a debug message and adds an event to the span if available.
// $msg is the message to log
// $ephemeralArgs are any additional key-value pairs to include in the log and span attributes
func (o *Observer) Debug(msg string, ephemeralArgs ...any) {
	logged := o.log(o.skipCallers, LevelDebug, msg, ephemeralArgs...)
	if logged && o.span != nil {
		attrs := argsToAttributes(append(o.stableArgs, ephemeralArgs)...)
		o.span.SetAttributes(attrs...)
		o.span.AddEvent(msg)
	}
}

// Info logs an informational message and adds an event to the span if available.
// $msg is the message to log
// $ephemeralArgs are any additional key-value pairs to include in the log and span attributes.
func (o *Observer) Info(msg string, ephemeralArgs ...any) {
	logged := o.log(o.skipCallers, LevelInfo, msg, ephemeralArgs...)
	if logged && o.span != nil {
		attrs := argsToAttributes(append(o.stableArgs, ephemeralArgs)...)
		o.span.SetAttributes(attrs...)
		o.span.AddEvent(msg)
	}
}

// Notice logs a notice message and adds an event to the span if available.
// $msg is the message to log
// $ephemeralArgs are any additional key-value pairs to include in the log and span attributes.
func (o *Observer) Notice(msg string, ephemeralArgs ...any) {
	logged := o.log(o.skipCallers, LevelNotice, msg, ephemeralArgs...)
	if logged && o.span != nil {
		attrs := argsToAttributes(append(o.stableArgs, ephemeralArgs)...)
		o.span.SetAttributes(attrs...)
		o.span.AddEvent(msg)
	}
}

// Warning logs a warning message and adds an event to the span if available.
// $msg is the message to log
// $ephemeralArgs are any additional key-value pairs to include in the log and span attributes.
func (o *Observer) Warning(msg string, ephemeralArgs ...any) {
	logged := o.log(o.skipCallers, LevelWarning, msg, ephemeralArgs...)
	if logged && o.span != nil {
		attrs := argsToAttributes(append(o.stableArgs, ephemeralArgs)...)
		o.span.SetAttributes(attrs...)
		o.span.AddEvent(msg)
	}
}

// Warn a backward compatibility alias for Warning.
// $msg is the message to log
// $ephemeralArgs are any additional key-value pairs to include in the log and span attributes.
func (o *Observer) Warn(msg string, ephemeralArgs ...any) {
	logged := o.log(o.skipCallers, LevelWarning, msg, ephemeralArgs...)
	if logged && o.span != nil {
		attrs := argsToAttributes(append(o.stableArgs, ephemeralArgs)...)
		o.span.SetAttributes(attrs...)
		o.span.AddEvent(msg)
	}
}

// Error logs an error message, records the error in the span if available, and sets the severity.
// $msg is the message to log
// $err is the error to record in the span and include in the log
// $severity is a string representing the severity of the error (e.g., "low", "medium", "high")
// $ephemeralArgs are any additional key-value pairs to include in the log and span attributes.
func (o *Observer) Error(msg string, err error, severity string, ephemeralArgs ...any) {
	if err == nil {
		panic(fmt.Sprintf("Error cannot be nil. Use Info or Debug for non-error messages. Message: %s", msg))
	}
	logged := o.error(o.skipCallers, LevelError, msg, append(ephemeralArgs, "error", err.Error(), "severity", severity)...)
	if logged && o.span != nil {
		attrs := argsToAttributes(append(o.stableArgs, ephemeralArgs)...)
		o.span.SetAttributes(attrs...)
		o.span.RecordError(err)
	}
}

// Fatal logs a fatal error message with the highest severity, records the error in the span if available, and then
// exits the application abruptly.
// $msg is the message to log
// $err is the error to record in the span and include in the log
// $ephemeralArgs are any additional key-value pairs to include in the log and span attributes.
func (o *Observer) Fatal(msg string, err error, ephemeralArgs ...any) {
	if err == nil {
		panic(fmt.Sprintf("Error cannot be nil. Use Info or Debug for non-error messages. Message: %s", msg))
	}

	logged := o.error(o.skipCallers, LevelFatal, msg, append(ephemeralArgs, "error", err.Error(), "severity", SeverityHighest)...)
	if logged && o.span != nil {
		attrs := argsToAttributes(append(o.stableArgs, ephemeralArgs)...)
		o.span.SetAttributes(attrs...)
		o.span.RecordError(err)
	}

	os.Exit(1)
}

// Panic logs a fatal error message with the highest severity, records the error in the span if available, and then
// exits the application cleanly.
// $msg is the message to log
// $err is the error to record in the span and include in the log
// $ephemeralArgs are any additional key-value pairs to include in the log and span attributes.
func (o *Observer) Panic(msg string, err error, ephemeralArgs ...any) {
	if err == nil {
		panic(fmt.Sprintf("Error cannot be nil. Use Info or Debug for non-error messages. Message: %s", msg))
	}

	logged := o.error(o.skipCallers, LevelPanic, msg, append(ephemeralArgs, "error", err.Error(), "severity", SeverityHighest)...)
	if logged && o.span != nil {
		attrs := argsToAttributes(append(o.stableArgs, ephemeralArgs)...)
		o.span.SetAttributes(attrs...)
		o.span.RecordError(err)
	}

	panic(msg)
}

// Panic is intended to be called before the observer has been configured and the context lacks an observer.
// It will log the fatal error to stderr in the JSON format used by go11y and exit the application cleanly
// $msg is the message to log
// $err is the error to record in the span and include in the log
// $ephemeralArgs are any additional key-value pairs to include in the log and span attributes.
func Panic(msg string, err error, ephemeralArgs ...any) {
	if err == nil {
		panic(fmt.Sprintf("Error cannot be nil. Use Info or Debug for non-error messages. Message: %s", msg))
	}

	cfg := &Configuration{
		logLevel:    LevelInfo,
		otelURL:     "",
		strLevel:    "info",
		databaseURL: "",
		serviceName: "",
		trimModules: []string{},
		trimPaths:   []string{},
	}

	_, o, _ := Initialise(context.Background(), cfg, nil, os.Stderr)
	ephemeralArgs = append(ephemeralArgs, "error", err.Error(), "severity", SeverityHighest)
	o.error(o.skipCallers, LevelPanic, msg, ephemeralArgs...)

	panic(msg)
}

// Fatal is intended to be called before the observer has been configured and the context lacks an observer.
// It will log the fatal error to stderr in the JSON format used by go11y and exit the application abruptly.
// $msg is the message to log
// $err is the error to record in the span and include in the log
// $exitCode is the code to exit the application with (defaults to 1 if less than 1)
// $ephemeralArgs are any additional key-value pairs to include in the log and span attributes.
func Fatal(msg string, err error, exitCode int, ephemeralArgs ...any) {
	if err == nil {
		panic(fmt.Sprintf("Error cannot be nil. Use Info or Debug for non-error messages. Message: %s", msg))
	}

	cfg := &Configuration{
		logLevel:    LevelInfo,
		otelURL:     "",
		strLevel:    "info",
		databaseURL: "",
		serviceName: "",
		trimModules: []string{},
		trimPaths:   []string{},
	}

	_, o, _ := Initialise(context.Background(), cfg, nil, os.Stderr)
	ephemeralArgs = append(ephemeralArgs, "error", err.Error(), "severity", SeverityHighest)
	o.error(o.skipCallers, LevelFatal, msg, ephemeralArgs...)

	if exitCode < 1 {
		exitCode = 1
	}

	os.Exit(exitCode)
}

// Error is intended to be called before the observer has been configured and the context lacks an observer.
// It will log the error to stderr in the JSON format used by go11y.
// $msg is the message to log
// $err is the error to record in the span and include in the log
// $severity is a string representing the severity of the error (e.g., "low", "medium", "high")
// $ephemeralArgs are any additional key-value pairs to include in the log and span attributes.
func Error(msg string, err error, severity string, ephemeralArgs ...any) {
	if err == nil {
		panic(fmt.Sprintf("Error cannot be nil. Use Info or Debug for non-error messages. Message: %s", msg))
	}

	cfg := &Configuration{
		logLevel:    LevelInfo,
		otelURL:     "",
		strLevel:    "info",
		databaseURL: "",
		serviceName: "",
		trimModules: []string{},
		trimPaths:   []string{},
	}

	_, o, _ := Initialise(context.Background(), cfg, nil, os.Stderr)
	ephemeralArgs = append(ephemeralArgs, "error", err.Error(), "severity", severity)
	o.error(o.skipCallers, LevelError, msg, ephemeralArgs...)
}

// DeduplicateArgs removes duplicate keys from a list of key-value pairs.
func DeduplicateArgs(args []any) (deduped []any) {
	keys := []string{}
	uniq := []any{}

	for i := 0; i < len(args); i += 2 {
		if len(args) >= i+2 {
			key := fmt.Sprintf("%v", args[i])
			if slices.Contains(keys, key) {
				continue
			}

			keys = append(keys, key)
			uniq = append(uniq, args[i], args[i+1])
		}
	}

	return uniq
}
