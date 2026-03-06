package again

import (
	"time"
)

// BackoffStrategy determines the delay duration before each retry attempt.
// Implementations must be safe for concurrent use.
type BackoffStrategy interface {
	// Next returns the delay duration for the given attempt number.
	// Attempt numbers start at 1 (first retry after initial failure).
	Next(attempt int) time.Duration
}

// JitterStrategy applies randomization to a delay duration to prevent
// thundering herd problems in distributed systems.
// Implementations must be safe for concurrent use.
type JitterStrategy interface {
	// Apply returns a potentially randomized version of the input duration.
	Apply(delay time.Duration) time.Duration
}

// RetryCondition determines whether a retry should be attempted based on
// the error returned from the operation.
// It returns true if the operation should be retried, false otherwise.
type RetryCondition func(err error) bool

// Config defines the retry behavior for an operation.
// Zero values are not valid; use NewConfig() or set fields explicitly.
type Config struct {
	// Attempts specifies the maximum number of retry attempts.
	// A value of 1 means the operation will be tried once with no retries.
	// A value of 0 or negative will result in no execution.
	Attempts int

	// Backoff determines the delay strategy between retry attempts.
	// If nil, no delay is applied between retries (not recommended).
	Backoff BackoffStrategy

	// Jitter applies randomization to backoff delays.
	// If nil, no jitter is applied.
	Jitter JitterStrategy

	// RetryIf determines whether an error should trigger a retry.
	// If nil, all errors will trigger retries (equivalent to Always).
	RetryIf RetryCondition

	// OnRetry is called before each retry attempt (not before the initial attempt).
	// It receives the attempt number (starting at 1) and the error that triggered the retry.
	// This hook is useful for logging or metrics.
	// If nil, no callback is invoked.
	OnRetry func(attempt int, err error)
}

// NewConfig creates a Config with sensible defaults:
//   - 3 attempts
//   - Exponential backoff starting at 100ms
//   - Full jitter
//   - Retry on all errors
func NewConfig() Config {
	return Config{
		Attempts: 3,
		Backoff:  Exponential(100 * time.Millisecond),
		Jitter:   FullJitter(),
		RetryIf:  Always,
		OnRetry:  nil,
	}
}

// Validate checks if the configuration is valid and returns an error if not.
func (c Config) Validate() error {
	if c.Attempts <= 0 {
		return &ConfigError{Field: "Attempts", Reason: "must be positive"}
	}
	return nil
}

// ConfigError represents an invalid configuration.
type ConfigError struct {
	Field  string
	Reason string
}

func (e *ConfigError) Error() string {
	return "invalid config: " + e.Field + " " + e.Reason
}
