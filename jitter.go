package again

import (
	"time"

	"github.com/jgfranco17/again/internal/randutil"
)

// noJitter implements a no-op jitter strategy.
type noJitter struct{}

func (noJitter) Apply(delay time.Duration) time.Duration {
	return delay
}

// NoJitter returns a jitter strategy that applies no randomization.
// Use this when you want deterministic retry delays.
func NoJitter() JitterStrategy {
	return noJitter{}
}

// fullJitter implements a full jitter strategy.
type fullJitter struct{}

func (fullJitter) Apply(delay time.Duration) time.Duration {
	if delay <= 0 {
		return 0
	}
	return randutil.Duration(delay)
}

// FullJitter returns a jitter strategy that randomizes delay to [0, delay).
// This is recommended for distributed systems to prevent thundering herd problems.
// The randomization spreads retry attempts uniformly across the delay window.
func FullJitter() JitterStrategy {
	return fullJitter{}
}

// equalJitter implements an equal jitter strategy.
type equalJitter struct{}

func (equalJitter) Apply(delay time.Duration) time.Duration {
	if delay <= 0 {
		return 0
	}
	half := delay / 2
	return half + randutil.Duration(half)
}

// EqualJitter returns a jitter strategy that randomizes delay to [delay/2, delay).
// This provides a compromise between no jitter and full jitter, ensuring some
// minimum delay while still providing randomization to prevent synchronized retries.
func EqualJitter() JitterStrategy {
	return equalJitter{}
}
