package o11y

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type requestIDKey string

const RequestIDInstance requestIDKey = "requestID"

func GetRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	if requestID, ok := ctx.Value(RequestIDInstance).(string); ok {
		return requestID
	}

	return ""
}

func SetRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Generate a new request ID
		requestID := uuid.New().String()

		// Set the request ID in the context
		ctx := context.WithValue(r.Context(), RequestIDInstance, requestID)

		// Set the request ID in the response header
		w.Header().Set("X-Request-ID", requestID)

		// Call the next handler with the new context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func LogRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log the request
		ctx := r.Context()

		requestID := GetRequestID(ctx)

		tracer := otel.Tracer(requestID)

		remoteTraceID := r.Header.Get("X-Trace-ID")
		remoteSpanID := r.Header.Get("X-Span-ID")

		ctx, span := tracer.Start(ctx, "HTTP "+r.Method+" "+r.URL.Path, trace.WithSpanKind(trace.SpanKindServer))

		args := []any{
			FieldRequestMethod, r.Method,
			FieldRequestPath, r.URL.Path,
			FieldRequestID, requestID,
			FieldSpanID, span.SpanContext().SpanID(),
			FieldTraceID, span.SpanContext().TraceID(),
		}
		if remoteTraceID != "" {
			args = append(args, "remote_trace_id", remoteTraceID)
		}
		if remoteSpanID != "" {
			args = append(args, "remote_span_id", remoteSpanID)
		}

		ctx = Reset(ctx)

		ctx, o := Extend(ctx, args...)

		o.Debug("request received", span)

		r = r.WithContext(ctx)

		// Call the next handler
		next.ServeHTTP(w, r)

		// Log the response
		// log.Printf("Response sent for: %s %s", r.Method, r.URL.Path)
		o.Debug("request processed", span)
		span.End()
	})
}
