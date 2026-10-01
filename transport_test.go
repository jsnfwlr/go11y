package go11y_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jsnfwlr/go11y"
	"github.com/jsnfwlr/go11y/storer"
	testingContainers "github.com/jsnfwlr/go11y/tests/containers"
	"github.com/jsnfwlr/go11y/tests/db"
	"github.com/jsnfwlr/go11y/tests/etc/migrations"

	"github.com/jackc/pgx/v5/pgtype"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestLoggingTransport(t *testing.T) {
	client := &go11y.HTTPClient{
		&http.Client{
			Transport: http.DefaultTransport,
		},
	}

	ctx := context.Background()

	t.Setenv("ENV", "test")
	t.Setenv("LOG_LEVEL", "develop")

	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)

	cfg, err := go11y.LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	ctx, o, err := go11y.Initialise(ctx, cfg, bufOut, bufErr)
	if err != nil {
		t.Fatalf("failed to initialise observer: %v", err)
	}

	err = client.AddLogging(ctx)
	if err != nil {
		t.Fatalf("failed to add logging to HTTP client: %v", err)
	}

	defer func() {
		o.Close()
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://ipapi.co/1.1.1.1/json/", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to execute request: %v", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	for {
		l, err := bufOut.ReadString('\n') // Read the first line to ensure logging output is flushed
		if err != nil {
			if err.Error() == "EOF" {
				break // End of file reached, exit the loop
			}
			t.Fatalf("failed to read log output: %v", err)
		}
		if l == "" {
			continue // Skip empty lines
		}
		t.Logf("Log output: %s", l)
	}
}

func TestStoringTransport(t *testing.T) {
	t.Setenv("ENV", "test")
	t.Setenv("LOG_LEVEL", "develop")

	ctx := context.Background()
	ctr, err := testingContainers.Postgres(t, ctx, "17")
	if err != nil {
		t.Fatalf("failed to start Postgres container: %v", err)
	}
	defer testcontainers.CleanupContainer(t, ctr.Postgres)

	defer func() {
		if err := testcontainers.TerminateContainer(ctr.Postgres); err != nil {
			t.Fatalf("failed to terminate Postgres container: %v", err)
		}
	}()

	t.Setenv("DATABASE_URL", ctr.DatabaseURL())

	cfg, err := go11y.LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	ctx, o, err := go11y.Initialise(ctx, cfg, nil, nil)
	if err != nil {
		t.Fatalf("failed to initialise observer: %v", err)
	}
	defer func() {
		o.Close()
	}()

	client := &go11y.HTTPClient{
		&http.Client{
			Transport: http.DefaultTransport,
		},
	}

	migFS, err := migrations.New()
	if err != nil {
		t.Fatalf("failed to create migrations: %v", err)
	}

	migrator, err := db.NewMigrator(ctx, o, ctr, migFS)
	if err != nil {
		t.Fatalf("failed to create migrator: %v", err)
	}

	err = migrator.Migrate()
	if err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	dbStorer, err := storer.New(ctx, ctr.DatabaseURL())
	if err != nil {
		t.Fatalf("failed to create DB storer: %v", err)
	}

	err = client.AddDBStore(ctx, dbStorer)
	if err != nil {
		t.Fatalf("failed to add logging to HTTP client: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://ipapi.co/1.1.1.1/json/", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "test-request-id-123")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to execute request: %v", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()
}

func TestRoundTripperFunc(t *testing.T) {
	t.Parallel()

	t.Run("nil round tripper falls back to the default transport", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTeapot)
		}))
		defer srv.Close()

		var rt go11y.RoundTripperFunc

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL, nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		resp, err := rt.RoundTrip(req)
		if err != nil {
			t.Fatalf("failed to execute request: %v", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusTeapot {
			t.Errorf("expected status %d, got %d", http.StatusTeapot, resp.StatusCode)
		}
	})

	t.Run("non-nil round tripper calls the wrapped function", func(t *testing.T) {
		called := 0
		rt := go11y.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			called++
			return newTestResponse(http.StatusOK, "pong", nil), nil
		})

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.com/ping", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		resp, err := rt.RoundTrip(req)
		if err != nil {
			t.Fatalf("failed to execute request: %v", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if called != 1 {
			t.Errorf("expected the wrapped function to be called once, got %d calls", called)
		}
	})

	t.Run("non-nil round tripper returns the wrapped function's error", func(t *testing.T) {
		wantErr := errors.New("boom")
		rt := go11y.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return nil, wantErr
		})

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.com/ping", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		//nolint:bodyclose // the round tripper returns a nil response alongside the error
		_, err = rt.RoundTrip(req)
		if !errors.Is(err, wantErr) {
			t.Errorf("expected error %v, got %v", wantErr, err)
		}
	})
}

