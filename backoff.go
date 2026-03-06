package again

import (
	"time"
)

// constantBackoff implements a constant delay strategy.
type constantBackoff struct {
	delay time.Duration
}

func (c constantBackoff) Next(attempt int) time.Duration {
	return c.delay
}

// Constant creates a backoff strategy that always returns the same delay.
// This is useful when you want a fixed delay between retries.
func Constant(delay time.Duration) BackoffStrategy {
	if delay < 0 {
		delay = 0
	}
	return constantBackoff{delay: delay}
}

// linearBackoff implements a linear delay strategy.
type linearBackoff struct {
	base time.Duration
}

func (l linearBackoff) Next(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}
	return l.base * time.Duration(attempt)
}

// Linear creates a backoff strategy where delay increases linearly.
// The delay for attempt n is: base * n
func Linear(base time.Duration) BackoffStrategy {
	if base < 0 {
		base = 0
	}
	return linearBackoff{base: base}
}

// exponentialBackoff implements an exponential delay strategy.
type exponentialBackoff struct {
	base time.Duration
	max  time.Duration
}

func (e exponentialBackoff) Next(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	// Calculate 2^(attempt-1) with overflow protection
	var multiplier int64 = 1
	for i := 1; i < attempt; i++ {
		multiplier *= 2
		// Prevent overflow by capping at a reasonable max
		if multiplier > 1<<30 {
			multiplier = 1 << 30
			break
		}
	}

	delay := e.base * time.Duration(multiplier)

	// Apply max delay cap if set
	if e.max > 0 && delay > e.max {
		return e.max
	}

	return delay
}

// Exponential creates a backoff strategy with exponential growth.
// The delay for attempt n is: base * 2^(n-1)
// Growth is capped to prevent overflow and indefinite delays.
func Exponential(base time.Duration) BackoffStrategy {
	if base < 0 {
		base = 0
	}
	// Cap at 1 hour by default to prevent excessively long delays
	return exponentialBackoff{base: base, max: time.Hour}
}

// ExponentialWithMax creates an exponential backoff with a custom maximum delay.
func ExponentialWithMax(base, max time.Duration) BackoffStrategy {
	if base < 0 {
		base = 0
	}
	if max < 0 {
		max = 0
	}
	return exponentialBackoff{base: base, max: max}
}
