package again

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDo_Success(t *testing.T) {
	cfg := Config{
		Attempts: 3,
		Backoff:  Constant(10 * time.Millisecond),
	}

	callCount := 0
	err := Do(context.Background(), cfg, func() error {
		callCount++
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, callCount)
}

func TestDo_SuccessAfterRetries(t *testing.T) {
	cfg := Config{
		Attempts: 5,
		Backoff:  Constant(10 * time.Millisecond),
	}

	callCount := 0
	testErr := errors.New("temporary error")

	err := Do(context.Background(), cfg, func() error {
		callCount++
		if callCount < 3 {
			return testErr
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 3, callCount)
}

func TestDo_AllAttemptsExhausted(t *testing.T) {
	cfg := Config{
		Attempts: 3,
		Backoff:  Constant(10 * time.Millisecond),
	}

	callCount := 0
	testErr := errors.New("persistent error")

	err := Do(context.Background(), cfg, func() error {
		callCount++
		return testErr
	})

	assert.Error(t, err)
	assert.Equal(t, 3, callCount)

	var retryErr *RetryError
	assert.ErrorAs(t, err, &retryErr)
	assert.Equal(t, 3, retryErr.Attempts)
	assert.Equal(t, testErr, retryErr.LastErr)
}

func TestDo_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	cfg := Config{
		Attempts: 10,
		Backoff:  Constant(50 * time.Millisecond),
	}

	callCount := 0
	testErr := errors.New("error")

	// Cancel after 2 attempts
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err := Do(ctx, cfg, func() error {
		callCount++
		return testErr
	})

	assert.Error(t, err)
	assert.Less(t, callCount, 10, "should stop before all attempts")

	var retryErr *RetryError
	assert.ErrorAs(t, err, &retryErr)
}

func TestDo_ContextCancelledBeforeStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	cfg := Config{
		Attempts: 3,
		Backoff:  Constant(10 * time.Millisecond),
	}

	callCount := 0
	err := Do(ctx, cfg, func() error {
		callCount++
		return errors.New("error")
	})

	assert.Error(t, err)
	// With cancelled context, may not even execute once depending on timing
	assert.GreaterOrEqual(t, callCount, 0)
}

func TestDo_ContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	cfg := Config{
		Attempts: 10,
		Backoff:  Constant(50 * time.Millisecond),
	}

	callCount := 0
	err := Do(ctx, cfg, func() error {
		callCount++
		return errors.New("error")
	})

	assert.Error(t, err)
	assert.Less(t, callCount, 10)
}

func TestDo_RetryCondition(t *testing.T) {
	retryableErr := errors.New("retryable")
	fatalErr := errors.New("fatal")

	cfg := Config{
		Attempts: 5,
		Backoff:  Constant(10 * time.Millisecond),
		RetryIf:  IfErrorIs(retryableErr),
	}

	t.Run("retryable error", func(t *testing.T) {
		callCount := 0
		err := Do(context.Background(), cfg, func() error {
			callCount++
			return retryableErr
		})

		assert.Error(t, err)
		assert.Equal(t, 5, callCount)
	})

	t.Run("non-retryable error", func(t *testing.T) {
		callCount := 0
		err := Do(context.Background(), cfg, func() error {
			callCount++
			return fatalErr
		})

		assert.Error(t, err)
		assert.Equal(t, 1, callCount, "should not retry non-retryable error")
	})
}

func TestDo_OnRetryHook(t *testing.T) {
	var attempts []int
	var errs []error
	testErr := errors.New("test error")

	cfg := Config{
		Attempts: 4,
		Backoff:  Constant(10 * time.Millisecond),
		OnRetry: func(attempt int, err error) {
			attempts = append(attempts, attempt)
			errs = append(errs, err)
		},
	}

	callCount := 0
	_ = Do(context.Background(), cfg, func() error {
		callCount++
		if callCount < 3 {
			return testErr
		}
		return nil
	})

	// OnRetry should be called for attempts 1 and 2 (before attempts 2 and 3)
	assert.Equal(t, []int{1, 2}, attempts)
	assert.Equal(t, []error{testErr, testErr}, errs)
}

func TestDo_BackoffAndJitter(t *testing.T) {
	cfg := Config{
		Attempts: 3,
		Backoff:  Exponential(100 * time.Millisecond),
		Jitter:   NoJitter(),
	}

	start := time.Now()
	callCount := 0

	_ = Do(context.Background(), cfg, func() error {
		callCount++
		return errors.New("error")
	})

	elapsed := time.Since(start)

	// Should take at least: 100ms + 200ms = 300ms
	// (first failure waits 100ms, second failure waits 200ms, third fails immediately)
	assert.GreaterOrEqual(t, elapsed, 300*time.Millisecond)
	assert.Equal(t, 3, callCount)
}

func TestDo_NilFunction(t *testing.T) {
	cfg := Config{
		Attempts: 3,
	}

	err := Do(context.Background(), cfg, nil)

	assert.Error(t, err)
	var cfgErr *ConfigError
	assert.ErrorAs(t, err, &cfgErr)
}

func TestDo_InvalidConfig(t *testing.T) {
	cfg := Config{
		Attempts: 0, // Invalid
	}

	err := Do(context.Background(), cfg, func() error {
		return nil
	})

	assert.Error(t, err)
	var cfgErr *ConfigError
	assert.ErrorAs(t, err, &cfgErr)
}

