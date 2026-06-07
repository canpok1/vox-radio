package eval

import (
	"math/rand/v2"
	"time"
)

// Sample selects n items from pool using a reproducible shuffle seeded by seed.
// nameFunc extracts a display name for each item (used for logging by callers).
// If n >= len(pool), all items are returned in shuffled order.
func Sample[T any](pool []T, n int, seed int64, _ func(T) string) []T {
	if n >= len(pool) {
		n = len(pool)
	}

	src := rand.New(rand.NewPCG(uint64(seed), 0)) //nolint:gosec
	indices := src.Perm(len(pool))

	result := make([]T, n)
	for i := range n {
		result[i] = pool[indices[i]]
	}
	return result
}

// isoWeekSeed returns a seed derived from the ISO year and week number of t.
// All times within the same ISO week return the same seed.
func isoWeekSeed(t time.Time) int64 {
	year, week := t.ISOWeek()
	return int64(year)*100 + int64(week)
}

// DefaultSeed returns the ISO week-based seed for the current time.
func DefaultSeed() int64 {
	return isoWeekSeed(time.Now())
}
