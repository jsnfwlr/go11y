package go11y_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/jsnfwlr/go11y"
	"github.com/jsnfwlr/go11y/config"
)

func TestLoggingContext(t *testing.T) {
	t.Setenv("ENV", "test")
	t.Setenv("LOG_LEVEL", "develop")

	buf := new(bytes.Buffer)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	ctx, o, err := go11y.Initialise(context.Background(), cfg, buf)
	if err != nil {
		t.Fatalf("failed to initialise observer: %v", err)
	}
	defer func() {
		o.Close()
	}()

	o.Fatal(errors.New("TestLoggingContext"), nil, "fatal", 1)
	ctx, o = go11y.Extend(ctx, nil, "", go11y.FieldRequestID, uuid.New())
	o.Info("TestLoggingContext", nil, "info", 1)
	ctx = AddFieldsToLoggerInContext(t, ctx, go11y.FieldRequestMethod, "GET", go11y.FieldRequestPath, "/api/v1/test")
	o = go11y.Get(ctx)
	o.Info("TestLoggingContext", nil, "info", 2)

	// @TODO: read the buffer and check the output matches expected log format
	// and content
}

func AddFieldsToLoggerInContext(t *testing.T, ctx context.Context, args ...any) (modCtx context.Context) {
	// Add fields to the logger in the context
	c, o := go11y.Extend(ctx, args...)

	o.Info("AddFieldsToLoggerInContext", nil, "info", 1)

	return c
}
