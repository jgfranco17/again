package again_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jgfranco17/again"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	errTransient = errors.New("transient")
	errPermanent = errors.New("permanent")
)

func TestRetryClient_DoSuccess(t *testing.T) {
	client := again.NewRetryClient(baseClientConfig())

	calls := 0
	err := client.Do(context.Background(), func() error {
		calls++
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestRetryClient_DoRetriesAndSucceeds(t *testing.T) {
	client := again.NewRetryClient(baseClientConfig())

	calls := 0
	err := client.Do(context.Background(), func() error {
		calls++
		if calls < 3 {
			return errTransient
		}
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 3, calls)
}

func TestRetryClient_DoExhaustsAttempts(t *testing.T) {
	client := again.NewRetryClient(baseClientConfig())

	err := client.Do(context.Background(), func() error {
		return errTransient
	})

	require.Error(t, err)
	assert.True(t, again.IsRetryError(err))
}

func TestRetryClient_StatsSuccess(t *testing.T) {
	client := again.NewRetryClient(baseClientConfig())

	calls := 0
	err := client.Do(context.Background(), func() error {
		calls++
		if calls < 2 {
			return errTransient
		}
		return nil
	})

	require.NoError(t, err)
	stats := client.Stats()
	assert.Equal(t, 1, stats.TotalRuns)
	assert.Equal(t, 1, stats.Successes)
	assert.Equal(t, 0, stats.Failures)
	assert.Equal(t, 1, stats.TotalAttempts) // OnRetry called once for the first failure
}

func TestRetryClient_StatsFailure(t *testing.T) {
	client := again.NewRetryClient(baseClientConfig())

	client.Do(context.Background(), func() error { //nolint:errcheck
		return errTransient
	})

	stats := client.Stats()
	assert.Equal(t, 1, stats.TotalRuns)
	assert.Equal(t, 0, stats.Successes)
	assert.Equal(t, 1, stats.Failures)
	// 3 attempts, OnRetry called on attempts 1 and 2
	assert.Equal(t, 2, stats.TotalAttempts)
}

func TestRetryClient_StatsCumulative(t *testing.T) {
	client := again.NewRetryClient(baseClientConfig())

	// First run: success
	client.Do(context.Background(), func() error { return nil }) //nolint:errcheck

	// Second run: failure
	client.Do(context.Background(), func() error { return errTransient }) //nolint:errcheck

	stats := client.Stats()
	assert.Equal(t, 2, stats.TotalRuns)
	assert.Equal(t, 1, stats.Successes)
	assert.Equal(t, 1, stats.Failures)
}

func TestRetryClient_ResetStats(t *testing.T) {
	client := again.NewRetryClient(baseClientConfig())

	client.Do(context.Background(), func() error { return nil })          //nolint:errcheck
	client.Do(context.Background(), func() error { return errTransient }) //nolint:errcheck

	client.ResetStats()

	stats := client.Stats()
	assert.Equal(t, 0, stats.TotalRuns)
	assert.Equal(t, 0, stats.Successes)
	assert.Equal(t, 0, stats.Failures)
	assert.Equal(t, 0, stats.TotalAttempts)
}

func TestRetryClient_StatsDoNotInheritFromDerive(t *testing.T) {
	parent := again.NewRetryClient(baseClientConfig())
	parent.Do(context.Background(), func() error { return nil }) //nolint:errcheck

	child := parent.WithAttempts(5)

	assert.Equal(t, 1, parent.Stats().TotalRuns)
	assert.Equal(t, 0, child.Stats().TotalRuns)
}

func TestRetryClient_WithAttempts(t *testing.T) {
	client := again.NewRetryClient(baseClientConfig()).WithAttempts(1)

	calls := 0
	err := client.Do(context.Background(), func() error {
		calls++
		return errTransient
	})

	require.Error(t, err)
	assert.Equal(t, 1, calls)
}

func TestRetryClient_WithBackoffDoesNotMutateOriginal(t *testing.T) {
	original := again.NewRetryClient(baseClientConfig())
	derived := original.WithBackoff(again.Linear(10 * time.Millisecond))

	// Execute on both; verify they are independent
	err1 := original.Do(context.Background(), func() error { return nil })
	err2 := derived.Do(context.Background(), func() error { return nil })

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Equal(t, 1, original.Stats().TotalRuns)
	assert.Equal(t, 1, derived.Stats().TotalRuns)
}

func TestRetryClient_WithRetryIf(t *testing.T) {
	client := again.NewRetryClient(baseClientConfig()).
		WithRetryIf(again.IfErrorIs(errTransient))

	calls := 0
	err := client.Do(context.Background(), func() error {
		calls++
		return errPermanent // not in retry condition
	})

	require.Error(t, err)
	// Should stop after first attempt since errPermanent doesn't match
	assert.Equal(t, 1, calls)
}

func TestRetryClient_WithOnRetryPreservesUserHook(t *testing.T) {
	hookCalls := 0
	client := again.NewRetryClient(baseClientConfig()).
		WithOnRetry(func(attempt int, err error) {
			hookCalls++
		})

	client.Do(context.Background(), func() error { return errTransient }) //nolint:errcheck

	// User hook called on each retry (2 retries for 3 attempts)
	assert.Equal(t, 2, hookCalls)
	// Stats also tracked
	assert.Equal(t, 2, client.Stats().TotalAttempts)
}

func TestRetryClient_Config(t *testing.T) {
	cfg := baseClientConfig()
	client := again.NewRetryClient(cfg)

	got := client.Config()
	assert.Equal(t, cfg.Attempts, got.Attempts)
}

func TestRetryClient_ConfigUsableWithDoWithValue(t *testing.T) {
	client := again.NewRetryClient(baseClientConfig())

	result, err := again.DoWithValue(context.Background(), client.Config(), func() (int, error) {
		return 42, nil
	})

	require.NoError(t, err)
	assert.Equal(t, 42, result)
}

func baseClientConfig() again.Config {
	return again.Config{
		Attempts: 3,
		Backoff:  again.Constant(0),
	}
}
