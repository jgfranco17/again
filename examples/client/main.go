package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jgfranco17/again"
)

var (
	errTemporary = errors.New("temporary error")
	errRateLimit = errors.New("rate limited")
)

// Example 1: Reusing a shared client across multiple operations.
// The client holds a base configuration so callers don't repeat themselves,
// and accumulates stats across every call made through it.
func sharedClient() {
	fmt.Println("=== Example 1: Shared Client ===")

	client := again.NewRetryClient(again.Config{
		Attempts: 4,
		Backoff:  again.Exponential(50 * time.Millisecond),
		Jitter:   again.FullJitter(),
		RetryIf:  again.IfErrorIs(errTemporary),
		OnRetry: func(attempt int, err error) {
			log.Printf("  retry %d: %v", attempt, err)
		},
	})

	// Simulate three distinct service calls through the same client.
	operations := []struct {
		name    string
		failFor int // number of leading attempts that will fail
	}{
		{"fetch user", 1},
		{"fetch orders", 2},
		{"fetch inventory", 0},
	}

	for _, op := range operations {
		calls := 0
		name := op.name
		failFor := op.failFor

		log.Printf("Running: %s", name)
		err := client.Do(context.Background(), func() error {
			calls++
			if calls <= failFor {
				return errTemporary
			}
			log.Printf("  %s succeeded on attempt %d", name, calls)
			return nil
		})

		if err != nil {
			log.Printf("  %s failed: %v", name, err)
		}
	}

	stats := client.Stats()
	fmt.Printf(
		"\nStats after %d runs: %d succeeded, %d failed, %d total attempts\n",
		stats.TotalRuns,
		stats.Successes,
		stats.Failures,
		stats.TotalAttempts,
	)
	fmt.Println()
}

// Example 2: Deriving a specialised client from a shared base.
// With* methods return a new client — the base is never mutated, so it is safe
// to hold a package-level default and derive per-call variants from it.
func derivedClients() {
	fmt.Println("=== Example 2: Derived Clients ===")

	// Application-wide default: conservative, transient errors only.
	base := again.NewRetryClient(again.Config{
		Attempts: 3,
		Backoff:  again.Exponential(100 * time.Millisecond),
		RetryIf:  again.TransientErrors,
	})

	// Critical path: more attempts and retry rate-limit errors too.
	critical := base.
		WithAttempts(6).
		WithRetryIf(again.AnyOf(again.TransientErrors, again.IfErrorIs(errRateLimit))).
		WithOnRetry(func(attempt int, err error) {
			log.Printf("  [critical] retry %d: %v", attempt, err)
		})

	// Best-effort path: single attempt, never retry.
	bestEffort := base.
		WithAttempts(1).
		WithRetryIf(again.Never)

	attempt := 0
	log.Println("Critical path:")
	critical.Do(context.Background(), func() error { //nolint:errcheck
		attempt++
		if attempt < 3 {
			return errRateLimit
		}
		log.Println("  critical succeeded")
		return nil
	})

	log.Println("Best-effort path:")
	err := bestEffort.Do(context.Background(), func() error {
		return errTemporary
	})
	if err != nil {
		log.Printf("  best-effort gave up immediately (as expected): %v", err)
	}

	// Base client stats are unaffected by derived clients.
	fmt.Printf(
		"\nBase client runs: %d (derived clients have independent stats)\n",
		base.Stats().TotalRuns,
	)
	fmt.Println()
}

// Example 3: Value-returning operations via DoWithValue.
// Because methods cannot carry additional type parameters in Go, the client
// exposes Config() so callers can pass it directly to the package-level
// DoWithValue without duplicating configuration.
func valueReturning() {
	fmt.Println("=== Example 3: Value-Returning Operations ===")

	client := again.NewRetryClient(again.Config{
		Attempts: 4,
		Backoff:  again.Linear(30 * time.Millisecond),
		OnRetry: func(attempt int, err error) {
			log.Printf("  retry %d: %v", attempt, err)
		},
	})

	calls := 0
	result, err := again.DoWithValue(context.Background(), client.Config(), func() (string, error) {
		calls++
		if calls < 3 {
			return "", errTemporary
		}
		return fmt.Sprintf("fetched record after %d attempts", calls), nil
	})

	if err != nil {
		log.Fatalf("failed: %v", err)
	}

	log.Printf("Result: %q", result)
	fmt.Println()
}

// Example 4: Monitoring via stats and periodic resets.
// Stats accumulate indefinitely until Reset is called, making them suitable
// for periodic reporting (e.g., flushing to a metrics system every minute).
func statsMonitoring() {
	fmt.Println("=== Example 4: Stats Monitoring ===")

	client := again.NewRetryClient(again.Config{
		Attempts: 3,
		Backoff:  again.Constant(10 * time.Millisecond),
	})

	runBatch := func(label string, failUntil int) {
		calls := 0
		for i := 0; i < 5; i++ {
			count := 0
			client.Do(context.Background(), func() error { //nolint:errcheck
				count++
				calls++
				if calls <= failUntil {
					return errTemporary
				}
				return nil
			})
		}
		s := client.Stats()
		log.Printf(
			"%s — runs: %d, succeeded: %d, failed: %d, total attempts: %d",
			label, s.TotalRuns, s.Successes, s.Failures, s.TotalAttempts,
		)
	}

	runBatch("First window (some failures)", 6)

	// Simulate flushing stats to a metrics system, then resetting.
	log.Println("Flushing metrics and resetting stats...")
	client.ResetStats()

	runBatch("Second window (clean slate)", 0)

	fmt.Println()
}

func main() {
	log.SetFlags(log.Ltime | log.Lmicroseconds)

	fmt.Println("Go Retry Framework - RetryClient Examples")
	fmt.Println("==========================================")
	fmt.Println()

	sharedClient()
	derivedClients()
	valueReturning()
	statsMonitoring()

	fmt.Println("All client examples completed!")
}
