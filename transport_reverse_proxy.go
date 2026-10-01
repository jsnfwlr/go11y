package go11y

import (
	"context"
	"fmt"
	"net/http/httputil"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// ReverseProxy is a wrapper around httputil.ReverseProxy that provides methods
// to add OpenTelemetry tracing, propagation, logging, and database storage functionality.
type ReverseProxy struct {
	*httputil.ReverseProxy
}

// AddTracing wraps a httputil.ReverseProxy's transporter with OpenTelemetry instrumentation
// This allows us to capture request and response details in our telemetry data
// Note: Ensure that the OpenTelemetry SDK and otelhttp package are properly initialized before using this client
func (r *ReverseProxy) AddTracing(ctxWithObserver context.Context) (fault error) {
	_, _, err := Get(ctxWithObserver)
	if err != nil {
		return fmt.Errorf("could not get go11y observer from context: %w", err)
	}
	r.Transport = otelhttp.NewTransport(r.Transport)
	return nil
}

// AddPropagation wraps a httputil.ReverseProxy's transporter with OpenTelemetry propagation
// This allows us to propagate tracing context across service boundaries
// Note: Ensure that the OpenTelemetry SDK and propagation package are properly initialized before using this client
func (r *ReverseProxy) AddPropagation(ctxWithObserver context.Context) (fault error) {
	_, _, err := Get(ctxWithObserver)
	if err != nil {
		return fmt.Errorf("could not get go11y observer from context: %w", err)
	}
	r.Transport = propagateRoundTripper(r.Transport)
	return nil
}

// AddLogging wraps a httputil.ReverseProxy's transporter with logging functionality
// This allows us to log request and response details for debugging and monitoring purposes
// Note: Ensure that the logging system is properly initialized before using this client
func (r *ReverseProxy) AddLogging(ctxWithObserver context.Context) (fault error) {
	_, _, err := Get(ctxWithObserver)
	if err != nil {
		return fmt.Errorf("could not get go11y observer from context: %w", err)
	}

	r.Transport = logRoundTripper(ctxWithObserver, r.Transport)

	return nil
}

// AddDBStore wraps a httputil.ReverseProxy's transporter with database storage functionality
// This allows us to store request and response details in a database for auditing and analysis purposes
// Note: Ensure that the database connection and storage system are properly initialized before using this client
func (r *ReverseProxy) AddDBStore(ctxWithObserver context.Context, dbStorer DBStorer) (fault error) {
	_, _, err := Get(ctxWithObserver)
	if err != nil {
		return fmt.Errorf("could not get go11y observer from context: %w", err)
	}

	r.Transport = dbStoreRoundTripper(ctxWithObserver, dbStorer, r.Transport)
	return nil
}
