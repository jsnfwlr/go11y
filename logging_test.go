package o11y_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/jsnfwlr/o11y"
)

func TestLoggingContext(t *testing.T) {
	t.Setenv("ENV", "test")
	t.Setenv("LOG_LEVEL", "develop")

	buf := new(bytes.Buffer)

	cfg, err := o11y.LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	ctx, o, err := o11y.Initialise(context.Background(), cfg, buf)
	if err != nil {
		t.Fatalf("failed to initialise observer: %v", err)
	}
	defer func() {
		o.Close()
	}()

	o.Fatal(errors.New("TestLoggingContext"), "fatal", 1)
	ctx, o = o11y.Extend(ctx, nil, "", o11y.FieldRequestID, uuid.New())
	o.Info("TestLoggingContext", "info", 1)
	ctx = AddFieldsToLoggerInContext(t, ctx, o11y.FieldRequestMethod, "GET", o11y.FieldRequestPath, "/api/v1/test")
	_, o = o11y.Get(ctx, nil)
	o.Info("TestLoggingContext", "info", 2)

	// @TODO: read the buffer and check the output matches expected log format
	// and content
}

func AddFieldsToLoggerInContext(t *testing.T, ctx context.Context, args ...any) (modCtx context.Context) {
	// Add fields to the logger in the context
	c, o := o11y.Extend(ctx, args...)

	o.Info("AddFieldsToLoggerInContext", "info", 1)

	return c
}
