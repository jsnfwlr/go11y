// Package go11y provides observability features including logging, tracing, and database logging of
// roundtrip requests to third-party APIs.
package go11y

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"slices"
	"strings"
	"time"

	otelSDKTrace "go.opentelemetry.io/otel/sdk/trace"
	otelTrace "go.opentelemetry.io/otel/trace"
)

// Fields represents a set of key-value pairs for logging.
type Fields map[string]any

// Observer is the main struct for observability, containing loggers, tracer providers, and database connections.
type Observer struct {
	cfg           Configurator
	output        io.Writer
	level         slog.Level
	outLogger     *slog.Logger
	errLogger     *slog.Logger
	traceProvider *otelSDKTrace.TracerProvider
	tracer        otelTrace.Tracer
	initialArgs   []any
	stableArgs    []any
	span          otelTrace.Span
	spans         []otelTrace.Span
	skipCallers   int
	errOutput     io.Writer
	logOutput     io.Writer
}

type go11yContextKey string

var obsKeyInstance go11yContextKey = "jsnfwlr/go11y"

// Initialise sets up the Observer with the provided configuration, log outputs, and initial arguments.
func Initialise(
	ctx context.Context,
	cfg Configurator,
	logOutput, errOutput io.Writer,
	initialArgs ...any,
) (
	ctxWithGo11y context.Context,
	observer *Observer,
	fault error,
) {
	if logOutput == nil {
		logOutput = os.Stdout
	}

	if errOutput == nil {
		errOutput = os.Stderr
	}

	var err error

	if cfg == nil {
		cfg, err = LoadConfig()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load configuration: %w", err)
		}
	}

	tp, err := tracerProvider(ctx, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create tracer: %w", err)
	}

	opts := defaultOptions(cfg)

	o := &Observer{
		cfg:           cfg,
		output:        logOutput,
		outLogger:     slog.New(slog.NewJSONHandler(logOutput, opts)),
		errLogger:     slog.New(slog.NewJSONHandler(errOutput, opts)),
		logOutput:     logOutput,
		errOutput:     errOutput,
		traceProvider: tp,
		stableArgs:    initialArgs,
		initialArgs:   initialArgs,
		skipCallers:   3, // default to 3 but allow it to be increased via o.IncreaseDistance()
	}

	ctx = context.WithValue(ctx, obsKeyInstance, o)
	if len(initialArgs) != 0 {
		ctx, o, err = Extend(ctx, initialArgs...)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to extend context with initial arguments: %w", err)
		}
	}

	slog.SetDefault(o.outLogger)

	o.Debug("Initialised observer with context")

	return ctx, o, nil
}

// Reset resets the Observer in the context to its initial state.
func Reset(originalObserver *Observer) (observer *Observer, fault error) {
	if originalObserver == nil {
		return nil, fmt.Errorf("observer cannot be nil")
	}

	newO := &Observer{
		cfg:           originalObserver.cfg,
		output:        originalObserver.output,
		outLogger:     slog.New(slog.NewJSONHandler(originalObserver.logOutput, defaultOptions(originalObserver.cfg))),
		errLogger:     slog.New(slog.NewJSONHandler(originalObserver.errOutput, defaultOptions(originalObserver.cfg))),
		traceProvider: originalObserver.traceProvider,
		skipCallers:   originalObserver.skipCallers,
		stableArgs:    originalObserver.initialArgs,
		initialArgs:   originalObserver.initialArgs,
	}

	return newO, nil
}

// Get retrieves the Observer from the context. If none exists, it initializes a new one with default settings.
func Get(ctx context.Context) (ctxWithObserver context.Context, observer *Observer, fault error) {
	ob := ctx.Value(obsKeyInstance)
	if ob == nil {
		return ctx, nil, fmt.Errorf("go11y Observer not found in context - please initialise go11y first")
	}

	if o, ok := ob.(*Observer); ok {
		return ctx, o, nil
	}
	return ctx, nil, fmt.Errorf("go11y Observer not found in context - please initialise go11y first")
}

