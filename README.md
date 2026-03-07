# Again

A minimal, idiomatic retry framework for Go that provides context-aware retry execution with
pluggable backoff and jitter strategies while maintaining zero external dependencies.

## Features

- **Zero Dependencies**: Pure Go implementation with no external dependencies in production code
- **Context-Aware**: Full support for context cancellation and timeouts
- **Pluggable Strategies**: Customizable backoff and jitter strategies
- **Flexible Conditions**: Composable retry conditions for fine-grained control
- **Generic Support**: Type-safe retry for value-returning operations via `DoWithValue[T]`
- **Production-Ready**: Comprehensive test coverage (95%+) and battle-tested patterns

## Installation

```bash
go get github.com/jgfranco17/again
```

## Quick Start

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "time"

    "github.com/jgfranco17/again"
)

func main() {
    cfg := again.Config{
        Attempts: 3,
        Backoff:  again.Exponential(100 * time.Millisecond),
        Jitter:   again.FullJitter(),
        RetryIf:  again.TransientErrors,
    }

    err := again.Do(context.Background(), cfg, func() error {
        // Your operation here
        return performOperation()
    })

    if err != nil {
        fmt.Printf("Failed after retries: %v\n", err)
    }
}
```

## Backoff Strategies

### Constant Backoff

Fixed delay between retries.

```go
again.Constant(1 * time.Second) // Always wait 1 second
```

### Linear Backoff

Delay increases linearly with attempt number.

```go
again.Linear(500 * time.Millisecond) // 500ms, 1s, 1.5s, 2s, ...
```

### Exponential Backoff

Delay doubles with each attempt, up to 1 hour maximum.

```go
again.Exponential(100 * time.Millisecond) // 100ms, 200ms, 400ms, 800ms, ...
```

### Exponential with Custom Max

Exponential growth with custom maximum delay.

```go
again.ExponentialWithMax(100 * time.Millisecond, 10 * time.Second)
```

## Jitter Strategies

Jitter helps prevent thundering herd problems when many clients retry simultaneously.

### No Jitter

Deterministic delays with no randomization (default for predictability).

```go
again.NoJitter()
```

### Full Jitter

Random delay between 0 and the calculated backoff time.

```go
again.FullJitter() // Random in [0, delay)
```

### Equal Jitter

Random delay between half and full backoff time.

```go
again.EqualJitter() // Random in [delay/2, delay)
```

## Retry Conditions

Control which errors trigger retries using composable conditions.

### Built-in Conditions

```go
again.Always          // Retry all errors
again.Never           // Never retry
again.TransientErrors // Retry temporary, timeout, and network errors
```

### Error Type Conditions

```go
again.IfErrorIs(context.DeadlineExceeded)
again.IfErrorAs(target)
```

### Combinators

```go
// Retry if ANY condition matches
again.AnyOf(
    again.IfErrorIs(ErrTimeout),
    again.IfErrorIs(ErrConnectionLost),
)

// Inverse condition
again.Not(again.IfErrorIs(ErrPermanent))

// Custom condition
again.OnlyIf(func(err error) bool {
    return err.Error() == "rate limited"
})
```

## RetryClient

`RetryClient` is a reusable, stateful alternative to the package-level `Do` function —
analologous to `http.Client` versus `http.Get`. Prefer it when you need shared
configuration across multiple call sites, per-call overrides, or runtime statistics.

### Creating a client

```go
client := again.NewRetryClient(again.Config{
    Attempts: 5,
    Backoff:  again.Exponential(100 * time.Millisecond),
    Jitter:   again.FullJitter(),
    RetryIf:  again.TransientErrors,
})
```

### Executing operations

```go
err := client.Do(ctx, func() error {
    return callService()
})
```

### Deriving specialised clients

`With*` methods return a **new** client with the override applied — the receiver
is never mutated, so a package-level default is safe to share and derive from.

```go
// Application-wide default
var defaultClient = again.NewRetryClient(again.Config{
    Attempts: 3,
    Backoff:  again.Exponential(100 * time.Millisecond),
    RetryIf:  again.TransientErrors,
})

// Critical path — more attempts, additional error types
criticalClient := defaultClient.
    WithAttempts(6).
    WithRetryIf(again.AnyOf(again.TransientErrors, again.IfErrorIs(ErrRateLimit))).
    WithOnRetry(func(attempt int, err error) {
        log.Printf("critical retry %d: %v", attempt, err)
    })
```

Available builder methods: `WithAttempts`, `WithBackoff`, `WithJitter`, `WithRetryIf`, `WithOnRetry`.

### Value-returning operations

Go methods cannot carry additional type parameters, so use `Config()` to pass the
client's configuration to the package-level `DoWithValue`:

```go
result, err := again.DoWithValue(ctx, client.Config(), func() (MyType, error) {
    return fetchData()
})
```

### Execution statistics

The client accumulates statistics across every `Do` call. Use them for logging,
metrics emission, or circuit-breaking logic:

```go
stats := client.Stats()
fmt.Printf("runs: %d, succeeded: %d, failed: %d, total attempts: %d\n",
    stats.TotalRuns, stats.Successes, stats.Failures, stats.TotalAttempts)

// Reset before the next reporting window
client.ResetStats()
```

## Generic Value-Returning Operations

Use `DoWithValue[T]` for operations that return values:

```go
result, err := again.DoWithValue(ctx, cfg, func() (MyType, error) {
    return fetchData()
})
```

## Configuration

The `Config` struct provides complete control over retry behavior:

```go
cfg := again.Config{
    Attempts: 5,              // Maximum retry attempts (required)
    Backoff:  backoffStrategy, // Backoff strategy (required)
    Jitter:   jitterStrategy,  // Optional: defaults to NoJitter()
    RetryIf:  condition,       // Optional: defaults to Always
    OnRetry: func(attempt int, err error) {
        log.Printf("Retry %d: %v", attempt, err)
    },
}
```

### Default Configuration

Use `NewConfig()` for sensible defaults:

```go
cfg := again.NewConfig() // 3 attempts, exponential backoff 100ms, full jitter
```

## Context Support

All retry operations respect context cancellation and timeouts:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

err := again.Do(ctx, cfg, operation)
```

## Error Handling

Retrieve detailed information about retry failures:

```go
err := again.Do(ctx, cfg, operation)

if retryErr, ok := err.(*again.RetryError); ok {
    fmt.Printf("Failed after %d attempts\n", retryErr.Attempts)
    fmt.Printf("Last error: %v\n", retryErr.LastErr)
}

// Or use the helper
if again.IsRetryError(err) {
    fmt.Println("All retries exhausted")
}
```

## Examples

The `examples/` directory contains comprehensive demonstrations:

- **[examples/basic](examples/basic/main.go)**: Core framework features and patterns
- **[examples/client](examples/client/main.go)**: `RetryClient` usage — shared clients,
  derived clients, stats monitoring
- **[examples/http](examples/http/main.go)**: HTTP client retry scenarios
- **[examples/database](examples/database/main.go)**: Database connection and transaction retries

Run examples:

```bash
go run examples/basic/main.go
go run examples/client/main.go
go run examples/http/main.go
go run examples/database/main.go
```

## Testing

Run the complete test suite:

```bash
go test ./... -cover
```

The framework maintains 95%+ test coverage with comprehensive unit and integration tests.

## License

MIT License - see [LICENSE](LICENSE) for details.

## Contributing

Contributions are welcome! Please ensure:

- All tests pass: `go test ./...`
- Code is formatted: `go fmt ./...`
- Linting passes: `go vet ./...`
- Cyclomatic complexity < 15 for all functions
