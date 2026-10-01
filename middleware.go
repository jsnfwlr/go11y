package go11y

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
	"uuid"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/routers"
	oapimux "github.com/getkin/kin-openapi/routers/gorillamux"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// SetRequestIDMiddleware is a middleware that ensures each incoming HTTP request has a requestID, either the one
// provided in the incoming request headers or a new one generated if missing.
// It also adds the requestID to the response headers and ensures that the request ID is available in the request
// context for downstream handlers.
// This middleware should be added first in the middleware chain.
func SetRequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Retrieve the request ID from the incoming request headers, if it exists.
		// If not, generate a new one
		requestIDUUID, err := GetHTTPRequestID(r)
		var requestID string
		if err != nil {
			requestID = uuid.New().String()
		} else {
			requestID = requestIDUUID.String()
		}

		// Set the request ID in the context
		ctx := context.WithValue(r.Context(), RequestIDInstance, requestID)

		// Set the request ID in the response header
		w.Header().Set(RequestIDHeader, requestID)

		r = r.WithContext(ctx)

		// Call the next handler with the new context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ObserverMiddlewareMux is a middleware that adds the go11y Observer to the request context
// This allows us to use the go11y Observer in downstream handlers and middlewares without initializing it again or
// passing it explicitly
// If the Observer cannot be retrieved from the provided context, an error is returned.
// This middleware does not need to be used if the RequestLoggerMiddlewareMux is being used.
// This middleware should be added as soon as possible in the chain, ideally after the SetRequestIDMiddleware.
func ObserverMiddlewareMux(observer *Observer) (observerMiddleware mux.MiddlewareFunc, fault error) {
	if observer == nil {
		return nil, fmt.Errorf("observer cannot be nil")
	}

	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			o, err := Reset(observer)
			if err != nil {
				Error("could not reset go11y observer in observer middleware", err, SeverityHighest)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			o.Develop("Reset go11y observer for Observer middleware")

			ctx := AddToContext(r.Context(), o)

			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}

	return mw, nil
}

// Origin represents the origin details of an HTTP request
type Origin struct {
	ClientIP  string `json:"client_ip"`
	UserAgent string `json:"user_agent"`
	Method    string `json:"method"`
	Path      string `json:"path"`
}

// RequestLoggerMiddlewareMux is a middleware that logs incoming HTTP requests and their details
// It extracts tracing information from the request headers and starts a new span for the request
// It also logs the request details using go11y, adding the go11y Observer to the request context in the process
// If the Observer cannot be retrieved from the provided context, an error is returned.
// If the request context does not already contain a go11y Observer, it is added to the context.
// This middleware should be added as soon as possible in the chain, ideally after the SetRequestIDMiddleware.
// If this middleware is being used, the ObserverMiddleware does not need to be added separately.
func RequestLoggerMiddlewareMux(observer *Observer) (loggerMiddleware mux.MiddlewareFunc, fault error) {
	if observer == nil {
		return nil, fmt.Errorf("observer cannot be nil")
	}

	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Log&Trace the request
			prop := otel.GetTextMapPropagator()
			rCtx := prop.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

			o, err := Reset(observer)
			if err != nil {
				Error("could not reset go11y observer in request logger middleware", err, SeverityHighest)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			o.Develop("Reset go11y observer for RequestLogger middleware")

			requestID, err := GetContextRequestID(rCtx)
			if err != nil {
				o.Develop("Generating requestID from context")
				requestID = uuid.New()
			}

			rCtx = AddToContext(rCtx, o)

			args := []any{
				"origin",
				Origin{
					ClientIP:  r.RemoteAddr,
					UserAgent: r.UserAgent(),
					Method:    r.Method,
					Path:      r.URL.Path,
				},
				FieldRequestID, requestID.String(),
			}

			var span trace.Span

			if o.cfg.OtelURL() != "" {
				blankName := ""
				tracer := otel.Tracer(blankName) // using a blankName for the tracer results in it using the default name for the provider

				// tracer
				opts := []trace.SpanStartOption{
					trace.WithSpanKind(trace.SpanKindServer),
					trace.WithAttributes(argsToAttributes(args...)...),
				}
				rCtx, span = tracer.Start(rCtx, "HTTP "+r.Method+" "+r.URL.Path, opts...)
				defer span.End()

				args = append(
					args,
					FieldSpanID, span.SpanContext().SpanID(),
					FieldTraceID, span.SpanContext().TraceID(),
				)
			}

			_, o, err = Extend(rCtx, args...)
			if err != nil {
				Error("could not extend go11y observer in request logger middleware", err, SeverityHighest)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			b, err := io.ReadAll(r.Body)
			if err != nil {
				Error("could not read request body in request logger middleware", err, SeverityMedium)
				http.Error(w, "could not read request body", http.StatusBadRequest)
				return
			}

			// Restore the io.ReadCloser to its original state
			r.Body = io.NopCloser(bytes.NewBuffer(b))

			o.Debug("request received", "request_body", RedactBody(b))

			r = r.WithContext(rCtx)

			hw := NewHTTPWriter(w)
			// Call the next handler
			next.ServeHTTP(hw, r)

			moreArgs := []any{}
			if resp, ok := hw.(*HTTPWriter); ok {
				moreArgs = append(moreArgs, "response_body", RedactBody(resp.body))
				moreArgs = append(moreArgs, "response_status", resp.statusCode)
			}

			// Log the response
			o.Debug("request processed", moreArgs...)
		})
	}

	return mw, nil
}

