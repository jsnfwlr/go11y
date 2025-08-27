package go11y

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
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

type Origin struct {
	ClientIP  string `json:"client_ip"`
	UserAgent string `json:"user_agent"`
	Method    string `json:"method"`
	Path      string `json:"path"`
}

// LogRequest is a middleware that logs incoming HTTP requests and their details
// It extracts tracing information from the request headers and starts a new span for the request
// It also logs the request details using go11y, adding the go11y Observer to the request context in the process
func LogRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log&Trace the request
		prop := otel.GetTextMapPropagator()

		ctx := prop.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		requestID := GetRequestID(ctx)

		tracer := otel.Tracer(requestID)

		ctx = Reset(ctx)

		args := []any{
			"origin", Origin{
				ClientIP:  r.RemoteAddr,
				UserAgent: r.UserAgent(),
				Method:    r.Method,
				Path:      r.URL.Path,
			},
		}

		// tracer
		opts := []trace.SpanStartOption{
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(argsToAttributes(args...)...),
		}

		ctx, span := tracer.Start(ctx, "HTTP "+r.Method+" "+r.URL.Path, opts...)

		args = append(args,
			FieldSpanID, span.SpanContext().SpanID(),
			FieldTraceID, span.SpanContext().TraceID(),
		)

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
