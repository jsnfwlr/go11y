// Package go11y provides observability features including logging, tracing, and database logging of
// roundtrip requests to third-party APIs.
package go11y

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/jsnfwlr/go11y/config"
	"github.com/jsnfwlr/go11y/db"
	"github.com/jsnfwlr/go11y/etc/migrations"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	otelSDKTrace "go.opentelemetry.io/otel/sdk/trace"
	otelTrace "go.opentelemetry.io/otel/trace"
	_ "google.golang.org/genproto/googleapis/api/httpbody"
	_ "google.golang.org/genproto/googleapis/rpc/status"
)

type Fields map[string]any

type Observer struct {
	cfg           config.Configuration
	output        io.Writer
	level         slog.Level
	logger        *slog.Logger
	traceProvider *otelSDKTrace.TracerProvider
	tracer        otelTrace.Tracer
	stableArgs    []any
	db            *ObserverDB
	span          otelTrace.Span
	spans         []otelTrace.Span
}

type ObserverDB struct {
	conn    *pgx.Conn
	pool    *pgxpool.Pool
	queries *db.Queries
}

type go11yContextKey string

var obsKeyInstance go11yContextKey = "jsnfwlr/go11y"

var og *Observer

func options(cfg config.Configuration) *slog.HandlerOptions {
	ho := &slog.HandlerOptions{
		AddSource:   true,
		Level:       cfg.LogLevel(),
		ReplaceAttr: MakeReplacer(cfg.TrimModules(), cfg.TrimPaths()),
	}

	return ho
}

func Initialise(ctx context.Context, cfg config.Configuration, logOutput io.Writer, initialArgs ...any) (ctxWithGo11y context.Context, observer *Observer, fault error) {
	if logOutput == nil {
		logOutput = os.Stdout
	}

	var err error

	if cfg == nil {
		cfg, err = config.Load()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load configuration: %w", err)
		}
	}

	tp, err := tracerProvider(ctx, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create tracer: %w", err)
	}

	opts := options(cfg)

	og = &Observer{
		cfg:           cfg,
		output:        logOutput,
		logger:        slog.New(slog.NewJSONHandler(logOutput, opts)),
		traceProvider: tp,
		stableArgs:    initialArgs,
	}

	dbConnStr := cfg.DBConStr()
	if dbConnStr != "" {
		odb := &ObserverDB{}

		odb.conn, err = pgx.Connect(ctx, dbConnStr)
		if err != nil {
			return ctx, nil, fmt.Errorf("could not connect to postgres: %w", err)
		}

		odb.pool, err = pgxpool.New(ctx, dbConnStr)
		if err != nil {
			return ctx, nil, fmt.Errorf("could not create connection pool: %w", err)
		}

		odb.queries = db.New(odb.conn)

		og.db = odb

		dbMig, err := db.NewMigrator(ctx, og, cfg, migrations.Migrations)
		if err != nil {
			return ctx, nil, fmt.Errorf("could not create migrator: %w", err)
		}
		err = dbMig.Migrate()
		if err != nil {
			return ctx, nil, fmt.Errorf("could not migrate database: %w", err)
		}
		og.Debug("Database migrated successfully", nil)
	}

	ctx = context.WithValue(ctx, obsKeyInstance, og)
	if len(initialArgs) != 0 {
		ctx, og = Extend(ctx, initialArgs...)
	}

	slog.SetDefault(og.logger)

	fmt.Println("Initialised observer with context")

	return ctx, og, nil
}

func Reset(ctxWithGo11y context.Context) (ctxWithResetObservability context.Context) {
	og.logger = slog.New(slog.NewJSONHandler(og.output, options(og.cfg)))
	og.Debug("Observer reset", nil)
	og.stableArgs = []any{}

	return context.WithValue(ctxWithGo11y, obsKeyInstance, og)
}

// Extend retrieves the Observer from the context and adds new arguments to its logger.
// If no Observer exists in the context, it initializes a new one with default settings and adds the arguments.
func Extend(ctx context.Context, newArgs ...any) (ctxWithGo11y context.Context, observer *Observer) {
	ctx, o := Get(ctx)

	if len(newArgs) != 0 {
		o.logger = o.logger.With(newArgs...)
		o.stableArgs = o.AddArgs(newArgs...)
	}

	return context.WithValue(ctx, obsKeyInstance, o), o
}

func Expand(ctx context.Context, tracer otelTrace.Tracer, spanName string, spanKind otelTrace.SpanKind, newArgs ...any) (ctxWithSpan context.Context, observer *Observer) {
	ctx, o := Span(ctx, tracer, spanName, spanKind)

	if len(newArgs) != 0 {
		o.logger = o.logger.With(newArgs...)
		o.stableArgs = o.AddArgs(newArgs...)
	}

	return context.WithValue(ctx, obsKeyInstance, o), o
}

// Get retrieves the Observer from the context. If none exists, it initializes a new one with default settings.
func Get(ctx context.Context) (ctxWithObserver context.Context, observer *Observer) {
	ob := ctx.Value(obsKeyInstance)
	if ob == nil {
		return context.WithValue(ctx, obsKeyInstance, og), og
	}

	o := ob.(*Observer)

	return ctx, o
}

