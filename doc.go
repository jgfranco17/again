// Package again provides a minimal, context-aware retry framework for Go
// with pluggable backoff and jitter strategies and zero external dependencies.
//
// The library is designed for backend services, CLI tools, distributed systems,
// API clients, and database integrations that need reliable retry logic.
//
// # Basic Usage
//
// The simplest way to use the retry framework is with the default configuration:
//
//	cfg := again.NewConfig()
//	err := again.Do(ctx, cfg, func() error {
//	    return callExternalService()
//	})
//
// # Custom Configuration
//
// Configure retry behavior with backoff strategies, jitter, and retry conditions:
//
//	cfg := again.Config{
//	    Attempts: 5,
//	    Backoff:  again.Exponential(100 * time.Millisecond),
//	    Jitter:   again.FullJitter(),
//	    RetryIf:  again.TransientErrors,
//	    OnRetry: func(attempt int, err error) {
//	        log.Printf("retry %d: %v", attempt, err)
//	    },
//	}
//
// # Backoff Strategies
//
// Built-in backoff strategies determine the delay between retry attempts:
//
//   - Constant(d): Fixed delay
//   - Linear(d): Linear growth (d * attempt)
//   - Exponential(d): Exponential growth (d * 2^(attempt-1))
//
// # Jitter Strategies
//
// Jitter strategies randomize delays to prevent thundering herd problems:
//
//   - NoJitter(): No randomization
//   - FullJitter(): Random delay in [0, delay)
//   - EqualJitter(): Random delay in [delay/2, delay)
//
// # Retry Conditions
//
// Control which errors trigger retries:
//
//   - Always: Retry all errors
//   - Never: Never retry
//   - TransientErrors: Retry network/timeout errors
//   - IfErrorIs(target): Retry specific error
//   - IfErrorAs(target): Retry error type
//
// Combine conditions with OnlyIf (AND), AnyOf (OR), and Not (negation).
//
// # Context Awareness
//
// All retry operations respect context cancellation:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//	err := again.Do(ctx, cfg, operation)
//
// # Error Handling
//
// When all retries are exhausted, a RetryError is returned:
//
//	err := again.Do(ctx, cfg, operation)
//	if retryErr, ok := err.(*again.RetryError); ok {
//	    log.Printf("failed after %d attempts: %v", retryErr.Attempts, retryErr.LastErr)
//	}
//
// # Generic Value Returns
//
// Use DoWithValue for operations that return values:
//
//	result, err := again.DoWithValue(ctx, cfg, func() (string, error) {
//	    return fetchData()
//	})
package again