// Extend retrieves the Observer from the context and adds new arguments to its logger.
// If no Observer exists in the context, it initializes a new one with default settings and adds the arguments.
// LLMs will report that this function mutates the Observer in place, but this is intentional to allow for dynamic
// updates to the logger's stable arguments.
func Extend(ctx context.Context, newArgs ...any) (ctxWithGo11y context.Context, observer *Observer, fault error) {
	ctx, o, err := Get(ctx)
	if err != nil {
		return ctx, nil, err
	}

	// add the newArgs to the existing stableArgs and update the loggers
	if len(newArgs) != 0 {
		o.outLogger = o.outLogger.With(newArgs...)
		o.errLogger = o.errLogger.With(newArgs...)
		o.stableArgs = o.AddArgs(newArgs...)
	}

	return context.WithValue(ctx, obsKeyInstance, o), o, nil
}

// Span gets the Observer from the context and starts a new tracing span with the given name.
// If no Observer exists in the context, it initializes a new one with default settings and starts the span.
// The tracing equivalent of Get()
func Span(
	ctx context.Context,
	tracer otelTrace.Tracer,
	spanName string,
	spanKind otelTrace.SpanKind,
) (
	ctxWithSpan context.Context,
	observer *Observer,
	fault error,
) {
	ctx, o, err := Get(ctx)
	if err != nil {
		return ctx, nil, err
	}

	ctx, span := tracer.Start(ctx, spanName, otelTrace.WithSpanKind(spanKind))

	o.span = span
	o.spans = append(o.spans, span)

	return context.WithValue(ctx, obsKeyInstance, o), o, nil
}

// Expand retrieves the Observer from the context, starts a new tracing span with the given name, and adds new arguments
// to its logger. If no Observer exists in the context, it initializes a new one with default settings and adds the
// arguments.
// This is effectively Extend() plus Span() in one function, and is useful for reducing boilerplate in handlers
// and middlewares.
// LLMs will report that this function mutates the Observer in place, but this is intentional to allow for dynamic
// updates to the logger's stable arguments.
func Expand(
	ctx context.Context,
	tracer otelTrace.Tracer,
	spanName string,
	spanKind otelTrace.SpanKind,
	newArgs ...any,
) (
	ctxWithSpan context.Context,
	observer *Observer,
	fault error,
) {
	ctx, o, err := Span(ctx, tracer, spanName, spanKind)
	if err != nil {
		return ctx, nil, err
	}

	if len(newArgs) != 0 {
		o.outLogger = o.outLogger.With(newArgs...)
		o.errLogger = o.errLogger.With(newArgs...)
		o.stableArgs = o.AddArgs(newArgs...)
	}

	return context.WithValue(ctx, obsKeyInstance, o), o, nil
}

// Close ends all active spans and shuts down the trace provider to ensure all traces are flushed.
// This does not return any error, but logs an error if the trace provider fails to shut down properly.
func (o *Observer) Close() {
	if o.span != nil {
		o.span.End()

		for _, s := range o.spans {
			s.End()
		}
	}
	if o.traceProvider != nil {
		if err := o.traceProvider.Shutdown(context.Background()); err != nil {
			o.Error("could not shut down tracer", err, SeverityMedium)
		}
	}
}

// defaultReplacer creates a function to replace or modify log attributes
func defaultReplacer(trimModules, trimPaths []string) func(groups []string, a slog.Attr) slog.Attr {
	return func(groups []string, a slog.Attr) slog.Attr {
		if os.Getenv("ENV") == "test" && a.Key == slog.TimeKey {
			return slog.Attr{} // remove time key in test to make it easier to compare
		}

		switch a.Key {
		case slog.SourceKey:
			source, ok := a.Value.Any().(*slog.Source)
			if !ok {
				return a
			}

			for _, path := range trimPaths {
				if idx := strings.Index(source.File, path); idx != -1 {
					source.File = source.File[idx+len(path):]
				}
			}

			for _, module := range trimModules {
				if idx := strings.Index(source.Function, module); idx != -1 {
					source.Function = source.Function[idx+len(module):]
				}
			}

			return slog.Any(a.Key, source)
		case slog.LevelKey:
			var level slog.Level

			if lvl, ok := a.Value.Any().(slog.Level); ok {
				level = lvl
			} else {
				level = StringToLevel(fmt.Sprintf("%v", a.Value.Any()))
			}

			a.Value = LevelToValue(level) // moved to a function to keep the translation alongside the level definitions
		}

		return a
	}
}

