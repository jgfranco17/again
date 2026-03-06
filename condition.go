package again

import (
	"errors"
	"net"
	"os"
	"syscall"
)

// Always is a RetryCondition that always retries, regardless of the error.
var Always RetryCondition = func(err error) bool {
	return true
}

// Never is a RetryCondition that never retries, regardless of the error.
func Never(err error) bool {
	return false
}

// IfErrorIs returns a RetryCondition that retries only if the error matches
// the target error using errors.Is.
func IfErrorIs(target error) RetryCondition {
	return func(err error) bool {
		return errors.Is(err, target)
	}
}

// IfErrorAs returns a RetryCondition that retries only if the error can be
// assigned to the target type using errors.As.
// The target must be a pointer to an error type.
func IfErrorAs(target any) RetryCondition {
	return func(err error) bool {
		return errors.As(err, target)
	}
}

// isTemporaryError checks if the error implements Temporary() bool.
func isTemporaryError(err error) bool {
	if te, ok := err.(interface{ Temporary() bool }); ok {
		return te.Temporary()
	}
	return false
}

// isTimeoutError checks if the error implements Timeout() bool.
func isTimeoutError(err error) bool {
	if te, ok := err.(interface{ Timeout() bool }); ok {
		return te.Timeout()
	}
	return false
}

// isNetworkError checks if the error is a network or DNS error.
func isNetworkError(err error) bool {
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return true
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return dnsErr.Temporary()
	}

	return false
}

// isRetryableSyscallError checks if the error is a retryable syscall error.
func isRetryableSyscallError(err error) bool {
	return errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ETIMEDOUT) ||
		errors.Is(err, syscall.ECONNABORTED) ||
		errors.Is(err, syscall.EHOSTUNREACH) ||
		errors.Is(err, syscall.ENETUNREACH)
}

// TransientErrors is a RetryCondition that retries on common transient errors
// such as network timeouts, connection refused, and temporary file system errors.
var TransientErrors RetryCondition = func(err error) bool {
	if err == nil {
		return false
	}

	return isTemporaryError(err) ||
		isTimeoutError(err) ||
		isNetworkError(err) ||
		isRetryableSyscallError(err) ||
		errors.Is(err, os.ErrDeadlineExceeded)
}

// OnlyIf creates a RetryCondition that combines multiple conditions with AND logic.
// All conditions must return true for the retry to occur.
func OnlyIf(conditions ...RetryCondition) RetryCondition {
	return func(err error) bool {
		for _, cond := range conditions {
			if !cond(err) {
				return false
			}
		}
		return true
	}
}

// AnyOf creates a RetryCondition that combines multiple conditions with OR logic.
// At least one condition must return true for the retry to occur.
func AnyOf(conditions ...RetryCondition) RetryCondition {
	return func(err error) bool {
		for _, cond := range conditions {
			if cond(err) {
				return true
			}
		}
		return false
	}
}

// Not negates a RetryCondition.
func Not(condition RetryCondition) RetryCondition {
	return func(err error) bool {
		return !condition(err)
	}
}
