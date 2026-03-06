package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/jgfranco17/again"
)

var (
	ErrConnectionFailed  = errors.New("connection failed")
	ErrDeadlock          = errors.New("deadlock detected")
	ErrConnectionTimeout = errors.New("connection timeout")
)

type MockDatabase struct {
	connected   atomic.Bool
	attemptNum  atomic.Int32
	failureMode string
}

func NewMockDatabase() *MockDatabase {
	return &MockDatabase{}
}

func (db *MockDatabase) Connect() error {
	attempt := db.attemptNum.Add(1)
	log.Printf("Connection attempt #%d", attempt)

	if db.failureMode == "flaky" && attempt < 3 {
		time.Sleep(50 * time.Millisecond)
		return ErrConnectionTimeout
	}

	time.Sleep(20 * time.Millisecond)
	db.connected.Store(true)
	log.Println("Connected to database")
	return nil
}

func (db *MockDatabase) Close() error {
	db.connected.Store(false)
	log.Println("Database connection closed")
	return nil
}

func example1ConnectionRetry() {
	fmt.Println("=== Example 1: Database Connection Retry ===")

	db := NewMockDatabase()
	db.failureMode = "flaky"

	cfg := again.Config{
		Attempts: 5,
		Backoff:  again.Exponential(100 * time.Millisecond),
		Jitter:   again.FullJitter(),
		RetryIf:  again.Always, // Retry all errors
		OnRetry: func(attempt int, err error) {
			log.Printf("Connection failed, retrying (attempt %d): %v", attempt, err)
		},
	}

	err := again.Do(context.Background(), cfg, func() error {
		return db.Connect()
	})

	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	log.Println("Successfully connected")
	db.Close()
	fmt.Println()
}

func example2TransactionRetry() {
	fmt.Println("=== Example 2: Transaction Retry (Deadlock) ===")

	attemptCount := 0
	cfg := again.Config{
		Attempts: 4,
		Backoff:  again.ExponentialWithMax(50*time.Millisecond, 1*time.Second),
		Jitter:   again.EqualJitter(),
		RetryIf: func(err error) bool {
			return errors.Is(err, ErrDeadlock)
		},
		OnRetry: func(attempt int, err error) {
			log.Printf("Deadlock detected, retrying (attempt %d)", attempt)
		},
	}

	err := again.Do(context.Background(), cfg, func() error {
		attemptCount++
		log.Printf("Starting transaction (attempt %d)", attemptCount)

		if attemptCount < 2 {
			time.Sleep(30 * time.Millisecond)
			return ErrDeadlock
		}

		log.Println("Transaction committed successfully")
		return nil
	})

	if err != nil {
		log.Fatalf("Transaction failed: %v", err)
	}

	fmt.Println()
}

func example3QueryWithReconnect() {
	fmt.Println("=== Example 3: Query with Auto-Reconnect ===")

	db := NewMockDatabase()
	db.Connect()

	queryAttempt := 0
	cfg := again.Config{
		Attempts: 4,
		Backoff:  again.Linear(200 * time.Millisecond),
		OnRetry: func(attempt int, err error) {
			log.Printf("Query failed, reconnecting (attempt %d)", attempt)
			db.Close()
			db.attemptNum.Store(0)
			db.Connect()
		},
	}

	result, err := again.DoWithValue(context.Background(), cfg, func() (string, error) {
		queryAttempt++
		log.Printf("Executing query (attempt %d)", queryAttempt)

		if !db.connected.Load() {
			return "", ErrConnectionFailed
		}

		if queryAttempt < 3 {
			db.connected.Store(false)
			return "", errors.New("connection lost")
		}

		time.Sleep(10 * time.Millisecond)
		return "Query result: 42 rows", nil
	})

	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	log.Printf("Success! %s", result)
	db.Close()
	fmt.Println()
}

func example4ContextTimeout() {
	fmt.Println("=== Example 4: Context Timeout ===")

	cfg := again.Config{
		Attempts: 10,
		Backoff:  again.Constant(200 * time.Millisecond),
		OnRetry: func(attempt int, err error) {
			log.Printf("Slow query, retrying (attempt %d)", attempt)
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := again.Do(ctx, cfg, func() error {
		log.Println("Executing slow query...")
		time.Sleep(500 * time.Millisecond)
		return errors.New("query timeout")
	})

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("Context timeout: %v", err)
		} else {
			log.Printf("Failed: %v", err)
		}
	}

	fmt.Println()
}

func main() {
	log.SetFlags(log.Ltime | log.Lmicroseconds)

	fmt.Println("Go Retry Framework - Database Examples")
	fmt.Println("=======================================")
	fmt.Println()

	example1ConnectionRetry()
	example2TransactionRetry()
	example3QueryWithReconnect()
	example4ContextTimeout()

	fmt.Println("All database examples completed!")
}
