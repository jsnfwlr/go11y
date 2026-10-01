package go11y

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// HTTPClient is a wrapper around http.Client that provides methods
// to add OpenTelemetry tracing, propagation, logging, metrics, and database storage functionality.
type HTTPClient struct {
	*http.Client
}

// AddTracing wraps a http.Client's transporter with OpenTelemetry instrumentation
// This allows us to capture request and response details in our telemetry data
// Note: Ensure that the OpenTelemetry SDK and otelhttp package are properly initialized before using this client
func (c *HTTPClient) AddTracing(ctxWithObserver context.Context) (fault error) {
	_, _, err := Get(ctxWithObserver)
	if err != nil {
		return fmt.Errorf("could not get go11y observer from context: %w", err)
	}
	c.Transport = otelhttp.NewTransport(c.Transport)
	return nil
}

// AddPropagation wraps a http.Client's transporter with OpenTelemetry propagation
// This allows us to propagate tracing context across service boundaries
// Note: Ensure that the OpenTelemetry SDK and propagation package are properly initialized before using this client
func (c *HTTPClient) AddPropagation(ctxWithObserver context.Context) (fault error) {
	_, _, err := Get(ctxWithObserver)
	if err != nil {
		return fmt.Errorf("could not get go11y observer from context: %w", err)
	}

	c.Transport = propagateRoundTripper(c.Transport)
	return nil
}

// AddLogging wraps a http.Client's transporter with logging functionality
// This allows us to log request and response details for debugging and monitoring purposes
// Note: Ensure that the logging system is properly initialized before using this client
func (c *HTTPClient) AddLogging(ctxWithObserver context.Context) (fault error) {
	_, _, err := Get(ctxWithObserver)
	if err != nil {
		return fmt.Errorf("could not get go11y observer from context: %w", err)
	}

	c.Transport = logRoundTripper(ctxWithObserver, c.Transport)
	return nil
}

// AddDBStore wraps a http.Client's transporter with database storage functionality
// This allows us to store request and response details in a database for auditing and analysis purposes
// Note: Ensure that the database connection and storage system are properly initialized before using this client
func (c *HTTPClient) AddDBStore(ctxWithObserver context.Context, dbStorer DBStorer) (fault error) {
	_, _, err := Get(ctxWithObserver)
	if err != nil {
		return fmt.Errorf("could not get go11y observer from context: %w", err)
	}

	c.Transport = dbStoreRoundTripper(ctxWithObserver, dbStorer, c.Transport)

	return nil
}

// MetricsRecorder is a function type for recording metrics.
type MetricsRecorder func(statusCode int, method, path string, startTime time.Time)

// AddMetrics wraps a http.Client's transporter with metrics recording functionality
// $recorder is the function that actually records the metrics - if it is nil an error is returned
// This allows us to record metrics for request and response details for monitoring purposes
func (c *HTTPClient) AddMetrics(recorder MetricsRecorder, pathMaskFunc PathMask) (fault error) {
	if recorder == nil {
		return errors.New("recorder cannot be nil")
	}

	c.Transport = metricsRoundTripper(c.Transport, recorder, pathMaskFunc)

	return nil
}

// AddRequestID wraps a http.Client's transporter with request ID tracking and propagation functionality
// The serviceName parameter is used to set the User-Agent header for the request to the service making the request
func (c *HTTPClient) AddRequestID(serviceName string) (fault error) {
	c.Transport = requestIDRoundTripper(c.Transport, serviceName)

	return nil
}
