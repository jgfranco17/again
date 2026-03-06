package again

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNoJitter(t *testing.T) {
	strategy := NoJitter()

	delays := []time.Duration{
		0,
		100 * time.Millisecond,
		1 * time.Second,
		5 * time.Second,
	}

	for _, delay := range delays {
		result := strategy.Apply(delay)
		assert.Equal(t, delay, result)
	}
}

func TestFullJitter(t *testing.T) {
	strategy := FullJitter()

	t.Run("positive delay", func(t *testing.T) {
		delay := 1 * time.Second

		for i := 0; i < 10; i++ {
			result := strategy.Apply(delay)
			assert.GreaterOrEqual(t, result, time.Duration(0))
			assert.Less(t, result, delay)
		}
	})

	t.Run("zero delay", func(t *testing.T) {
		result := strategy.Apply(0)
		assert.Equal(t, time.Duration(0), result)
	})

	t.Run("negative delay", func(t *testing.T) {
		result := strategy.Apply(-1 * time.Second)
		assert.Equal(t, time.Duration(0), result)
	})
}

func TestEqualJitter(t *testing.T) {
	strategy := EqualJitter()

	t.Run("positive delay", func(t *testing.T) {
		delay := 1 * time.Second
		half := delay / 2

		for i := 0; i < 10; i++ {
			result := strategy.Apply(delay)
			assert.GreaterOrEqual(t, result, half)
			assert.Less(t, result, delay)
		}
	})

	t.Run("zero delay", func(t *testing.T) {
		result := strategy.Apply(0)
		assert.Equal(t, time.Duration(0), result)
	})

	t.Run("negative delay", func(t *testing.T) {
		result := strategy.Apply(-1 * time.Second)
		assert.Equal(t, time.Duration(0), result)
	})
}
