package randutil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInt63n(t *testing.T) {
	t.Run("positive n", func(t *testing.T) {
		n := int64(100)
		for i := 0; i < 100; i++ {
			result := Int63n(n)
			assert.GreaterOrEqual(t, result, int64(0))
			assert.Less(t, result, n)
		}
	})

	t.Run("one", func(t *testing.T) {
		result := Int63n(1)
		assert.Equal(t, int64(0), result)
	})
}

func TestDuration(t *testing.T) {
	t.Run("positive duration", func(t *testing.T) {
		max := 1 * time.Second
		for i := 0; i < 100; i++ {
			result := Duration(max)
			assert.GreaterOrEqual(t, result, time.Duration(0))
			assert.Less(t, result, max)
		}
	})

	t.Run("zero duration", func(t *testing.T) {
		result := Duration(0)
		assert.Equal(t, time.Duration(0), result)
	})

	t.Run("negative duration", func(t *testing.T) {
		result := Duration(-1 * time.Second)
		assert.Equal(t, time.Duration(0), result)
	})
}