func TestLoggingRoundTripper(t *testing.T) {
	t.Parallel()

	bufOut := new(bytes.Buffer)
	ctx, o := initGo11y(t, t.Context(), bufOut, bufOut)
	defer o.Close()

	testResp := newTestResponse(
		http.StatusOK,
		`{"token":"response-secret-token-value","status":"ok"}`,
		http.Header{"Set-Cookie": []string{"session=response-cookie-value"}},
	)

	defer func() {
		_ = testResp.Body.Close()
	}()

	next := &recordingRoundTripper{
		response: testResp,
	}

	client := &go11y.HTTPClient{Client: &http.Client{Transport: next}}

	if err := client.AddLogging(ctx); err != nil {
		t.Fatalf("failed to add logging to HTTP client: %v", err)
	}

	reqBody := `{"password":"request-secret-password","username":"tester"}`

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://example.com/v1/thing", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer request-secret-token-value")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to execute request: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if next.calls != 1 {
		t.Fatalf("expected the next round tripper to be called once, got %d calls", next.calls)
	}

	// the round tripper consumes both bodies to log them, so both must be readable downstream
	if string(next.receivedBody) != reqBody {
		t.Errorf("expected the next round tripper to receive body %q, got %q", reqBody, next.receivedBody)
	}

	gotRespBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	if !strings.Contains(string(gotRespBody), `"status":"ok"`) {
		t.Errorf("expected the caller to still be able to read the response body, got %q", gotRespBody)
	}

	requestEntry := findLogEntry(t, bufOut, "outbound call - request")

	if requestEntry[go11y.FieldRequestMethod] != http.MethodPost {
		t.Errorf("expected %s %q, got %v", go11y.FieldRequestMethod, http.MethodPost, requestEntry[go11y.FieldRequestMethod])
	}
	if requestEntry[go11y.FieldRequestURL] != "http://example.com/v1/thing" {
		t.Errorf("expected %s %q, got %v", go11y.FieldRequestURL, "http://example.com/v1/thing", requestEntry[go11y.FieldRequestURL])
	}

	loggedAuth := logEntryHeader(t, requestEntry, go11y.FieldRequestHeaders, "Authorization")
	if strings.Contains(loggedAuth, "request-secret-token-value") {
		t.Errorf("expected the Authorization header to be redacted, got %q", loggedAuth)
	}

	loggedReqBody := logEntryBody(t, requestEntry, go11y.FieldRequestBody)
	if strings.Contains(loggedReqBody, "request-secret-password") {
		t.Errorf("expected the request body password to be redacted, got %q", loggedReqBody)
	}
	if !strings.Contains(loggedReqBody, "tester") {
		t.Errorf("expected the request body to retain non-secret fields, got %q", loggedReqBody)
	}

	responseEntry := findLogEntry(t, bufOut, "outbound call - response")

	if responseEntry[go11y.FieldStatusCode] != float64(http.StatusOK) {
		t.Errorf("expected %s %d, got %v", go11y.FieldStatusCode, http.StatusOK, responseEntry[go11y.FieldStatusCode])
	}
	if _, ok := responseEntry[go11y.FieldCallDuration]; !ok {
		t.Errorf("expected the response log to include %s, got %v", go11y.FieldCallDuration, responseEntry)
	}

	loggedCookie := logEntryHeader(t, responseEntry, go11y.FieldResponseHeaders, "Set-Cookie")
	if strings.Contains(loggedCookie, "response-cookie-value") {
		t.Errorf("expected the Set-Cookie header to be redacted, got %q", loggedCookie)
	}

	loggedRespBody := logEntryBody(t, responseEntry, go11y.FieldResponseBody)
	if strings.Contains(loggedRespBody, "response-secret-token-value") {
		t.Errorf("expected the response body token to be redacted, got %q", loggedRespBody)
	}
}

