package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jgfranco17/again"
)

// Example 1: Basic retry with constant backoff
func basicRetry() {
	fmt.Println("=== Example 1: Basic Retry ===")

	cfg := again.Config{
		Attempts: 3,
		Backoff:  again.Constant(500 * time.Millisecond),
		OnRetry: func(attempt int, err error) {
			log.Printf("Retry attempt %d after error: %v", attempt, err)
		},
	}

	attempt := 0
	err := again.Do(context.Background(), cfg, func() error {
		attempt++
		log.Printf("Executing attempt %d", attempt)

		// Simulate failure on first two attempts
		if attempt < 3 {
			return errors.New("temporary failure")
		}

		log.Println("Success!")
		return nil
	})

	if err != nil {
		log.Fatalf("Operation failed: %v", err)
	}

	fmt.Println()
}

// Example 2: Exponential backoff with jitter
func exponentialBackoff() {
	fmt.Println("=== Example 2: Exponential Backoff with Jitter ===")

	cfg := again.Config{
		Attempts: 5,
		Backoff:  again.Exponential(100 * time.Millisecond),
		Jitter:   again.FullJitter(),
		OnRetry: func(attempt int, err error) {
			log.Printf("Retry attempt %d (exponential backoff)", attempt)
		},
	}

	attempt := 0
	start := time.Now()

	err := again.Do(context.Background(), cfg, func() error {
		attempt++
		elapsed := time.Since(start)
		log.Printf("Attempt %d at %v", attempt, elapsed.Round(time.Millisecond))

		if attempt < 4 {
			return errors.New("still failing")
		}

		log.Println("Operation succeeded!")
		return nil
	})

	if err != nil {
		log.Fatalf("Failed after retries: %v", err)
	}

	fmt.Println()
}

// Example 3: Using NewConfig with defaults
func defaultConfig() {
	fmt.Println("=== Example 3: Default Configuration ===")

	// NewConfig provides sensible defaults:
	// - 3 attempts
	// - Exponential backoff (100ms base)
	// - Full jitter
	// - Retry all errors
	cfg := again.NewConfig()
	cfg.OnRetry = func(attempt int, err error) {
		log.Printf("Retry %d with default config", attempt)
	}

	attempt := 0
	err := again.Do(context.Background(), cfg, func() error {
		attempt++
		log.Printf("Attempt %d", attempt)

		if attempt < 2 {
			return errors.New("transient error")
		}

		log.Println("Success with defaults!")
		return nil
	})

	if err != nil {
		log.Fatalf("Failed: %v", err)
	}

	fmt.Println()
}

// Example 4: Conditional retry (only retry specific errors)
func conditionalRetry() {
	fmt.Println("=== Example 4: Conditional Retry ===")

	// Define specific errors
	errTemporary := errors.New("temporary error")
	errFatal := errors.New("fatal error")

	cfg := again.Config{
		Attempts: 5,
		Backoff:  again.Linear(200 * time.Millisecond),
		RetryIf:  again.IfErrorIs(errTemporary),
		OnRetry: func(attempt int, err error) {
			log.Printf("Retrying attempt %d (only temporary errors)", attempt)
		},
	}

	// Case 1: Temporary error (will retry)
	log.Println("Case 1: Temporary error (will retry)")
	attempt := 0
	err := again.Do(context.Background(), cfg, func() error {
		attempt++
		log.Printf("Attempt %d", attempt)

		if attempt < 3 {
			return errTemporary
		}

		log.Println("Recovered from temporary error!")
		return nil
	})

	if err != nil {
		log.Printf("Failed: %v", err)
	}

	// Case 2: Fatal error (won't retry)
	log.Println("\nCase 2: Fatal error (won't retry)")
	attempt = 0
	err = again.Do(context.Background(), cfg, func() error {
		attempt++
		log.Printf("Attempt %d", attempt)
		return errFatal // This won't be retried
	})

	if err != nil {
		log.Printf("Failed immediately: %v", err)
	}

	fmt.Println()
}

// Example 5: Context cancellation
func contextCancellation() {
	fmt.Println("=== Example 5: Context Cancellation ===")

	cfg := again.Config{
		Attempts: 10,
		Backoff:  again.Constant(500 * time.Millisecond),
		OnRetry: func(attempt int, err error) {
			log.Printf("Retry attempt %d", attempt)
		},
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	attempt := 0
	start := time.Now()

	err := again.Do(ctx, cfg, func() error {
		attempt++
		elapsed := time.Since(start)
		log.Printf("Attempt %d at %v", attempt, elapsed.Round(time.Millisecond))

		// This will keep failing, but context will cancel after 2 seconds
		return errors.New("persistent error")
	})

	elapsed := time.Since(start)
	log.Printf("Stopped after %v with %d attempts", elapsed.Round(time.Millisecond), attempt)

	if again.IsRetryError(err) {
		log.Printf("Retry error: %v", err)
	}

	fmt.Println()
}

// Example 6: DoWithValue for operations returning values
func valueReturning() {
	fmt.Println("=== Example 6: Returning Values ===")

	cfg := again.Config{
		Attempts: 4,
		Backoff:  again.Exponential(100 * time.Millisecond),
		Jitter:   again.EqualJitter(),
		OnRetry: func(attempt int, err error) {
			log.Printf("Fetching data (attempt %d)", attempt)
		},
	}

	attempt := 0
	result, err := again.DoWithValue(context.Background(), cfg, func() (string, error) {
		attempt++
		log.Printf("Fetching data, attempt %d", attempt)

		if attempt < 3 {
			return "", errors.New("network error")
		}

		return "Important Data", nil
	})

	if err != nil {
		log.Fatalf("Failed to fetch data: %v", err)
	}

	log.Printf("Successfully retrieved: %s", result)
	fmt.Println()
}

// Example 7: Combining retry conditions
func combinedConditions() {
	fmt.Println("=== Example 7: Combined Retry Conditions ===")

	// Define multiple error types
	errNetwork := errors.New("network error")
	errTimeout := errors.New("timeout error")
	errPermanent := errors.New("permanent error")

	// Retry on either network or timeout errors
	cfg := again.Config{
		Attempts: 4,
		Backoff:  again.Constant(300 * time.Millisecond),
		RetryIf: again.AnyOf(
			again.IfErrorIs(errNetwork),
			again.IfErrorIs(errTimeout),
		),
		OnRetry: func(attempt int, err error) {
			log.Printf("Retrying after: %v", err)
		},
	}

	// Test with network error (will retry)
	log.Println("Case 1: Network error (retryable)")
	attempt := 0
	err := again.Do(context.Background(), cfg, func() error {
		attempt++
		if attempt < 2 {
			return errNetwork
		}
		log.Println("Recovery successful!")
		return nil
	})

	if err != nil {
		log.Printf("Failed: %v", err)
	}

	// Test with permanent error (won't retry)
	log.Println("\nCase 2: Permanent error (not retryable)")
	attempt = 0
	err = again.Do(context.Background(), cfg, func() error {
		attempt++
		log.Printf("Attempt %d", attempt)
		return errPermanent
	})

	if err != nil {
		log.Printf("Failed without retry: %v", err)
	}

	fmt.Println()
}

func main() {
	log.SetFlags(log.Ltime | log.Lmicroseconds)

	fmt.Println("Go Retry Framework - Basic Examples")
	fmt.Println("====================================")
	fmt.Println()

	// Run all examples
	basicRetry()
	exponentialBackoff()
	defaultConfig()
	conditionalRetry()
	contextCancellation()
	valueReturning()
	combinedConditions()

	fmt.Println("All examples completed successfully!")
}
