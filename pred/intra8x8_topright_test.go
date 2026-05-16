package pred

import "testing"

func TestPredIntra8x8TopRightAvailabilityIsExplicit(t *testing.T) {
	top := []uint8{50, 52, 54, 55, 45, 58, 54, 53, 53, 55, 57, 54, 54, 54, 55, 57}
	left := []uint8{51, 51, 59, 62, 59, 62, 56, 56}
	with := make([]uint8, 64)
	without := make([]uint8, 64)
	PredIntra8x8WithTopRight(with, Intra4x4DiagDownLeft, top, left, 52, true)
	PredIntra8x8WithTopRight(without, Intra4x4DiagDownLeft, top, left, 52, false)
	if with[7] == without[7] {
		t.Fatalf("expected explicit top-right availability to affect predictor when top[8] equals top[7]")
	}
	if with[7] != 54 {
		t.Fatalf("with top-right pixel 7 = %d, want filtered top-right value 54", with[7])
	}
}