func TestLoggingRoundTripperTransportError(t *testing.T) {
	t.Parallel()

	bufOut := new(bytes.Buffer)
	ctx, o := initGo11y(t, t.Context(), bufOut, bufOut)
	defer o.Close()

	wantErr := errors.New("connection refused")
	client := &go11y.HTTPClient{Client: &http.Client{Transport: &recordingRoundTripper{err: wantErr}}}

	if err := client.AddLogging(ctx); err != nil {
		t.Fatalf("failed to add logging to HTTP client: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com/v1/thing", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	//nolint:bodyclose // no response is returned when the transport fails
	_, err = client.Do(req)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}

	if strings.Contains(bufOut.String(), "outbound call - response") {
		t.Errorf("expected no response log when the transport fails, got %s", bufOut.String())
	}
	if !strings.Contains(bufOut.String(), "outbound call - request") {
		t.Errorf("expected the request to still be logged, got %s", bufOut.String())
	}
}

func TestAddTransportsWithoutObserver(t *testing.T) {
	t.Parallel()

	tests := map[string]func(c *go11y.HTTPClient, ctx context.Context) error{
		"AddLogging": func(c *go11y.HTTPClient, ctx context.Context) error {
			return c.AddLogging(ctx)
		},
		"AddPropagation": func(c *go11y.HTTPClient, ctx context.Context) error {
			return c.AddPropagation(ctx)
		},
		"AddDBStore": func(c *go11y.HTTPClient, ctx context.Context) error {
			return c.AddDBStore(ctx, &fakeDBStorer{})
		},
	}

	for name, add := range tests {
		t.Run(name, func(t *testing.T) {
			testResp := newTestResponse(http.StatusOK, "", nil)
			defer func() {
				_ = testResp.Body.Close()
			}()

			transport := &recordingRoundTripper{response: testResp}
			client := &go11y.HTTPClient{Client: &http.Client{Transport: transport}}

			err := add(client, context.Background())
			if err == nil {
				t.Fatalf("expected an error when the context has no go11y observer")
			}
			if !strings.Contains(err.Error(), "could not get go11y observer from context") {
				t.Errorf("unexpected error: %v", err)
			}

			if client.Transport != http.RoundTripper(transport) {
				t.Errorf("expected the transport to be left untouched when the observer is missing")
			}
		})
	}
}

// TestRoundTripperBodyReadErrors covers the round trippers that consume the request and response bodies in order to
// record them - a body that cannot be read must surface as an error rather than an incomplete record.
func TestRoundTripperBodyReadErrors(t *testing.T) {
	t.Parallel()

	adders := map[string]func(c *go11y.HTTPClient, ctx context.Context) error{
		"AddLogging": func(c *go11y.HTTPClient, ctx context.Context) error {
			return c.AddLogging(ctx)
		},
		"AddDBStore": func(c *go11y.HTTPClient, ctx context.Context) error {
			return c.AddDBStore(ctx, &fakeDBStorer{})
		},
	}

	testResp := newTestResponse(http.StatusOK, "", nil)
	defer func() {
		_ = testResp.Body.Close()
	}()

	bodies := map[string]struct {
		requestBody io.Reader
		response    *http.Response
		wantMessage string
	}{
		"request body": {
			requestBody: errReader{err: errors.New("request read failed")},
			response:    testResp,
			wantMessage: "failed to read request body",
		},
		"response body": {
			requestBody: nil,
			response: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{},
				Body:       io.NopCloser(errReader{err: errors.New("response read failed")}),
			},
			wantMessage: "failed to read response body",
		},
	}

	for adderName, add := range adders {
		for bodyName, tc := range bodies {
			t.Run(adderName+" "+bodyName, func(t *testing.T) {
				bufOut := new(bytes.Buffer)
				ctx, o := initGo11y(t, t.Context(), bufOut, bufOut)
				defer o.Close()

				client := &go11y.HTTPClient{
					Client: &http.Client{Transport: &recordingRoundTripper{response: tc.response}},
				}

				if err := add(client, ctx); err != nil {
					t.Fatalf("failed to wrap the HTTP client: %v", err)
				}

				req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://example.com/v1/thing", tc.requestBody)
				if err != nil {
					t.Fatalf("failed to create request: %v", err)
				}

				//nolint:bodyclose // no response is returned when the body cannot be read
				_, err = client.Do(req)
				if err == nil {
					t.Fatalf("expected an error when the %s cannot be read", bodyName)
				}
				if !strings.Contains(err.Error(), tc.wantMessage) {
					t.Errorf("expected an error containing %q, got %v", tc.wantMessage, err)
				}
			})
		}
	}
}

