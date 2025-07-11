// Package o11y provides observability features including logging, tracing, and database logging of
// roundtrip requests to third-party APIs.
package o11y

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

	"github.com/jsnfwlr/o11y/etc/migrations"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	otelAttribute "go.opentelemetry.io/otel/attribute"
	otelExportTrace "go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	otelExportTraceHTTP "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otelResource "go.opentelemetry.io/otel/sdk/resource"
	otelSDKTrace "go.opentelemetry.io/otel/sdk/trace"
	otelSemConv "go.opentelemetry.io/otel/semconv/v1.4.0"
	otelTrace "go.opentelemetry.io/otel/trace"
)

type Fields map[string]any

type Observer struct {
	cfg           Configuration
	output        io.Writer
	level         slog.Level
	logger        *slog.Logger
	traceProvider *otelSDKTrace.TracerProvider
	tracer        otelTrace.Tracer
	stableArgs    []any
	db            bool
	conn          *pgx.Conn
	pool          *pgxpool.Pool
	queries       *Queries
	spans         []otelTrace.Span
	span          otelTrace.Span
}

type logKey string

var obsKeyInstance logKey = "jsnfwlr/o11y"

var gObserver *Observer

func options(level slog.Level) *slog.HandlerOptions {
	o := &slog.HandlerOptions{
		AddSource:   true,
		Level:       level,
		ReplaceAttr: replaceAttr,
	}

	return o
}

func Initialise(ctx context.Context, cfg Configuration, logOutput io.Writer, initialArgs ...any) (ctxWithObservability context.Context, observer *Observer, fault error) {
	if logOutput == nil {
		logOutput = os.Stdout
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
		gObserver.conn, err = pgx.Connect(ctx, dbConnStr)
		if err != nil {
			return ctx, nil, fmt.Errorf("could not connect to postgres: %w", err)
		}

		gObserver.pool, err = pgxpool.New(ctx, dbConnStr)
		if err != nil {
			return ctx, nil, fmt.Errorf("could not create connection pool: %w", err)
		}

		gObserver.queries = New(gObserver.conn)

		gObserver.db = true

		dbMig, err := NewMigrator(ctx, cfg, migrations.Migrations)
		if err != nil {
			return ctx, nil, fmt.Errorf("could not create migrator: %w", err)
		}
		err = dbMig.Migrate()
		if err != nil {
			return ctx, nil, fmt.Errorf("could not migrate database: %w", err)
		}
		gObserver.Debug("Database migrated successfully")
	}

	o := gObserver
	if len(initialArgs) != 0 {
		ctx, o = Extend(ctx, initialArgs...)
	}

	slog.SetDefault(o.logger)

	return ctx, o, nil
}

func Reset(ctxWithObservability context.Context) (ctxWithResetObservability context.Context) {
	gObserver.logger = slog.New(slog.NewJSONHandler(gObserver.output, options(gObserver.level)))
	gObserver.Debug("Observer reset")
	return context.WithValue(ctxWithObservability, obsKeyInstance, gObserver)
}

func Extend(ctx context.Context, newArgs ...any) (ctxWithObservability context.Context, observer *Observer) {
	ctx, o := Get(ctx, nil)

	if len(newArgs) != 0 {
		o.logger = o.logger.With(newArgs...)
		o.stableArgs = o.AddArgs(newArgs...)
	}

	return context.WithValue(ctx, obsKeyInstance, o), o
}

func tracerProvider(ctx context.Context, cfg Configuration) (tracerProvider *otelSDKTrace.TracerProvider, fault error) {
	headers := map[string]string{
		"content-type": "application/json",
	}

	oc := otelExportTraceHTTP.NewClient(
		otelExportTraceHTTP.WithEndpoint(fmt.Sprintf("%s:%s", cfg.OtelHost(), cfg.OtelPort())),
		otelExportTraceHTTP.WithCompression(otelExportTraceHTTP.GzipCompression),
		otelExportTraceHTTP.WithHeaders(headers),
		otelExportTraceHTTP.WithInsecure(),
	)

	exporter, err := otelExportTrace.New(ctx, oc)
	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	randy := otelSDKTrace.NewTracerProvider(
		otelSDKTrace.WithBatcher(
			exporter,
			otelSDKTrace.WithMaxExportBatchSize(otelSDKTrace.DefaultMaxExportBatchSize),
			otelSDKTrace.WithBatchTimeout(otelSDKTrace.DefaultScheduleDelay*time.Millisecond),
			otelSDKTrace.WithMaxExportBatchSize(otelSDKTrace.DefaultMaxExportBatchSize),
		),
		otelSDKTrace.WithResource(
			otelResource.NewWithAttributes(
				otelSemConv.SchemaURL,
				otelSemConv.ServiceNameKey.String("corum-ng"),
			),
		),
	)

	otel.SetTracerProvider(randy)

	return randy, nil
}

