package again

import (
	"errors"
	"fmt"
)

// RetryError is returned when all retry attempts have been exhausted.
// It wraps the last error encountered and includes the total number of attempts made.
type RetryError struct {
	// Attempts is the total number of attempts made (including the initial attempt).
	Attempts int

	// LastErr is the error from the final attempt.
	LastErr error
}

// Error implements the error interface.
func (e *RetryError) Error() string {
	if e.LastErr != nil {
		return fmt.Sprintf("retry failed after %d attempts: %v", e.Attempts, e.LastErr)
	}
	return fmt.Sprintf("retry failed after %d attempts", e.Attempts)
}

// Unwrap returns the wrapped error, allowing errors.Is and errors.As to work.
func (e *RetryError) Unwrap() error {
	return e.LastErr
}

// IsRetryError checks if an error is a RetryError.
func IsRetryError(err error) bool {
	var retryErr *RetryError
	return errors.As(err, &retryErr)
}