func TestDBStoreRoundTripper(t *testing.T) {
	t.Parallel()

	bufOut := new(bytes.Buffer)
	ctx, o := initGo11y(t, t.Context(), bufOut, bufOut)
	defer o.Close()

	testResp := newTestResponse(
		http.StatusOK,
		`{"secret":"response-secret-value","status":"ok"}`,
		http.Header{"Content-Type": []string{"application/json"}},
	)
	defer func() {
		_ = testResp.Body.Close()
	}()

	next := &recordingRoundTripper{
		response: testResp,
	}

	dbStorer := &fakeDBStorer{}

	client := &go11y.HTTPClient{Client: &http.Client{Transport: next}}

	if err := client.AddDBStore(ctx, dbStorer); err != nil {
		t.Fatalf("failed to add DB storage to HTTP client: %v", err)
	}

	reqBody := `{"password":"request-secret-password","username":"tester"}`

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://example.com/v1/thing", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer request-secret-token-value")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to execute request: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	gotRespBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	if !strings.Contains(string(gotRespBody), `"status":"ok"`) {
		t.Errorf("expected the caller to still be able to read the response body, got %q", gotRespBody)
	}

	if dbStorer.execCalls != 1 {
		t.Fatalf("expected Exec to be called once, got %d calls", dbStorer.execCalls)
	}

	if dbStorer.url != "http://example.com/v1/thing" {
		t.Errorf("expected URL %q, got %q", "http://example.com/v1/thing", dbStorer.url)
	}
	if dbStorer.method != http.MethodPost {
		t.Errorf("expected method %q, got %q", http.MethodPost, dbStorer.method)
	}
	if dbStorer.statusCode != int32(http.StatusOK) {
		t.Errorf("expected status code %d, got %d", http.StatusOK, dbStorer.statusCode)
	}
	if dbStorer.responseTimeMS < 0 {
		t.Errorf("expected a non-negative response time, got %d", dbStorer.responseTimeMS)
	}

	if !dbStorer.requestBody.Valid {
		t.Errorf("expected the stored request body to be valid")
	}
	if strings.Contains(dbStorer.requestBody.String, "request-secret-password") {
		t.Errorf("expected the stored request body password to be redacted, got %q", dbStorer.requestBody.String)
	}
	if !strings.Contains(dbStorer.requestBody.String, "tester") {
		t.Errorf("expected the stored request body to retain non-secret fields, got %q", dbStorer.requestBody.String)
	}

	if !dbStorer.responseBody.Valid {
		t.Errorf("expected the stored response body to be valid")
	}
	if strings.Contains(dbStorer.responseBody.String, "response-secret-value") {
		t.Errorf("expected the stored response body secret to be redacted, got %q", dbStorer.responseBody.String)
	}

	storedReqHeaders := http.Header{}
	if err := json.Unmarshal(dbStorer.requestHeaders, &storedReqHeaders); err != nil {
		t.Fatalf("failed to unmarshal the stored request headers: %v", err)
	}
	if strings.Contains(storedReqHeaders.Get("Authorization"), "request-secret-token-value") {
		t.Errorf("expected the stored Authorization header to be redacted, got %q", storedReqHeaders.Get("Authorization"))
	}

	storedRespHeaders := http.Header{}
	if err := json.Unmarshal(dbStorer.responseHeaders, &storedRespHeaders); err != nil {
		t.Fatalf("failed to unmarshal the stored response headers: %v", err)
	}
	if storedRespHeaders.Get("Content-Type") != "application/json" {
		t.Errorf("expected the stored response headers to include the content type, got %v", storedRespHeaders)
	}
}

