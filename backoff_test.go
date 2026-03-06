package again

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConstant(t *testing.T) {
	tests := []struct {
		name     string
		delay    time.Duration
		attempts []int
		want     []time.Duration
	}{
		{
			name:     "positive delay",
			delay:    100 * time.Millisecond,
			attempts: []int{1, 2, 3, 4, 5},
			want: []time.Duration{
				100 * time.Millisecond,
				100 * time.Millisecond,
				100 * time.Millisecond,
				100 * time.Millisecond,
				100 * time.Millisecond,
			},
		},
		{
			name:     "zero delay",
			delay:    0,
			attempts: []int{1, 2, 3},
			want:     []time.Duration{0, 0, 0},
		},
		{
			name:     "negative delay normalized to zero",
			delay:    -100 * time.Millisecond,
			attempts: []int{1, 2, 3},
			want:     []time.Duration{0, 0, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := Constant(tt.delay)
			for i, attempt := range tt.attempts {
				got := strategy.Next(attempt)
				assert.Equal(t, tt.want[i], got)
			}
		})
	}
}

func TestLinear(t *testing.T) {
	tests := []struct {
		name     string
		base     time.Duration
		attempts []int
		want     []time.Duration
	}{
		{
			name:     "linear growth",
			base:     100 * time.Millisecond,
			attempts: []int{1, 2, 3, 4, 5},
			want: []time.Duration{
				100 * time.Millisecond,
				200 * time.Millisecond,
				300 * time.Millisecond,
				400 * time.Millisecond,
				500 * time.Millisecond,
			},
		},
		{
			name:     "zero attempt",
			base:     100 * time.Millisecond,
			attempts: []int{0},
			want:     []time.Duration{0},
		},
		{
			name:     "negative attempt",
			base:     100 * time.Millisecond,
			attempts: []int{-1},
			want:     []time.Duration{0},
		},
		{
			name:     "negative base normalized",
			base:     -100 * time.Millisecond,
			attempts: []int{1, 2, 3},
			want:     []time.Duration{0, 0, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := Linear(tt.base)
			for i, attempt := range tt.attempts {
				got := strategy.Next(attempt)
				assert.Equal(t, tt.want[i], got)
			}
		})
	}
}

func TestExponential(t *testing.T) {
	tests := []struct {
		name     string
		base     time.Duration
		attempts []int
		want     []time.Duration
	}{
		{
			name:     "exponential growth",
			base:     100 * time.Millisecond,
			attempts: []int{1, 2, 3, 4, 5},
			want: []time.Duration{
				100 * time.Millisecond,  // 100 * 2^0
				200 * time.Millisecond,  // 100 * 2^1
				400 * time.Millisecond,  // 100 * 2^2
				800 * time.Millisecond,  // 100 * 2^3
				1600 * time.Millisecond, // 100 * 2^4
			},
		},
		{
			name:     "zero attempt",
			base:     100 * time.Millisecond,
			attempts: []int{0},
			want:     []time.Duration{0},
		},
		{
			name:     "negative attempt",
			base:     100 * time.Millisecond,
			attempts: []int{-1},
			want:     []time.Duration{0},
		},
		{
			name:     "capped at max",
			base:     1 * time.Second,
			attempts: []int{1, 10, 20, 30},
			want: []time.Duration{
				1 * time.Second,
				512 * time.Second, // 2^9
				time.Hour,         // capped
				time.Hour,         // capped
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := Exponential(tt.base)
			for i, attempt := range tt.attempts {
				got := strategy.Next(attempt)
				assert.Equal(t, tt.want[i], got)
			}
		})
	}
}

func TestExponentialWithMax(t *testing.T) {
	base := 100 * time.Millisecond
	max := 500 * time.Millisecond
	strategy := ExponentialWithMax(base, max)

	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{1, 100 * time.Millisecond},
		{2, 200 * time.Millisecond},
		{3, 400 * time.Millisecond},
		{4, 500 * time.Millisecond}, // capped
		{5, 500 * time.Millisecond}, // capped
	}

	for _, tt := range tests {
		t.Run("attempt", func(t *testing.T) {
			got := strategy.Next(tt.attempt)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExponential_OverflowProtection(t *testing.T) {
	strategy := Exponential(1 * time.Second)

	// Very high attempt number that would overflow without protection
	delay := strategy.Next(100)

	// Should be capped, not overflow
	assert.LessOrEqual(t, delay, time.Hour)
	assert.Greater(t, delay, time.Duration(0))
}
