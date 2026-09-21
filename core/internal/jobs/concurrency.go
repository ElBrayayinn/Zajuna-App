package jobs

import "strconv"

const (
	DefaultConcurrency = 2
	MinConcurrency     = 1
	MaxConcurrency     = 4
)

// ResolveConcurrency clamps n into [MinConcurrency, MaxConcurrency].
// Values outside the range fall back to DefaultConcurrency.
func ResolveConcurrency(n int) int {
	if n < MinConcurrency || n > MaxConcurrency {
		return DefaultConcurrency
	}
	return n
}

// ResolveConcurrencyFromString parses a flag/env value.
// Empty or invalid input yields defaultN (itself resolved).
func ResolveConcurrencyFromString(raw string, defaultN int) int {
	fallback := ResolveConcurrency(defaultN)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return ResolveConcurrency(n)
}
