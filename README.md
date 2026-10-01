# go11y<sup>1</sup> for observability in Go

Opinionated but simple Go implementation of structured logging and open telemetry
tracing for your application.

## Features

### Structured Logging

go11y wraps the Go standard lib slog package, so it's structured logging with JSON from the outset, just more convenient.

```go
ctx := context.Background()
cfg := go11y.CreateConfig(go11y.Info, "", "postgres://user:pass@localhost:5234/database_name", "serviceName". []string{}, []string{})

_, o, _ := go11y.Initialise(ctx, cfg, os.Stdout, os.Strerr, "arg1", "val1")
o.Info("structured logging", "arg2", "val2")
```
```json
{
    "time":"2025-08-04T10:14:19.780509481+08:00",
    "level":"INFO",
    "source":{
        "function":"main.main",
        "file":"/home/user/demo/main.go",
        "line":81
    },
    "msg":"structured logging",
    "arg1": "val1",
    "arg2": "val2",
}
```

### Tracing

go11y has been expanded to cover integrated tracing by calling the Span() function and passing in a context that already contains go11y's observer.


```go
ctx := context.Background()
cfg := go11y.CreateConfig(go11y.Info, "http://otelcollector:8080", "postgres://user:pass@localhost:5234/database_name", "serviceName". []string{}, []string{})

ctx, _, _ = go11y.Initialise(ctx, cfg, os.Stdout, os.Stderr)
tracer := otel.Tracer("packageName")
_, o, _ = go11y.Span(ctx, tracer, "functionName", trace.WithSpanKind(trace.SpanKindClient))

o.Info("Tracing and Logging in one")
```

### Roundtrippers

### Middleware

## Configuration

### Hard Coded - BYO or Built in

### Environment Variables

## Examples

<!--
* WIP [Just Logging](./logging_example_test.go#L3)
* WIP [Logging and Tracing](./logging_example_test.go#L13)
* WIP [Middleware - SetRequestID()](./middleware_example_test.go#L3)
* WIP [Middleware - GetRequestID()](./middleware_example_test.go#L8)
* WIP [Middleware - LogRequest()](./middleware_example_test.go#L13)
* WIP [Logging Round Tripper](./roundtripper_example_test.go#L3)
* WIP [Tracing Round Tripper](./roundtripper_example_test.go#L8)
* WIP [DB Storing Round Tripper](./roundtripper_example_test.go#L13)
-->

## Used by

* [Kiss My Creative](https://kissmycreative.com)
* [Swoop Telecommunications](https://swoop.com.au)

## Todo

* Implement integration tests for log ingestion and tracing with Grafana-LGTM testcontainer
* Expand GoDoc details and add examples
* Try to get tracing integrated into go11y so there is less boilerplate needed

## Notes
<sup>1</sup> sounds like golly
