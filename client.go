package again

import (
	"context"
	"sync"
)

// Stats holds cumulative execution statistics for a RetryClient.
type Stats struct {
	// TotalRuns is the number of times Do was called.
	TotalRuns int

	// TotalAttempts is the total number of operation attempts across all runs,
	// including the initial attempt and all retries.
	TotalAttempts int

	// Successes is the number of runs that completed without error.
	Successes int

	// Failures is the number of runs that exhausted all attempts or were
	// stopped by a non-retryable error.
	Failures int
}

// RetryClient is a reusable client that executes operations with a fixed retry
// configuration and accumulates execution statistics across calls.
type RetryClient struct {
	config Config
	mu     sync.RWMutex
	stats  Stats
}

// NewRetryClient creates a RetryClient with the given configuration.
func NewRetryClient(cfg Config) *RetryClient {
	return &RetryClient{config: cfg}
}

// Do executes the operation with the client's retry configuration and records
// the result in the client's statistics.
func (c *RetryClient) Do(ctx context.Context, operation func() error) error {
	cfg := c.buildTrackedConfig()

	err := Do(ctx, cfg, operation)

	c.mu.Lock()
	defer c.mu.Unlock()
	c.stats.TotalRuns++
	if err != nil {
		c.stats.Failures++
	} else {
		c.stats.Successes++
	}

	return err
}

// Config returns a copy of the client's current configuration. This can be
// used to run value-returning operations via the package-level DoWithValue:
//
//	result, err := again.DoWithValue(ctx, client.Config(), fn)
func (c *RetryClient) Config() Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config
}

// Stats returns a snapshot of the client's cumulative execution statistics.
func (c *RetryClient) Stats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}

// ResetStats clears all accumulated execution statistics.
func (c *RetryClient) ResetStats() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stats = Stats{}
}

// WithAttempts returns a new RetryClient with the attempt count overridden.
func (c *RetryClient) WithAttempts(attempts int) *RetryClient {
	return c.derive(func(cfg *Config) { cfg.Attempts = attempts })
}

// WithBackoff returns a new RetryClient with the backoff strategy overridden.
func (c *RetryClient) WithBackoff(backoff BackoffStrategy) *RetryClient {
	return c.derive(func(cfg *Config) { cfg.Backoff = backoff })
}

// WithJitter returns a new RetryClient with the jitter strategy overridden.
func (c *RetryClient) WithJitter(jitter JitterStrategy) *RetryClient {
	return c.derive(func(cfg *Config) { cfg.Jitter = jitter })
}

// WithRetryIf returns a new RetryClient with the retry condition overridden.
func (c *RetryClient) WithRetryIf(condition RetryCondition) *RetryClient {
	return c.derive(func(cfg *Config) { cfg.RetryIf = condition })
}

// WithOnRetry returns a new RetryClient with the OnRetry hook overridden.
func (c *RetryClient) WithOnRetry(hook func(attempt int, err error)) *RetryClient {
	return c.derive(func(cfg *Config) { cfg.OnRetry = hook })
}

// derive creates a new RetryClient by copying the current config and applying
// the provided mutation. Stats are not inherited by the derived client.
func (c *RetryClient) derive(apply func(*Config)) *RetryClient {
	c.mu.RLock()
	cfg := c.config
	c.mu.RUnlock()

	apply(&cfg)
	c = &RetryClient{config: cfg}
	return c
}

// buildTrackedConfig returns a copy of the config with an OnRetry wrapper
// that increments TotalAttempts before delegating to any user-provided hook.
func (c *RetryClient) buildTrackedConfig() Config {
	c.mu.RLock()
	cfg := c.config
	c.mu.RUnlock()

	userHook := cfg.OnRetry
	cfg.OnRetry = func(attempt int, err error) {
		c.mu.Lock()
		c.stats.TotalAttempts++
		c.mu.Unlock()

		if userHook != nil {
			userHook(attempt, err)
		}
	}

	return cfg
}
