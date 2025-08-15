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
	"time"

	"github.com/jsnfwlr/go11y/config"
	"github.com/jsnfwlr/go11y/db"
	"github.com/jsnfwlr/go11y/etc/migrations"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	otelSDKTrace "go.opentelemetry.io/otel/sdk/trace"
	otelTrace "go.opentelemetry.io/otel/trace"
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
	activeSpan    otelTrace.Span
	otherSpans    []SpanTree
}

type SpanTree struct {
	Name     string
	Span     otelTrace.Span
	Children []SpanTree
}

type ObserverDB struct {
	conn    *pgx.Conn
	pool    *pgxpool.Pool
	queries *db.Queries
}

type logKey string

var obsKeyInstance logKey = "jsnfwlr/go11y"

var gObserver *Observer

func options(level slog.Level) *slog.HandlerOptions {
	o := &slog.HandlerOptions{
		AddSource:   true,
		Level:       level,
		ReplaceAttr: replaceAttr,
	}

	return o
}

func Initialise(ctx context.Context, cfg config.Configuration, logOutput io.Writer, initialArgs ...any) (ctxWithgo11y context.Context, observer *Observer, fault error) {
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

	gObserver = &Observer{
		cfg:           cfg,
		output:        logOutput,
		logger:        slog.New(slog.NewJSONHandler(logOutput, options(cfg.LogLevel()))),
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

		gObserver.db = odb

		dbMig, err := db.NewMigrator(ctx, gObserver, cfg, migrations.Migrations)
		if err != nil {
			return ctx, nil, fmt.Errorf("could not create migrator: %w", err)
		}
		err = dbMig.Migrate()
		if err != nil {
			return ctx, nil, fmt.Errorf("could not migrate database: %w", err)
		}
		gObserver.Debug("Database migrated successfully", nil)
	}

	o := gObserver
	if len(initialArgs) != 0 {
		ctx, o = Extend(ctx, initialArgs...)
	}

	slog.SetDefault(o.logger)

	return ctx, o, nil
}

func Reset(ctxWithgo11y context.Context) (ctxWithResetObservability context.Context) {
	gObserver.logger = slog.New(slog.NewJSONHandler(gObserver.output, options(gObserver.level)))
	gObserver.Debug("Observer reset", nil)
	gObserver.stableArgs = []any{}

	return context.WithValue(ctxWithgo11y, obsKeyInstance, gObserver)
}

func Extend(ctx context.Context, newArgs ...any) (ctxWithgo11y context.Context, observer *Observer) {
	o := Get(ctx)

	if len(newArgs) != 0 {
		o.logger = o.logger.With(newArgs...)
		o.stableArgs = o.AddArgs(newArgs...)
	}

	return context.WithValue(ctx, obsKeyInstance, o), o
}

func Get(ctx context.Context) (observer *Observer) {
	o := gObserver

	ob := ctx.Value(obsKeyInstance)
	if ob != nil {
		o = ob.(*Observer)
	}

	return o
}

func (o *Observer) Close() {
	if err := o.traceProvider.Shutdown(context.Background()); err != nil {
		o.Fatal(err, nil)
	}
}

func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	if os.Getenv("ENV") == "test" && a.Key == slog.TimeKey {
		return slog.Attr{} // remove time key in test to make it easier to compare
	}

	if a.Key == slog.LevelKey {
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
