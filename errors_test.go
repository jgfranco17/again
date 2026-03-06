package again

import (
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRetryError_Error(t *testing.T) {
	tests := []struct {
		name     string
		retryErr RetryError
		want     string
	}{
		{
			name: "with underlying error",
			retryErr: RetryError{
				Attempts: 3,
				LastErr:  errors.New("connection refused"),
			},
			want: "retry failed after 3 attempts: connection refused",
		},
		{
			name: "without underlying error",
			retryErr: RetryError{
				Attempts: 5,
				LastErr:  nil,
			},
			want: "retry failed after 5 attempts",
		},
		{
			name: "single attempt",
			retryErr: RetryError{
				Attempts: 1,
				LastErr:  errors.New("timeout"),
			},
			want: "retry failed after 1 attempts: timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.retryErr.Error())
		})
	}
}

func TestRetryError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	retryErr := &RetryError{
		Attempts: 3,
		LastErr:  originalErr,
	}

	unwrapped := retryErr.Unwrap()
	assert.Equal(t, originalErr, unwrapped)
}

func TestRetryError_ErrorsIs(t *testing.T) {
	originalErr := errors.New("connection error")
	retryErr := &RetryError{
		Attempts: 3,
		LastErr:  originalErr,
	}

	assert.True(t, errors.Is(retryErr, originalErr))
	assert.False(t, errors.Is(retryErr, errors.New("different error")))
}

func TestRetryError_ErrorsAs(t *testing.T) {
	// Create a custom error type
	netErr := &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")}
	retryErr := &RetryError{
		Attempts: 3,
		LastErr:  netErr,
	}

	// Test that errors.As can find the wrapped net.OpError
	var target *net.OpError
	assert.True(t, errors.As(retryErr, &target))
	assert.Equal(t, "dial", target.Op)
}

func TestIsRetryError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "is retry error",
			err:  &RetryError{Attempts: 3},
			want: true,
		},
		{
			name: "wrapped retry error",
			err:  fmt.Errorf("wrapped: %w", &RetryError{Attempts: 3}),
			want: true,
		},
		{
			name: "not retry error",
			err:  errors.New("regular error"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsRetryError(tt.err))
		})
	}
}
