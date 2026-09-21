package jobs

import "testing"

func TestResolveConcurrency(t *testing.T) {
	cases := []struct {
		in   int
		want int
	}{
		{2, 2},
		{1, 1},
		{3, 3},
		{4, 4},
		{0, DefaultConcurrency},
		{-1, DefaultConcurrency},
		{5, DefaultConcurrency},
		{100, DefaultConcurrency},
	}
	for _, tc := range cases {
		if got := ResolveConcurrency(tc.in); got != tc.want {
			t.Fatalf("ResolveConcurrency(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestResolveConcurrencyFromString(t *testing.T) {
	cases := []struct {
		raw      string
		defaultN int
		want     int
	}{
		{"", 2, 2},
		{"3", 2, 3},
		{"1", 2, 1},
		{"4", 2, 4},
		{"0", 2, 2},
		{"5", 2, 2},
		{"abc", 2, 2},
		{"", 0, 2}, // default also resolved
		{"2", 0, 2},
	}
	for _, tc := range cases {
		if got := ResolveConcurrencyFromString(tc.raw, tc.defaultN); got != tc.want {
			t.Fatalf("ResolveConcurrencyFromString(%q, %d) = %d, want %d", tc.raw, tc.defaultN, got, tc.want)
		}
	}
}
