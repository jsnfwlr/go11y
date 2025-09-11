package go11y_test

import (
	"bytes"
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/jsnfwlr/go11y"
	"github.com/jsnfwlr/go11y/config"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
)

func TestRoundtripLogger(t *testing.T) {
	client := &http.Client{
		Transport: http.DefaultTransport,
	}

	err := go11y.AddLoggingToHTTPClient(client)
	if err != nil {
		t.Fatalf("failed to add logging to HTTP client: %v", err)
	}

	ctx := context.Background()

	t.Setenv("ENV", "test")
	t.Setenv("LOG_LEVEL", "develop")

	buf := new(bytes.Buffer)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	ctx, o, err := go11y.Initialise(ctx, cfg, buf)
	if err != nil {
		t.Fatalf("failed to initialise observer: %v", err)
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
		pos := 0
		l, err := buf.ReadString('\n') // Read the first line to ensure logging output is flushed
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
		pos++
	}
}

func TestRoundtripStorer(t *testing.T) {
	t.Setenv("ENV", "test")
	t.Setenv("LOG_LEVEL", "develop")

	ctx := context.Background()
	ctr, err := go11y.Postgres(t, ctx)
	if err != nil {
		t.Fatalf("failed to start Postgres container: %v", err)
	}
	defer testcontainers.CleanupContainer(t, ctr)

	defer func() {
		if err := testcontainers.TerminateContainer(ctr); err != nil {
			t.Fatalf("failed to terminate Postgres container: %v", err)
		}
	}()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	_, o, err := go11y.Initialise(ctx, cfg, nil)
	if err != nil {
		t.Fatalf("failed to initialise observer: %v", err)
	}
	defer func() {
		o.Close()
	}()

	client := &http.Client{
		Transport: http.DefaultTransport,
	}

	err = go11y.AddDBStoreToHTTPClient(client)
	if err != nil {
		t.Fatalf("failed to add logging to HTTP client: %v", err)
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

	record, err := o.CheckStore()
	if err != nil {
		t.Fatalf("failed to check store: %v", err)
	}

	if record.Url != "https://ipapi.co/1.1.1.1/json/" {
		t.Fatalf("expected a url in the record to be '%s' but got '%s'", "https://ipapi.co/1.1.1.1/json/", record.Url)
	}
}

func TestRoundtripperTracer(t *testing.T) {
	t.Setenv("ENV", "test")
	t.Setenv("LOG_LEVEL", "develop")

	ctx := context.Background()
	ctr, err := go11y.LGTM(t, ctx)
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

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	_, o, err := go11y.Initialise(ctx, cfg, nil)
	if err != nil {
		t.Fatalf("failed to initialise observer: %v", err)
	}
	defer func() {
		o.Close()
	}()

	client := &http.Client{
		Transport: http.DefaultTransport,
	}

	err = go11y.AddTracingToHTTPClient(client)
	if err != nil {
		t.Fatalf("failed to add tracing to HTTP client: %v", err)
	}
}
