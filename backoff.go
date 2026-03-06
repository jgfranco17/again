package again

import "time"

// Exponential creates a backoff strategy with exponential growth.
// The delay for attempt n is: base * 2^(n-1)
// Placeholder implementation - will be fully implemented in next iteration.
func Exponential(base time.Duration) BackoffStrategy {
	return nil // TODO: implement
}
