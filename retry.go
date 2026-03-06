package again

import (
	"context"
	"time"
)

// retryState holds the mutable state during retry execution.
type retryState struct {
	attempt int
	lastErr error
}

// shouldContinue checks if we should continue retrying.
func (s *retryState) shouldContinue(cfg Config) bool {
	if s.attempt >= cfg.Attempts {
		return false
	}
	if cfg.RetryIf != nil && !cfg.RetryIf(s.lastErr) {
		return false
	}
	return true
}

// Do executes the provided function with retry logic according to the configuration.
// It returns nil on success, or a RetryError if all attempts are exhausted.
//
// The function respects context cancellation and will stop retrying immediately
// if the context is cancelled.
//
// Example:
//
//	cfg := again.NewConfig()
//	err := again.Do(ctx, cfg, func() error {
//	    return callExternalService()
//	})
func Do(ctx context.Context, cfg Config, fn func() error) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	if fn == nil {
		return &ConfigError{Field: "fn", Reason: "cannot be nil"}
	}

	state := &retryState{}

	for state.attempt = 1; state.attempt <= cfg.Attempts; state.attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return handleContextDone(state, ctx.Err())
		default:
		}

		// Execute operation
		state.lastErr = fn()
		if state.lastErr == nil {
			return nil
		}

		// Check if we should continue retrying
		if !state.shouldContinue(cfg) {
			return &RetryError{
				Attempts: state.attempt,
				LastErr:  state.lastErr,
			}
		}

		// Call retry hook
		if cfg.OnRetry != nil {
			cfg.OnRetry(state.attempt, state.lastErr)
		}

		// Wait with backoff and jitter
		delay := calculateDelay(cfg, state.attempt)
		if !waitWithContext(ctx, delay) {
			return &RetryError{
				Attempts: state.attempt,
				LastErr:  state.lastErr,
			}
		}
	}

	return &RetryError{
		Attempts: cfg.Attempts,
		LastErr:  state.lastErr,
	}
}

// waitWithContext sleeps for the specified delay or until context is cancelled.
func waitWithContext(ctx context.Context, delay time.Duration) bool {
	if delay <= 0 {
		return true
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// calculateDelay computes the delay with backoff and jitter.
func calculateDelay(cfg Config, attempt int) time.Duration {
	if cfg.Backoff == nil {
		return 0
	}
	delay := cfg.Backoff.Next(attempt)
	if cfg.Jitter != nil {
		delay = cfg.Jitter.Apply(delay)
	}
	return delay
}

// handleContextDone returns appropriate error when context is cancelled.
func handleContextDone(state *retryState, ctxErr error) error {
	if state.lastErr != nil {
		return &RetryError{
			Attempts: state.attempt - 1,
			LastErr:  state.lastErr,
		}
	}
	return ctxErr
}

// DoWithValue executes the provided function with retry logic and returns a value.
// This is useful when the operation returns both a value and an error.
//
// Example:
//
//	cfg := again.NewConfig()
//	result, err := again.DoWithValue(ctx, cfg, func() (string, error) {
//	    return fetchData()
//	})
func DoWithValue[T any](ctx context.Context, cfg Config, fn func() (T, error)) (T, error) {
	var result T

	if err := cfg.Validate(); err != nil {
		return result, err
	}
	if fn == nil {
		return result, &ConfigError{Field: "fn", Reason: "cannot be nil"}
	}

	state := &retryState{}

	for state.attempt = 1; state.attempt <= cfg.Attempts; state.attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return result, handleContextDone(state, ctx.Err())
		default:
		}

		// Execute operation
		val, err := fn()
		if err == nil {
			return val, nil
		}

		state.lastErr = err

		// Check if we should continue retrying
		if !state.shouldContinue(cfg) {
			return result, &RetryError{
				Attempts: state.attempt,
				LastErr:  state.lastErr,
			}
		}

		// Call retry hook
		if cfg.OnRetry != nil {
			cfg.OnRetry(state.attempt, state.lastErr)
		}

		// Wait with backoff and jitter
		delay := calculateDelay(cfg, state.attempt)
		if !waitWithContext(ctx, delay) {
			return result, &RetryError{
				Attempts: state.attempt,
				LastErr:  state.lastErr,
			}
		}
	}

	return result, &RetryError{
		Attempts: cfg.Attempts,
		LastErr:  state.lastErr,
	}
}
