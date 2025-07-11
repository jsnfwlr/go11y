package o11y

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// AddTracingToHTTPClient wraps a HTTP client's transporter with OpenTelemetry instrumentation
// If the provided httpClient is nil, it creates a new http.Client with a default
// timeout of 2 minutes and a transport that is instrumented with OpenTelemetry
// to
// This allows us to capture request and response details in our telemetry data
//
// Note: Ensure that the OpenTelemetry SDK and otelhttp package are properly initialized before using this client
func AddTracingToHTTPClient(httpClient *http.Client) (fault error) {
	if httpClient == nil {
		return errors.New("httpClient cannot be nil")
	}

	// Wrap the existing transport with OpenTelemetry tracing
	httpClient.Transport = otelhttp.NewTransport(httpClient.Transport)
	return nil
}

func AddLoggingToHTTPClient(httpClient *http.Client) (fault error) {
	if httpClient == nil {
		return errors.New("httpClient cannot be nil")
	}

	// Wrap the existing transport with logging
	httpClient.Transport = LogRoundTripper(httpClient.Transport)
	return nil
}

func AddDBStoreToHTTPClient(httpClient *http.Client) (fault error) {
	if httpClient == nil {
		return errors.New("httpClient cannot be nil")
	}

	// Wrap the existing transport with logging
	httpClient.Transport = DBStoreRoundTripper(httpClient.Transport)
	return nil
}

type RoundTripperFunc func(*http.Request) (*http.Response, error)

func (rt RoundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	if rt == nil {
		return http.DefaultTransport.RoundTrip(r)
	}
	return rt(r)
}

func LogRoundTripper(next http.RoundTripper) http.RoundTripper {
	return RoundTripperFunc(func(r *http.Request) (w *http.Response, fault error) {
		ctx := r.Context()
		ctx, o := Get(ctx, nil)

		reqBody := []byte{}
		if r.Body != nil {
			defer func() {
				if err := r.Body.Close(); err != nil {
					o.Error(err)
				}
			}()
			var err error
			reqBody, err = io.ReadAll(r.Body)
			if err != nil {
				o.Error(err)
				return nil, fmt.Errorf("failed to read request body: %w", err)
			}
			// Create a new request with the read body
			r.Body = io.NopCloser(bytes.NewBuffer(reqBody)) // Use NopCloser to allow reading the body again if needed
		}

		requestArgs := []any{
			FieldRequestHeaders, RedactHeaders(r.Header),
			FieldRequestMethod, r.Method,
			FieldRequestURL, r.URL.String(),
			FieldRequestBody, reqBody,
		}

		o.log(ctx, 8, LevelInfo, "outbound call - request", requestArgs...)
		start := time.Now()

		// Send the actual request
		resp, err := next.RoundTrip(r)
		duration := time.Since(start)
		if err != nil {
			o.Error(err, FieldCallDuration, duration)
			return nil, err
		}

		respBody := []byte{}
		// read the response body, use it to log the response body, then build a new response to return
		if resp.Body != nil {
			defer func() {
				if err = resp.Body.Close(); err != nil {
					o.Error(err)
				}
			}()

			respBody, err = io.ReadAll(resp.Body)
			if err != nil {
				o.Error(err)
				return nil, fmt.Errorf("failed to read response body: %w", err)
			}
			// Create a new response with the read body
			resp.Body = io.NopCloser(bytes.NewBuffer(respBody)) // Use NopCloser to allow reading the body again if needed
		}

		responseArgs := []any{
			FieldCallDuration, duration,
			FieldStatusCode, resp.StatusCode,
			FieldResponseHeaders, RedactHeaders(resp.Header),
			FieldResponseBody, string(respBody),
		}
		o.log(ctx, 8, LevelInfo, "outbound call - response", responseArgs...)
		return resp, nil
	})
}

func DBStoreRoundTripper(next http.RoundTripper) http.RoundTripper {
	return RoundTripperFunc(func(r *http.Request) (w *http.Response, fault error) {
		ctx := r.Context()
		ctx, o := Get(ctx, nil)

		reqBody := []byte{}
		if r.Body != nil {
			defer func() {
				if err := r.Body.Close(); err != nil {
					o.Error(err)
				}
			}()
			var err error
			reqBody, err = io.ReadAll(r.Body)
			if err != nil {
				o.Error(err)
				return nil, fmt.Errorf("failed to read request body: %w", err)
			}
			// Create a new request with the read body
			r.Body = io.NopCloser(bytes.NewBuffer(reqBody)) // Use NopCloser to allow reading the body again if needed
		}

		start := time.Now()

		resp, err := next.RoundTrip(r)
		duration := time.Since(start)
		if err != nil {
			o.Error(err, FieldCallDuration, duration)
			return nil, err
		}

		respBody := []byte{}
		// read the response body, use it to log the response body, then build a new response to return
		if resp.Body != nil {
			defer func() {
				if err = resp.Body.Close(); err != nil {
					o.Error(err)
				}
			}()

			respBody, err = io.ReadAll(resp.Body)
			if err != nil {
				o.Error(err)
				return nil, fmt.Errorf("failed to read response body: %w", err)
			}
			// Create a new response with the read body
			resp.Body = io.NopCloser(bytes.NewBuffer(respBody)) // Use NopCloser to allow reading the body again if needed
		}

		err = o.store(ctx, r.URL.String(), r.Method, int32(resp.StatusCode), duration, reqBody, respBody, r.Header, resp.Header)
		if err != nil {
			o.Error(err, "failed to store API request")
			return nil, fmt.Errorf("failed to store API request: %w", err)
		}

		return resp, nil
	})
}

func RedactHeaders(headers http.Header) http.Header {
	redactedHeaders := make(http.Header)
	for key, values := range headers {
		if key == "Authorization" || key == "Cookie" {
			redactedHeaders[key] = []string{"REDACTED"}
		} else {
			redactedHeaders[key] = values
		}
	}
	return redactedHeaders
}