func TestDBStoreRoundTripperErrors(t *testing.T) {
	t.Parallel()

	t.Run("returns the transport error without storing anything", func(t *testing.T) {
		bufOut := new(bytes.Buffer)
		ctx, o := initGo11y(t, t.Context(), bufOut, bufOut)
		defer o.Close()

		wantErr := errors.New("connection refused")
		dbStorer := &fakeDBStorer{}

		client := &go11y.HTTPClient{Client: &http.Client{Transport: &recordingRoundTripper{err: wantErr}}}
		if err := client.AddDBStore(ctx, dbStorer); err != nil {
			t.Fatalf("failed to add DB storage to HTTP client: %v", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com/v1/thing", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		//nolint:bodyclose // no response is returned when the transport fails
		_, err = client.Do(req)
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected error %v, got %v", wantErr, err)
		}

		if dbStorer.execCalls != 0 {
			t.Errorf("expected Exec not to be called, got %d calls", dbStorer.execCalls)
		}
	})

	t.Run("returns an error when the storer fails", func(t *testing.T) {
		bufOut := new(bytes.Buffer)
		bufErr := new(bytes.Buffer)
		ctx, o := initGo11y(t, t.Context(), bufOut, bufErr)
		defer o.Close()

		wantErr := errors.New("insert failed")
		dbStorer := &fakeDBStorer{execErr: wantErr}

		testResp := newTestResponse(http.StatusOK, `{"status":"ok"}`, nil)
		defer func() {
			_ = testResp.Body.Close()
		}()

		next := &recordingRoundTripper{response: testResp}

		client := &go11y.HTTPClient{Client: &http.Client{Transport: next}}
		if err := client.AddDBStore(ctx, dbStorer); err != nil {
			t.Fatalf("failed to add DB storage to HTTP client: %v", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com/v1/thing", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		//nolint:bodyclose // no response is returned when the storer fails
		_, err = client.Do(req)
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected error %v, got %v", wantErr, err)
		}

		if !strings.Contains(bufErr.String(), "failed to store request/response in database") {
			t.Errorf("expected the storer failure to be logged as an error, got %s", bufErr.String())
		}
	})
}

func TestPropagateRoundTripper(t *testing.T) {
	t.Parallel()

	previous := otel.GetTextMapPropagator()
	otel.SetTextMapPropagator(propagation.TraceContext{})
	defer otel.SetTextMapPropagator(previous)

	bufOut := new(bytes.Buffer)
	ctx, o := initGo11y(t, t.Context(), bufOut, bufOut)
	defer o.Close()

	traceID, err := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	if err != nil {
		t.Fatalf("failed to build a trace ID: %v", err)
	}
	spanID, err := trace.SpanIDFromHex("0102030405060708")
	if err != nil {
		t.Fatalf("failed to build a span ID: %v", err)
	}

	ctx = trace.ContextWithSpanContext(ctx, trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	}))

	testResp := newTestResponse(http.StatusOK, "", nil)
	defer func() {
		_ = testResp.Body.Close()
	}()

	next := &recordingRoundTripper{response: testResp}

	client := &go11y.HTTPClient{Client: &http.Client{Transport: next}}
	if err := client.AddPropagation(ctx); err != nil {
		t.Fatalf("failed to add propagation to HTTP client: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com/v1/thing", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to execute request: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	want := "00-0102030405060708090a0b0c0d0e0f10-0102030405060708-01"
	if got := next.receivedHeaders.Get("Traceparent"); got != want {
		t.Errorf("expected traceparent %q, got %q", want, got)
	}
}

func TestMetricsRoundTripper(t *testing.T) {
	t.Parallel()

	type recorded struct {
		statusCode int
		method     string
		path       string
		calls      int
	}

	t.Run("records the call details", func(t *testing.T) {
		got := recorded{}
		recorder := func(statusCode int, method, path string, startTime time.Time) {
			got = recorded{statusCode: statusCode, method: method, path: path, calls: got.calls + 1}
			if startTime.IsZero() {
				t.Errorf("expected a non-zero start time")
			}
		}

		testResp := newTestResponse(http.StatusNotFound, "", nil)
		defer func() {
			_ = testResp.Body.Close()
		}()

		next := &recordingRoundTripper{response: testResp}
		client := &go11y.HTTPClient{Client: &http.Client{Transport: next}}

		if err := client.AddMetrics(recorder, nil); err != nil {
			t.Fatalf("failed to add metrics to HTTP client: %v", err)
		}

		req, err := http.NewRequestWithContext(t.Context(), http.MethodPut, "http://example.com/v1/things/123", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("failed to execute request: %v", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if got.calls != 1 {
			t.Fatalf("expected the recorder to be called once, got %d calls", got.calls)
		}
		if got.statusCode != http.StatusNotFound {
			t.Errorf("expected status code %d, got %d", http.StatusNotFound, got.statusCode)
		}
		if got.method != http.MethodPut {
			t.Errorf("expected method %q, got %q", http.MethodPut, got.method)
		}
		if got.path != "/v1/things/123" {
			t.Errorf("expected an unmasked path when no mask is supplied, got %q", got.path)
		}
	})

	t.Run("applies the path mask", func(t *testing.T) {
		got := recorded{}
		recorder := func(statusCode int, method, path string, startTime time.Time) {
			got = recorded{statusCode: statusCode, method: method, path: path, calls: got.calls + 1}
		}

		mask := func(path string) string {
			return "/v1/things/{id}"
		}

		testResp := newTestResponse(http.StatusOK, "", nil)
		defer func() {
			_ = testResp.Body.Close()
		}()

		next := &recordingRoundTripper{response: testResp}
		client := &go11y.HTTPClient{Client: &http.Client{Transport: next}}

		if err := client.AddMetrics(recorder, mask); err != nil {
			t.Fatalf("failed to add metrics to HTTP client: %v", err)
		}

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.com/v1/things/123", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("failed to execute request: %v", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if got.path != "/v1/things/{id}" {
			t.Errorf("expected the masked path, got %q", got.path)
		}
	})

	t.Run("does not record when the transport fails", func(t *testing.T) {
		calls := 0
		recorder := func(statusCode int, method, path string, startTime time.Time) {
			calls++
		}

		wantErr := errors.New("connection refused")
		client := &go11y.HTTPClient{Client: &http.Client{Transport: &recordingRoundTripper{err: wantErr}}}

		if err := client.AddMetrics(recorder, nil); err != nil {
			t.Fatalf("failed to add metrics to HTTP client: %v", err)
		}

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.com/v1/thing", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		//nolint:bodyclose // no response is returned when the transport fails
		_, err = client.Do(req)
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected error %v, got %v", wantErr, err)
		}

		if calls != 0 {
			t.Errorf("expected the recorder not to be called, got %d calls", calls)
		}
	})

	t.Run("rejects a nil recorder", func(t *testing.T) {
		testResp := newTestResponse(http.StatusOK, "", nil)
		defer func() {
			_ = testResp.Body.Close()
		}()

		transport := &recordingRoundTripper{response: testResp}
		client := &go11y.HTTPClient{Client: &http.Client{Transport: transport}}

		if err := client.AddMetrics(nil, nil); err == nil {
			t.Fatalf("expected an error when the recorder is nil")
		}

		if client.Transport != http.RoundTripper(transport) {
			t.Errorf("expected the transport to be left untouched when the recorder is nil")
		}
	})
}

func TestRequestIDRoundTripper(t *testing.T) {
	t.Parallel()

	t.Run("sets the request ID header from the context", func(t *testing.T) {
		ctx := generateRequestIDForContext(t)

		want, err := go11y.GetContextRequestID(ctx)
		if err != nil {
			t.Fatalf("expected a generated request ID in the context: %v", err)
		}

		testResp := newTestResponse(http.StatusOK, "", nil)
		defer func() {
			_ = testResp.Body.Close()
		}()

		next := &recordingRoundTripper{response: testResp}
		client := &go11y.HTTPClient{Client: &http.Client{Transport: next}}

		if err := client.AddRequestID("test-service"); err != nil {
			t.Fatalf("failed to add the request ID to HTTP client: %v", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com/v1/thing", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("failed to execute request: %v", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if got := next.receivedHeaders.Get(go11y.RequestIDHeader); got != want.String() {
			t.Errorf("expected request ID %q, got %q", want, got)
		}
	})

	t.Run("keeps an existing request ID header", func(t *testing.T) {
		ctx := generateRequestIDForContext(t)

		testResp := newTestResponse(http.StatusOK, "", nil)
		defer func() {
			_ = testResp.Body.Close()
		}()

		next := &recordingRoundTripper{response: testResp}
		client := &go11y.HTTPClient{Client: &http.Client{Transport: next}}

		if err := client.AddRequestID("test-service"); err != nil {
			t.Fatalf("failed to add the request ID to HTTP client: %v", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com/v1/thing", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}
		req.Header.Set(go11y.RequestIDHeader, "already-set")

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("failed to execute request: %v", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if got := next.receivedHeaders.Get(go11y.RequestIDHeader); got != "already-set" {
			t.Errorf("expected the existing request ID to be kept, got %q", got)
		}
	})

	t.Run("generates a new request ID when the header is unset and the context has no request ID", func(t *testing.T) {
		testResp := newTestResponse(http.StatusOK, "", nil)
		defer func() {
			_ = testResp.Body.Close()
		}()

		next := &recordingRoundTripper{response: testResp}
		client := &go11y.HTTPClient{Client: &http.Client{Transport: next}}

		if err := client.AddRequestID("test-service"); err != nil {
			t.Fatalf("failed to add the request ID to HTTP client: %v", err)
		}

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.com/v1/thing", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("failed to execute request: %v", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if got := next.receivedHeaders.Get(go11y.RequestIDHeader); got == "" {
			t.Errorf("expected a new request ID to be generated, got empty header")
		}
	})
}

// recordingRoundTripper is the last round tripper in a chain: it records what it was handed and returns a canned
// response or error.
type recordingRoundTripper struct {
	response        *http.Response
	err             error
	calls           int
	receivedHeaders http.Header
	receivedBody    []byte
}

func (f *recordingRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	f.calls++
	f.receivedHeaders = r.Header.Clone()

	if r.Body != nil {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		f.receivedBody = body
	}

	if f.err != nil {
		return nil, f.err
	}

	return f.response, nil
}

// errReader is an io.Reader that always fails, used to exercise the body reading error paths.
type errReader struct {
	err error
}

func (e errReader) Read(p []byte) (int, error) {
	return 0, e.err
}

// fakeDBStorer captures the values handed to it so tests can assert on what would have been persisted.
type fakeDBStorer struct {
	url             string
	method          string
	requestHeaders  []byte
	requestBody     pgtype.Text
	responseTimeMS  int64
	responseHeaders []byte
	responseBody    pgtype.Text
	statusCode      int32
	execCalls       int
	execErr         error
}

func (f *fakeDBStorer) SetURL(url string)                 { f.url = url }
func (f *fakeDBStorer) SetMethod(method string)           { f.method = method }
func (f *fakeDBStorer) SetRequestHeaders(headers []byte)  { f.requestHeaders = headers }
func (f *fakeDBStorer) SetRequestBody(body pgtype.Text)   { f.requestBody = body }
func (f *fakeDBStorer) SetResponseTimeMS(ms int64)        { f.responseTimeMS = ms }
func (f *fakeDBStorer) SetResponseHeaders(headers []byte) { f.responseHeaders = headers }
func (f *fakeDBStorer) SetResponseBody(body pgtype.Text)  { f.responseBody = body }
func (f *fakeDBStorer) SetStatusCode(statusCode int32)    { f.statusCode = statusCode }

func (f *fakeDBStorer) Exec(ctx context.Context) error {
	f.execCalls++
	return f.execErr
}

func newTestResponse(statusCode int, body string, header http.Header) *http.Response {
	if header == nil {
		header = http.Header{}
	}

	return &http.Response{
		StatusCode: statusCode,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// findLogEntry returns the first JSON log line in the buffer whose message matches msg.
func findLogEntry(t *testing.T, logs *bytes.Buffer, msg string) map[string]any {
	t.Helper()

	for line := range strings.SplitSeq(logs.String(), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}

		entry := map[string]any{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("failed to unmarshal log line %q: %v", line, err)
		}

		if entry["msg"] == msg {
			return entry
		}
	}

	t.Fatalf("could not find a log entry with the message %q in: %s", msg, logs.String())

	return nil
}

// logEntryHeader pulls a single header value out of a logged header field.
func logEntryHeader(t *testing.T, entry map[string]any, field, headerName string) string {
	t.Helper()

	headers, ok := entry[field].(map[string]any)
	if !ok {
		t.Fatalf("expected %s to be an object, got %v", field, entry[field])
	}

	values, ok := headers[headerName].([]any)
	if !ok || len(values) == 0 {
		t.Fatalf("expected %s to contain the %s header, got %v", field, headerName, headers)
	}

	value, ok := values[0].(string)
	if !ok {
		t.Fatalf("expected the %s header to be a string, got %v", headerName, values[0])
	}

	return value
}

// logEntryBody decodes a logged body field, which slog writes out as a base64 encoded byte slice.
func logEntryBody(t *testing.T, entry map[string]any, field string) string {
	t.Helper()

	encoded, ok := entry[field].(string)
	if !ok {
		t.Fatalf("expected %s to be a string, got %v", field, entry[field])
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("failed to decode %s %q: %v", field, encoded, err)
	}

	return string(decoded)
}

func TestPropagatingTransport(t *testing.T) {
	t.Skipf("Skipping test as it is flaky in CI/CD pipelines")
	t.Parallel()

	t.Setenv("ENV", "test")
	t.Setenv("LOG_LEVEL", "develop")

	ctx := context.Background()
	ctr, err := testingContainers.LGTM(t, ctx)
	if err != nil {
		t.Fatalf("failed to start Grafana LGTM container: %v", err)
	}
	testcontainers.CleanupContainer(t, ctr)

	defer func() {
		if err := testcontainers.TerminateContainer(ctr); err != nil {
			t.Fatalf("failed to terminate Grafana LGTM container: %v", err)
		}
	}()

	time.Sleep(60 * time.Second)

	cfg, err := go11y.LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	ctx, o, err := go11y.Initialise(ctx, cfg, nil, nil)
	if err != nil {
		t.Fatalf("failed to initialise observer: %v", err)
	}
	defer func() {
		o.Close()
	}()

	client := &go11y.HTTPClient{
		&http.Client{
			Transport: http.DefaultTransport,
		},
	}

	err = client.AddPropagation(ctx)
	if err != nil {
		t.Fatalf("failed to add tracing to HTTP client: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://ipapi.co/1.1.1.1/json/", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to execute request: %v", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()
}

func generateRequestIDForContext(t *testing.T) context.Context {
	t.Helper()
	return context.WithValue(t.Context(), go11y.RequestIDInstance, uuid.New().String())
}
