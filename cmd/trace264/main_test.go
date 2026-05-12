package main

import (
	"testing"

	"github.com/rcarmo/go-264/syntax"
)

func TestTraceMVCacheHelpersHandleInvalidInputs(t *testing.T) {
	writeBackInter4x4(nil, nil, 0, 0, 0, nil)
	writeBackInter4x4(make([]syntax.MotionVector, 1), nil, 4, 0, 0, &syntax.MBInter{MBType: syntax.PMBTypeP16x16})
	writeBackIntra4x4(make([]int8, 1), 4, -1, -1)
	fillMV4(make([]syntax.MotionVector, 1), nil, 4, 0, 0, 2, 2, syntax.MotionVector{}, 0)
	if _, ref := getMV4(nil, []int8{0}, 4, 0, 0); ref != -2 {
		t.Fatalf("getMV4 with short mv cache ref=%d want -2", ref)
	}
}

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