func (o *Observer) log(skipCallers int, level slog.Level, msg string, args ...any) (levelEnabled bool) {
	if o.outLogger == nil || !o.outLogger.Enabled(context.Background(), level) {
		return false
	}
	var pc uintptr
	var pcs [1]uintptr
	// skip [runtime.Callers, this function, this function's caller]
	runtime.Callers(skipCallers, pcs[:])
	pc = pcs[0]

	r := slog.NewRecord(time.Now(), level, msg, pc)

	if len(args) != 0 {
		r.Add(DeduplicateArgs(args)...)
	}

	_ = o.outLogger.Handler().Handle(context.Background(), r)

	return true
}

func (o *Observer) error(skipCallers int, level slog.Level, msg string, args ...any) (levelEnabled bool) {
	if o.errLogger == nil || !o.errLogger.Enabled(context.Background(), level) {
		return false
	}
	var pc uintptr
	var pcs [1]uintptr
	// skip [runtime.Callers, this function, this function's caller]
	runtime.Callers(skipCallers, pcs[:])
	pc = pcs[0]

	r := slog.NewRecord(time.Now(), level, msg, pc)

	if len(args) != 0 {
		r.Add(DeduplicateArgs(args)...)
	}

	_ = o.errLogger.Handler().Handle(context.Background(), r)

	return true
}

// AddArgs processes the provided arguments, ensuring that they are stable, unique, ordered, and formatted correctly.
func (o *Observer) AddArgs(args ...any) (filteredArgs []any) {
	args = append(o.stableArgs, args...)

	exArgs := map[any]any{}

	// Deduplicate the arguments
	for len(args) > 1 {
		exArgs[args[0]] = args[1]
		args = args[2:]
	}

	keys := make([]string, 0, len(exArgs))
	for k := range exArgs {
		keys = append(keys, fmt.Sprintf("%v", k))
	}

	// Sort the keys to maintain a consistent order
	slices.Sort(keys)

	resArgs := make([]any, 0, len(exArgs)*2)
	for _, k := range keys {
		resArgs = append(resArgs, k, exArgs[k])
	}

	return resArgs
}

// End ends the current tracing span and reverts to the previous span in the stack.
func (o *Observer) End() {
	if o.span != nil {
		o.span.End()
	}

	if len(o.spans) > 0 {
		o.spans = o.spans[:len(o.spans)-1]
		if len(o.spans) > 0 {
			o.span = o.spans[len(o.spans)-1]
		} else {
			o.span = nil
		}
	}
}

// InContext can be used to check if go11y has been added to a context before calling go11y.Get()
// This is useful for other packages imported by services that use go11y as well as other services that still use the
// go-logging package.
func InContext(ctx context.Context) (response bool) {
	return (ctx.Value(obsKeyInstance) != nil)
}

// IncreaseDistance increases the caller skip distance for logging purposes.
// This is useful when wrapping go11y (such as the go-common splitLog)
func (o *Observer) IncreaseDistance(distance int) {
	o.skipCallers += distance
}

// DecreaseDistance decreases the caller skip distance for logging purposes.
// This is useful when wrapping go11y (such as the go-common splitLog)
func (o *Observer) DecreaseDistance(distance int) {
	o.skipCallers -= distance
}

// SetDistance sets the caller skip distance for logging purposes.
// This is useful when wrapping go11y (such as the go-common splitLog)
func (o *Observer) SetDistance(distance int) {
	o.skipCallers = distance
}

// AddToContext adds the Observer to the provided context.
// This is useful for reducing boilerplate in handlers and middlewares.
func AddToContext(ctx context.Context, o *Observer) context.Context {
	return context.WithValue(ctx, obsKeyInstance, o)
}
