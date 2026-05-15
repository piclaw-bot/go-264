package decode

import "testing"

func TestCABACPredIntraModeUnavailableNeighbourUsesDC(t *testing.T) {
	cases := []struct {
		name      string
		left, top int8
		want      int8
	}{
		{name: "top unavailable", left: 1, top: -1, want: 2},
		{name: "left unavailable", left: -1, top: 4, want: 2},
		{name: "both unavailable", left: -1, top: -1, want: 2},
		{name: "both available takes min", left: 6, top: 4, want: 4},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := cabacPredIntraMode(tc.left, tc.top); got != tc.want {
				t.Fatalf("cabacPredIntraMode(%d, %d) = %d, want %d", tc.left, tc.top, got, tc.want)
			}
		})
	}
}