func Get(ctx context.Context, span otelTrace.Span) (ctxWithTrace context.Context, observer *Observer) {
	ob := gObserver

	o := ctx.Value(obsKeyInstance)
	if o != nil {
		ob = o.(*Observer)
	}

	if span != nil {
		ob.spans = append(ob.spans, span)
		ob.span = span
	}

	return ctx, ob
}

func (o *Observer) End() {
	if o.span != nil {
		o.span.End()
	}

	if len(o.spans)-1 > 0 {
		o.span = o.spans[len(o.spans)-1]
		o.spans = o.spans[:len(o.spans)-1]
	}
}

func (o *Observer) Tracer(name string, opts ...otelTrace.TracerOption) otelTrace.Tracer {
	return o.traceProvider.Tracer(name, opts...)
}

func (o *Observer) Close() {
	if err := o.traceProvider.Shutdown(context.Background()); err != nil {
		o.Fatal(err)
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
			level = StringToLevel(fmt.Sprintf("%v", a.Value.Any()))
		}

		switch level {
		case LevelDebug:
			a.Value = slog.StringValue("DEBUG")
		case LevelInfo:
			a.Value = slog.StringValue("INFO")
		case LevelNotice:
			a.Value = slog.StringValue("NOTICE")
		case LevelWarning:
			a.Value = slog.StringValue("WARN")
		case LevelError:
			a.Value = slog.StringValue("ERR")
		case LevelFatal:
			a.Value = slog.StringValue("FATAL")
		default:
			a.Value = slog.StringValue("DEBUG")
		}
	}

	return a
}

func (o *Observer) event(span otelTrace.Span, msg string, ephemeralArgs ...any) {
	if span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		span.SetAttributes(attrs...)

		span.AddEvent(msg)
	}
}

func (o *Observer) error(span otelTrace.Span, err error, ephemeralArgs ...any) {
	if span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		span.SetAttributes(attrs...)

		span.RecordError(err)
	}
}

func argsToAttributes(combinedArgs ...any) []otelAttribute.KeyValue {
	if len(combinedArgs) == 0 {
		return nil
	}

	attrs := make([]otelAttribute.KeyValue, 0, len(combinedArgs)/2)
	for i := 0; i < len(combinedArgs); i += 2 {
		if i+1 < len(combinedArgs) {
			key := fmt.Sprintf("%v", combinedArgs[i])
			value := fmt.Sprintf("%v", combinedArgs[i+1])
			attrs = append(attrs, otelAttribute.String(key, value))
		} else {
			// If there's an odd number of arguments, the last one is ignored
			break
		}
	}

	return attrs
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
	if !o.db {
		o.Debug("Database is not enabled, skipping storage of API request")
		return nil
	}

	reqHead, err := json.Marshal(requestHeaders)
	if err != nil {
		o.Error(err, "Failed to marshal request headers to JSON")
		return err
	}

	respHead, err := json.Marshal(responseHeaders)
	if err != nil {
		o.Error(err, "Failed to marshal response headers to JSON")
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
	entry := StoreAPIRequestParams{
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
	if err := o.queries.StoreAPIRequest(ctx, entry); err != nil {
		o.Error(err, "Failed to store entry in database")
		return err
	}

	return nil
}

func (o *Observer) CheckStore() (record RemoteApiRequest, fault error) {
	if !o.db {
		return RemoteApiRequest{}, nil
	}

	record, err := o.queries.GetAPIRequests(context.Background())
	if err != nil {
		return RemoteApiRequest{}, fmt.Errorf("failed to get last remote API request: %w", err)
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

func (o *Observer) SpanContext() otelTrace.SpanContext {
	if o.span == nil {
		return otelTrace.SpanContext{}
	}

	return o.span.SpanContext()
}

func (o *Observer) Mute(ctx context.Context) {
}
