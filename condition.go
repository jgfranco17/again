package again

// Always is a RetryCondition that always retries, regardless of the error.
// Placeholder implementation - will be fully implemented in next iteration.
var Always RetryCondition = func(err error) bool {
	return true
}
