package main

import "testing"

func TestUpdateQPMatchesDecoderModulo(t *testing.T) {
	cases := []struct {
		current, delta int
		want           int
	}{
		{26, 0, 26},
		{26, 1, 27},
		{26, -1, 25},
		{51, 1, 0},
		{0, -1, 51},
		{50, 5, 3},
	}
	for _, tc := range cases {
		if got := updateQP(tc.current, tc.delta); got != tc.want {
			t.Fatalf("updateQP(%d,%d) got %d want %d", tc.current, tc.delta, got, tc.want)
		}
	}
}
