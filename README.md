# go11y<sup>1</sup> for observability in Go

Opinionated but simple Go implementation of structured logging and open telemetry
tracing for your application.

## Features

### Structured Logging

go11y wraps the Go standard lib slog package, so it's structured logging with JSON from the outset, just more convenient.

```go
_, o, _ := go11y.Initialise(ctx, nil, os.Stdout, "arg1", "val1")
o.Info("structured logging", nil, "arg2", "val2")
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

go11y doesn't handle the tracing for you (yet) but it does leave room for it so you don't need to go to too much effort to integrate it.


```go
_, o, _ := go11y.Initialise(ctx, nil, os.Stdout)

ctx, span := otel.Tracer("packageName").Start(ctx, "functionName", trace.WithSpanKind(trace.SpanKindClient))

o.Info("structured logging", span)
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

## Todo

* Implement integration tests for log ingestion and tracing with Grafana-LGTM testcontainer
* Expand GoDoc details and add examples
* Try to get tracing integrated into go11y so there is less boilerplate needed

## Notes
<sup>1</sup> sounds like golly
