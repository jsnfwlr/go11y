package go11y_test

import (
	"bytes"
	"context"
	"errors"
	"regexp"
	"testing"
	"uuid"

	"github.com/jsnfwlr/go11y"
)

func TestLoggingContext(t *testing.T) {
	t.Setenv("ENV", "test")
	t.Setenv("LOG_LEVEL", "develop")

	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)

	cfg, err := go11y.LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	ctx, o, err := go11y.Initialise(context.Background(), cfg, bufOut, bufErr)
	if err != nil {
		t.Fatalf("failed to initialise observer: %v", err)
	}
	defer func() {
		o.Close()
	}()

	o.Error("Test Logging Context", errors.New("TestLoggingContext"), go11y.SeverityHighest, "fatal", 1)
	ctx, o, _ = go11y.Extend(ctx, nil, "", go11y.FieldRequestID, uuid.New())
	o.Info("TestLoggingContext", nil, "info", 1)
	ctx = addFieldsToLoggerInContext(t, ctx, go11y.FieldRequestMethod, "GET", go11y.FieldRequestPath, "/api/v1/test")
	_, o, _ = go11y.Get(ctx)
	o.Info("TestLoggingContext", nil, "info", 2)

	// @TODO: read the buffer and check the output matches expected log format
	// and content
}

func addFieldsToLoggerInContext(t *testing.T, ctx context.Context, args ...any) (modCtx context.Context) {
	t.Helper()

	// Add fields to the logger in the context
	c, o, _ := go11y.Extend(ctx, args...)

	o.Info("AddFieldsToLoggerInContext", nil, "info", 1)

	return c
}

func TestDeduplication(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		input     []any
		expected  []any
		dupedKeys []string
	}{
		{
			name: "no duplicates",
			input: []any{
				"key1", "value1",
				"key2", "value2",
			},
			expected: []any{
				"key1", "value1",
				"key2", "value2",
			},
			dupedKeys: []string{},
		},
		{
			name: "just identical duplicates",
			input: []any{
				"key1", "value1",
				"key1", "value1",
			},
			expected: []any{
				"key1", "value1",
			},
			dupedKeys: []string{
				"key1",
			},
		},
		{
			name: "just duplicate keys with different values",
			input: []any{
				"key1", "value1",
				"key1", "value2",
			},
			expected: []any{
				"key1", "value1",
			},
			dupedKeys: []string{
				"key1",
			},
		},
		{
			name: "unique arg-pairs plus duplicate keys with different values",
			input: []any{
				"key1", "value1",
				"key2", "value2",
				"key1", "value2",
			},
			expected: []any{
				"key1", "value1",
				"key2", "value2",
			},
			dupedKeys: []string{
				"key1",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := go11y.DeduplicateArgs(tc.input)
			if len(result) != len(tc.expected) {
				t.Fatalf("expected length %d, got %d", len(tc.expected), len(result))
			}

			for i := range result {
				if result[i] != tc.expected[i] {
					t.Errorf("at index %d, expected %v, got %v", i, tc.expected[i], result[i])
				}
			}
		})
	}
}

func TestLevels(t *testing.T) {
	t.Setenv("ENV", "test")
	t.Setenv("LOG_LEVEL", "develop")

	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)

	cfg, err := go11y.LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	_, o, err := go11y.Initialise(context.Background(), cfg, bufOut, bufErr)
	if err != nil {
		t.Fatalf("failed to initialise observer: %v", err)
	}
	defer func() {
		o.Close()
	}()

	bufOut.Reset()

	o.Develop("message at develop level")
	o.Debug("message at debug level")
	o.Info("message at info level")
	o.Notice("message at notice level")
	o.Warning("message at warning level")
	o.Error("message at error level", errors.New("an error occurred"), go11y.SeverityHigh)

	expectedLogs := []string{
		`{"level":"DEVELOP","source":{"function":"github.com/jsnfwlr/go11y_test.TestLevels","file":"/logging_test.go","line":150},"msg":"message at develop level"}`,
		`{"level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y_test.TestLevels","file":"/logging_test.go","line":151},"msg":"message at debug level"}`,
		`{"level":"INFO","source":{"function":"github.com/jsnfwlr/go11y_test.TestLevels","file":"/logging_test.go","line":152},"msg":"message at info level"}`,
		`{"level":"NOTICE","source":{"function":"github.com/jsnfwlr/go11y_test.TestLevels","file":"/logging_test.go","line":153},"msg":"message at notice level"}`,
		`{"level":"WARN","source":{"function":"github.com/jsnfwlr/go11y_test.TestLevels","file":"/logging_test.go","line":154},"msg":"message at warning level"}`,
	}

	compareLogs(t, bufOut, expectedLogs, "TestLevelsLog", "logging_out.tmp", []*regexp.Regexp{})

	expectedErr := []string{
		`{"level":"ERROR","source":{"function":"github.com/jsnfwlr/go11y_test.TestLevels","file":"/logging_test.go","line":155},"msg":"message at error level","error":"an error occurred","severity":"high"}`,
	}

	compareLogs(t, bufErr, expectedErr, "TestLevelsErr", "logging_err.tmp", []*regexp.Regexp{})
}