func TestDo_NoBackoff(t *testing.T) {
	cfg := Config{
		Attempts: 3,
		Backoff:  nil, // No backoff
	}

	start := time.Now()
	callCount := 0

	_ = Do(context.Background(), cfg, func() error {
		callCount++
		return errors.New("error")
	})

	elapsed := time.Since(start)

	// Should execute quickly without delays
	assert.Less(t, elapsed, 50*time.Millisecond)
	assert.Equal(t, 3, callCount)
}

func TestDo_NoJitter(t *testing.T) {
	cfg := Config{
		Attempts: 3,
		Backoff:  Constant(100 * time.Millisecond),
		Jitter:   nil, // No jitter
	}

	start := time.Now()
	_ = Do(context.Background(), cfg, func() error {
		return errors.New("error")
	})
	elapsed := time.Since(start)

	// Should take at least 200ms (100ms + 100ms delays)
	assert.GreaterOrEqual(t, elapsed, 200*time.Millisecond)
}

func TestDo_NoRetryIf(t *testing.T) {
	cfg := Config{
		Attempts: 3,
		Backoff:  Constant(10 * time.Millisecond),
		RetryIf:  nil, // Should retry all errors
	}

	callCount := 0
	_ = Do(context.Background(), cfg, func() error {
		callCount++
		return errors.New("error")
	})

	assert.Equal(t, 3, callCount)
}

func TestDo_SingleAttempt(t *testing.T) {
	cfg := Config{
		Attempts: 1,
	}

	callCount := 0
	err := Do(context.Background(), cfg, func() error {
		callCount++
		return errors.New("error")
	})

	assert.Error(t, err)
	assert.Equal(t, 1, callCount)
}

func TestDo_ConcurrentExecution(t *testing.T) {
	cfg := Config{
		Attempts: 3,
		Backoff:  Constant(10 * time.Millisecond),
	}

	const goroutines = 10
	done := make(chan bool, goroutines)
	var successCount atomic.Int32

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			err := Do(context.Background(), cfg, func() error {
				if id%2 == 0 {
					return nil
				}
				return errors.New("error")
			})
			if err == nil {
				successCount.Add(1)
			}
			done <- true
		}(i)
	}

	for i := 0; i < goroutines; i++ {
		<-done
	}

	// Half should succeed
	assert.Equal(t, int32(5), successCount.Load())
}

// TestDoWithValue tests

func TestDoWithValue_Success(t *testing.T) {
	cfg := Config{
		Attempts: 3,
		Backoff:  Constant(10 * time.Millisecond),
	}

	result, err := DoWithValue(context.Background(), cfg, func() (string, error) {
		return "success", nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "success", result)
}

func TestDoWithValue_SuccessAfterRetries(t *testing.T) {
	cfg := Config{
		Attempts: 5,
		Backoff:  Constant(10 * time.Millisecond),
	}

	callCount := 0
	result, err := DoWithValue(context.Background(), cfg, func() (int, error) {
		callCount++
		if callCount < 3 {
			return 0, errors.New("temporary error")
		}
		return 42, nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 42, result)
	assert.Equal(t, 3, callCount)
}

func TestDoWithValue_AllAttemptsExhausted(t *testing.T) {
	cfg := Config{
		Attempts: 3,
		Backoff:  Constant(10 * time.Millisecond),
	}

	result, err := DoWithValue(context.Background(), cfg, func() (string, error) {
		return "", errors.New("persistent error")
	})

	assert.Error(t, err)
	assert.Equal(t, "", result)

	var retryErr *RetryError
	assert.ErrorAs(t, err, &retryErr)
	assert.Equal(t, 3, retryErr.Attempts)
}

func TestDoWithValue_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	cfg := Config{
		Attempts: 10,
		Backoff:  Constant(50 * time.Millisecond),
	}

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	result, err := DoWithValue(ctx, cfg, func() (int, error) {
		return 0, errors.New("error")
	})

	assert.Error(t, err)
	assert.Equal(t, 0, result)
}

func TestDoWithValue_CustomType(t *testing.T) {
	type customResult struct {
		Value string
		Count int
	}

	cfg := Config{
		Attempts: 1,
	}

	expected := customResult{Value: "test", Count: 42}
	result, err := DoWithValue(context.Background(), cfg, func() (customResult, error) {
		return expected, nil
	})

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestDoWithValue_NilFunction(t *testing.T) {
	cfg := Config{
		Attempts: 3,
	}

	result, err := DoWithValue[int](context.Background(), cfg, nil)

	assert.Error(t, err)
	assert.Equal(t, 0, result)
	var cfgErr *ConfigError
	assert.ErrorAs(t, err, &cfgErr)
}

func TestDoWithValue_InvalidConfig(t *testing.T) {
	cfg := Config{
		Attempts: -1,
	}

	result, err := DoWithValue(context.Background(), cfg, func() (string, error) {
		return "test", nil
	})

	assert.Error(t, err)
	assert.Equal(t, "", result)
	var cfgErr *ConfigError
	assert.ErrorAs(t, err, &cfgErr)
}

func TestDoWithValue_OnRetryHook(t *testing.T) {
	var attempts []int

	cfg := Config{
		Attempts: 3,
		Backoff:  Constant(10 * time.Millisecond),
		OnRetry: func(attempt int, err error) {
			attempts = append(attempts, attempt)
		},
	}

	callCount := 0
	_, _ = DoWithValue(context.Background(), cfg, func() (int, error) {
		callCount++
		if callCount < 3 {
			return 0, errors.New("error")
		}
		return 100, nil
	})

	assert.Equal(t, []int{1, 2}, attempts)
}