// Span gets the Observer from the context and starts a new tracing span with the given name.
// If no Observer exists in the context, it initializes a new one with default settings and starts the span.
// The tracing equivalent of Get()
func Span(ctx context.Context, tracer otelTrace.Tracer, spanName string, spanKind otelTrace.SpanKind) (ctxWithSpan context.Context, observer *Observer) {
	ctx, o := Get(ctx)

	ctx, span := tracer.Start(ctx, spanName, otelTrace.WithSpanKind(spanKind))

	o.span = span
	o.spans = append(o.spans, span)

	return context.WithValue(ctx, obsKeyInstance, o), o
}

func (o *Observer) Close() {
	if o.span != nil {
		o.span.End()

		for _, s := range o.spans {
			s.End()
		}
	}

	if err := o.traceProvider.Shutdown(context.Background()); err != nil {
		o.Fatal(err, nil)
	}
}

func MakeReplacer(trimModules, trimPaths []string) func(groups []string, a slog.Attr) slog.Attr {
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
				level = config.StringToLevel(fmt.Sprintf("%v", a.Value.Any()))
			}

			switch level {
			case config.LevelDebug:
				a.Value = slog.StringValue("DEBUG")
			case config.LevelInfo:
				a.Value = slog.StringValue("INFO")
			case config.LevelNotice:
				a.Value = slog.StringValue("NOTICE")
			case config.LevelWarning:
				a.Value = slog.StringValue("WARN")
			case config.LevelError:
				a.Value = slog.StringValue("ERR")
			case config.LevelFatal:
				a.Value = slog.StringValue("FATAL")
			default:
				a.Value = slog.StringValue("DEBUG")
			}
		}

		return a
	}
}

func (o *Observer) log(ctx context.Context, skipCallers int, level slog.Level, msg string, args ...any) {
	if o.logger == nil || !o.logger.Enabled(ctx, level) {
		return
	}
	var pc uintptr
	var pcs [1]uintptr
	// skip [runtime.Callers, this function, this function's caller]
	runtime.Callers(skipCallers, pcs[:])
	pc = pcs[0]

	r := slog.NewRecord(time.Now(), level, msg, pc)

	if len(args) != 0 {
		r.Add(args...)
	}

	if ctx == nil {
		ctx = context.Background()
	}
	_ = o.logger.Handler().Handle(ctx, r)
}

func (o *Observer) store(ctx context.Context, url, method string, statusCode int32, duration time.Duration, requestBody, responseBody []byte, requestHeaders, responseHeaders http.Header) (fault error) {
	if o.db == nil {
		o.Debug("Database is not enabled, skipping storage of API request", nil)
		return nil
	}

	reqHead, err := json.Marshal(requestHeaders)
	if err != nil {
		o.Error(err, nil, "msg", "Failed to marshal request headers to JSON")
		return err
	}

	respHead, err := json.Marshal(responseHeaders)
	if err != nil {
		o.Error(err, nil, "msg", "Failed to marshal response headers to JSON")
		return err
	}

	rqB := pgtype.Text{
		String: string(requestBody),
		Valid:  len(requestBody) > 0,
	}

	rsB := pgtype.Text{
		String: string(responseBody),
		Valid:  len(responseBody) > 0,
	}

	// Create a new entry in the database
	entry := db.StoreAPIRequestParams{
		Url:             url,
		Method:          method,
		StatusCode:      statusCode,
		RequestBody:     rqB,
		RequestHeaders:  reqHead,
		ResponseBody:    rsB,
		ResponseHeaders: respHead,
		ResponseTimeMs:  int64(duration),
	}

	// Store the entry in the database
	if err := o.db.queries.StoreAPIRequest(ctx, entry); err != nil {
		o.Error(err, nil, "msg", "Failed to store entry in database")
		return err
	}

	return nil
}

func (o *Observer) CheckStore() (record db.RemoteApiRequest, fault error) {
	if o.db == nil {
		return db.RemoteApiRequest{}, nil
	}

	record, err := o.db.queries.GetAPIRequests(context.Background())
	if err != nil {
		return db.RemoteApiRequest{}, fmt.Errorf("failed to get last remote API request: %w", err)
	}

	return record, nil
}

// AddArgs processes the provided arguments, ensuring that they are stable and formatted correctly.
func (o *Observer) AddArgs(args ...any) (filteredArgs []any) {
	args = append(o.stableArgs, args...)

	exArgs := map[any]any{}

	for len(args) > 0 {
		exArgs, args = processArgs(exArgs, args)
	}

	resArgs := make([]any, 0, len(exArgs)/2)
	for k, v := range exArgs {
		resArgs = append(resArgs, k, v)
	}

	return resArgs
}

func processArgs(exArgs map[any]any, args []any) (map[any]any, []any) {
	if len(args) < 2 {
		return exArgs, []any{}
	}

	exArgs[args[0]] = args[1]

	return exArgs, args[2:]
}

func (o *Observer) Mute(ctx context.Context) {
}

func (o *Observer) End() {
	o.span.End()

	o.spans = o.spans[:len(o.spans)-1]
	if len(o.spans) > 0 {
		o.span = o.spans[len(o.spans)-1]
	} else {
		o.span = nil
	}
}
