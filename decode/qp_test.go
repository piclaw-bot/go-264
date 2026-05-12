package decode

import "testing"

func TestUpdateQPWrapsArbitraryDeltas(t *testing.T) {
	cases := []struct {
		current, delta int
		want           int
	}{
		{26, 0, 26},
		{51, 1, 0},
		{0, -1, 51},
		{50, 5, 3},
		{10, -70, 44},
	}
	for _, tc := range cases {
		if got := updateQP(tc.current, tc.delta); got != tc.want {
			t.Fatalf("updateQP(%d,%d) got %d want %d", tc.current, tc.delta, got, tc.want)
		}
	}
}
