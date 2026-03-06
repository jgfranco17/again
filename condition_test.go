package again

import (
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAlways(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"nil error", nil},
		{"regular error", errors.New("error")},
		{"syscall error", syscall.ECONNREFUSED},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, Always(tt.err))
		})
	}
}

func TestNever(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"nil error", nil},
		{"regular error", errors.New("error")},
		{"syscall error", syscall.ECONNREFUSED},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.False(t, Never(tt.err))
		})
	}
}

func TestIfErrorIs(t *testing.T) {
	targetErr := errors.New("target error")
	otherErr := errors.New("other error")
	wrappedErr := errors.Join(targetErr, errors.New("additional"))
	condition := IfErrorIs(targetErr)

	t.Run("truthy condition", func(t *testing.T) {
		assert.True(t, condition(targetErr))
		assert.True(t, condition(wrappedErr))
	})

	t.Run("falsy condition", func(t *testing.T) {
		assert.False(t, condition(otherErr))
		assert.False(t, condition(nil))
	})
}

func TestIfErrorAs(t *testing.T) {
	var targetType *customError
	condition := IfErrorAs(&targetType)

	assert.True(t, condition(&customError{code: 42}))
	assert.False(t, condition(errors.New("regular error")))
	assert.False(t, condition(nil))
}

func TestTransientErrors(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		shouldRetry bool
	}{
		{
			name:        "nil error",
			err:         nil,
			shouldRetry: false,
		},
		{
			name:        "connection refused",
			err:         syscall.ECONNREFUSED,
			shouldRetry: true,
		},
		{
			name:        "connection reset",
			err:         syscall.ECONNRESET,
			shouldRetry: true,
		},
		{
			name:        "timeout",
			err:         syscall.ETIMEDOUT,
			shouldRetry: true,
		},
		{
			name:        "connection aborted",
			err:         syscall.ECONNABORTED,
			shouldRetry: true,
		},
		{
			name:        "host unreachable",
			err:         syscall.EHOSTUNREACH,
			shouldRetry: true,
		},
		{
			name:        "network unreachable",
			err:         syscall.ENETUNREACH,
			shouldRetry: true,
		},
		{
			name:        "deadline exceeded",
			err:         os.ErrDeadlineExceeded,
			shouldRetry: true,
		},
		{
			name: "net.OpError",
			err: &net.OpError{
				Op:  "dial",
				Net: "tcp",
				Err: errors.New("connection refused"),
			},
			shouldRetry: true,
		},
		{
			name:        "DNS temporary error",
			err:         &net.DNSError{IsTemporary: true},
			shouldRetry: true,
		},
		{
			name:        "DNS non-temporary error",
			err:         &net.DNSError{IsTemporary: false},
			shouldRetry: false,
		},
		{
			name:        "regular error",
			err:         errors.New("regular error"),
			shouldRetry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TransientErrors(tt.err)
			assert.Equal(t, tt.shouldRetry, result)
		})
	}
}

// temporaryError is a test helper that implements Temporary()
type temporaryError struct {
	temporary bool
}

func (e temporaryError) Error() string {
	return "temporary error"
}

func (e temporaryError) Temporary() bool {
	return e.temporary
}

func TestTransientErrors_TemporaryInterface(t *testing.T) {
	assert.True(t, TransientErrors(temporaryError{temporary: true}))
	assert.False(t, TransientErrors(temporaryError{temporary: false}))
}

// timeoutError is a test helper that implements Timeout()
type timeoutError struct {
	timeout bool
}

func (e timeoutError) Error() string {
	return "timeout error"
}

func (e timeoutError) Timeout() bool {
	return e.timeout
}

func TestTransientErrors_TimeoutInterface(t *testing.T) {
	assert.True(t, TransientErrors(timeoutError{timeout: true}))
	assert.False(t, TransientErrors(timeoutError{timeout: false}))
}

func TestOnlyIf(t *testing.T) {
	err1 := errors.New("error1")
	err2 := errors.New("error2")

	condition := OnlyIf(
		IfErrorIs(err1),
		Always,
	)

	assert.True(t, condition(err1))
	assert.False(t, condition(err2))
}

func TestOnlyIf_EmptyConditions(t *testing.T) {
	condition := OnlyIf()
	assert.True(t, condition(errors.New("any error")))
}

func TestAnyOf(t *testing.T) {
	err1 := errors.New("error1")
	err2 := errors.New("error2")
	err3 := errors.New("error3")

	condition := AnyOf(
		IfErrorIs(err1),
		IfErrorIs(err2),
	)

	assert.True(t, condition(err1))
	assert.True(t, condition(err2))
	assert.False(t, condition(err3))
}

func TestAnyOf_EmptyConditions(t *testing.T) {
	condition := AnyOf()
	assert.False(t, condition(errors.New("any error")))
}

func TestNot(t *testing.T) {
	err := errors.New("error")

	condition := Not(Always)
	assert.False(t, condition(err))

	condition = Not(Never)
	assert.True(t, condition(err))
}

func TestConditionComposition(t *testing.T) {
	netErr := &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")}
	regularErr := errors.New("regular error")

	// Retry if it's a transient error OR a specific error
	condition := AnyOf(
		TransientErrors,
		IfErrorIs(regularErr),
	)

	assert.True(t, condition(netErr))
	assert.True(t, condition(regularErr))
	assert.False(t, condition(errors.New("other error")))
}

type customError struct {
	code int
}

func (e *customError) Error() string {
	return fmt.Sprintf("custom error: %d", e.code)
}
