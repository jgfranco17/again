package randutil

import (
	"math/rand/v2"
	"time"
)

// Int63n returns a non-negative pseudo-random 63-bit integer in [0,n).
// It is safe for concurrent use.
func Int63n(n int64) int64 {
	return rand.Int64N(n)
}

// Duration returns a random duration in the range [0, max).
// It is safe for concurrent use.
func Duration(max time.Duration) time.Duration {
	if max <= 0 {
		return 0
	}
	return time.Duration(Int63n(int64(max)))
}
