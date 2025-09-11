package go11y

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jsnfwlr/go11y/config"

	"go.opentelemetry.io/otel"
	otelAttribute "go.opentelemetry.io/otel/attribute"
	otelExportTrace "go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	otelExportTraceHTTP "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otelResource "go.opentelemetry.io/otel/sdk/resource"
	otelSDKTrace "go.opentelemetry.io/otel/sdk/trace"
	otelSemConv "go.opentelemetry.io/otel/semconv/v1.4.0"
	otelTrace "go.opentelemetry.io/otel/trace"
)

func (o *Observer) Tracer(name string, opts ...otelTrace.TracerOption) otelTrace.Tracer {
	return o.traceProvider.Tracer(name, opts...)
}

// func (o *Observer) SpanContext() otelTrace.SpanContext {
// 	if o.activeSpan == nil {
// 		return otelTrace.SpanContext{}
// 	}

// 	return o.activeSpan.SpanContext()
// }

func tracerProvider(ctx context.Context, cfg config.Configuration) (tracerProvider *otelSDKTrace.TracerProvider, fault error) {
	headers := map[string]string{
		"content-type": "application/json",
	}

	options := []otelExportTraceHTTP.Option{
		otelExportTraceHTTP.WithEndpointURL(cfg.URL()),
		otelExportTraceHTTP.WithCompression(otelExportTraceHTTP.GzipCompression),
		otelExportTraceHTTP.WithHeaders(headers),
	}

	if !strings.HasPrefix(cfg.URL(), "https://") {
		options = append(options, otelExportTraceHTTP.WithInsecure())
	}

	oc := otelExportTraceHTTP.NewClient(options...)

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
				otelSemConv.ServiceNameKey.String(cfg.ServiceName()),
			),
		),
	)

	otel.SetTracerProvider(randy)

	return randy, nil
}

func argsToAttributes(combinedArgs ...any) []otelAttribute.KeyValue {
	if len(combinedArgs) == 0 {
		return nil
	}

	dropKeys := []string{
		FieldSpanID,
		FieldTraceID,
	}
	attrs := make([]otelAttribute.KeyValue, 0, len(combinedArgs)/2)
	for i := 0; i < len(combinedArgs); i += 2 {
		if i+1 < len(combinedArgs) {
			key := fmt.Sprintf("%v", combinedArgs[i])

			if !slices.Contains(dropKeys, key) {
				switch V := combinedArgs[i+1].(type) {
				case int:
					attrs = append(attrs, otelAttribute.Int(key, V))
				case int64:
					attrs = append(attrs, otelAttribute.Int64(key, V))
				case float32:
					attrs = append(attrs, otelAttribute.Float64(key, float64(V)))
				case float64:
					attrs = append(attrs, otelAttribute.Float64(key, V))
				case bool:
					attrs = append(attrs, otelAttribute.Bool(key, V))
				case string:
					attrs = append(attrs, otelAttribute.String(key, V))
				case []string:
					attrs = append(attrs, otelAttribute.StringSlice(key, V))
				case []int:
					attrs = append(attrs, otelAttribute.IntSlice(key, V))
				case []int64:
					attrs = append(attrs, otelAttribute.Int64Slice(key, V))
				case []float64:
					attrs = append(attrs, otelAttribute.Float64Slice(key, V))
				case []bool:
					attrs = append(attrs, otelAttribute.BoolSlice(key, V))
				default:
					value := fmt.Sprintf("%v", V)
					attrs = append(attrs, otelAttribute.String(key, value))
				}
			}
		} else {
			// If there's an odd number of arguments, the last one is ignored
			break
		}
	}

	return attrs
}
