package go11y

import (
	"context"
	"fmt"
	"io"
	"log/slog"
)

// InitialiseTestLogger set up a logger for use in tests - no tracing, no db logging
func InitialiseTestLogger(ctx context.Context, level slog.Level, logOut, logErr io.Writer) (ctxWithObserver context.Context, observer *Observer, fault error) {
	cfg := CreateConfig(level, "", "", "", []string{}, []string{})

	ctx, o, err := Initialise(ctx, cfg, logOut, logErr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialise observer: %w", err)
	}

	return ctx, o, nil
}

// InitialiseTestTracer set up a tracer for use in tests - with tracing, but no db logging
func InitialiseTestTracer(ctx context.Context, level slog.Level, logOut, logErr io.Writer, otelURL, serviceName string) (ctxWithObserver context.Context, observer *Observer, fault error) {
	cfg := CreateConfig(level, otelURL, "", serviceName, []string{}, []string{})

	ctx, o, err := Initialise(ctx, cfg, logOut, logErr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialise observer: %w", err)
	}

	return ctx, o, nil
}