// Requests is the metric for the number of requests the calling service has handled
var Requests *prometheus.CounterVec

// RequestTimes is the metric for the amount of time the calling service has taken to handle requests
var RequestTimes *prometheus.HistogramVec

// RuntimeOpts are the options used to initialise the metrics middleware
var RuntimeOpts MetricsMiddlewareMuxOpts

// MetricsMiddlewareMuxOpts are the options used to initialise the metrics middleware for a mux.Router
type MetricsMiddlewareMuxOpts struct {
	Service      string         // required - the name of the service being instrumented
	Router       *mux.Router    // required - the router for the service being instrumented. This is used to register the /internal/metrics endpoint.
	PathMaskFunc PathMask       // required - function to remove variable parts of the path for metrics aggregation. If nil, the path for metrics will not me masked
	Swagger      *openapi3.T    // optional - the swagger spec for the service being instrumented. This is used to get the endpoint names. If nil, the raw request paths are used.
	validRouter  routers.Router // the validated router created from the swagger spec
}

// PathMask is a function that takes a path string and returns a masked path string
// This can be used to remove variable parts of the path for metrics aggregation
type PathMask func(path string) (maskedPath string)

// GetMetricsMiddlewareMux initialises a promhttp metrics route on the provided mux router with a path of
// /internal/metrics and returns a mux middleware that records request-count and request-time Prometheus metrics for
// incoming HTTP requests and publishes the values on the endpoint/route.
func GetMetricsMiddlewareMux(ctxWithObserver context.Context, opts MetricsMiddlewareMuxOpts) (metricsMiddleware mux.MiddlewareFunc, fault error) {
	_, o, err := Get(ctxWithObserver)
	if err != nil {
		return nil, fmt.Errorf("could not get go11y observer from context: %w", err)
	}

	Requests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: fmt.Sprintf("%s_requests_total", opts.Service),
		Help: fmt.Sprintf("Number of requests the %s service has handled", opts.Service),
	}, []string{"endpoint", "method", "status"})

	RequestTimes = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: fmt.Sprintf("%s_requests_times", opts.Service),
		Help: fmt.Sprintf("Time %s service takes to handle requests", opts.Service),
	}, []string{"endpoint", "method", "status"})

	// Register the metrics on Prometheus endpoint
	prometheus.MustRegister(Requests)
	prometheus.MustRegister(RequestTimes)

	opts.Router.Handle("/internal/metrics", promhttp.Handler()).Methods(http.MethodGet).Name("internal-metrics")

	if opts.Swagger != nil {
		vr, err := oapimux.NewRouter(opts.Swagger)
		if err != nil {
			o.Error("error creating oapi validation router: %+v", err, SeverityHigh)
			return nil, fmt.Errorf("could not create oapi validation router: %w", err)
		}

		opts.validRouter = vr
	}

	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t0 := time.Now()

			mrw := newMiddlewareResponseWriter(w)
			// Call the next handler
			next.ServeHTTP(mrw, r)

			path := r.URL.Path

			if opts.Swagger != nil {
				route, _, err := opts.validRouter.FindRoute(r)
				if err == nil && route != nil {
					if route.Operation != nil {
						path = route.Operation.OperationID
					} else {
						path = route.Path
					}
				}
			}

			if opts.PathMaskFunc != nil {
				path = opts.PathMaskFunc(path)
			}

			requestTime := time.Since(t0)
			Requests.WithLabelValues(path, r.Method, fmt.Sprintf("%d", mrw.statusCode)).Inc()
			RequestTimes.WithLabelValues(path, r.Method, fmt.Sprintf("%d", mrw.statusCode)).Observe(requestTime.Seconds())
		})
	}

	return mw, nil
}

// MiddlewareResponseWriter is a custom http.ResponseWriter that captures the status code of the response.
type MiddlewareResponseWriter struct {
	http.ResponseWriter
	statusCode    int
	headerWritten bool
}

// WriteHeader sends an HTTP response header with the provided status code.
func (mrw *MiddlewareResponseWriter) WriteHeader(code int) {
	if !mrw.headerWritten {
		mrw.statusCode = code
		mrw.ResponseWriter.WriteHeader(code)
		mrw.headerWritten = true
	}
}

// Write writes the data to the connection as part of an HTTP reply. If WriteHeader has not yet been called, Write calls
// WriteHeader(http.StatusOK) before writing the data. Thus, explicit calls to WriteHeader are mainly used to send error
// codes.
func (mrw *MiddlewareResponseWriter) Write(b []byte) (int, error) {
	if !mrw.headerWritten {
		mrw.WriteHeader(http.StatusOK)
	}
	return mrw.ResponseWriter.Write(b)
}

func newMiddlewareResponseWriter(w http.ResponseWriter) *MiddlewareResponseWriter {
	return &MiddlewareResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
}
