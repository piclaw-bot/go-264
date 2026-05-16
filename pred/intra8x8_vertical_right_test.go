package pred

import "testing"

func TestPredIntra8x8VerticalRightMatchesFFmpeg(t *testing.T) {
	top := []uint8{85, 81, 79, 79, 78, 79, 80, 78, 80, 84, 80, 80, 83, 79, 76, 75}
	left := []uint8{88, 95, 100, 91, 92, 101, 95, 82}
	want := []uint8{
		86, 84, 81, 80, 79, 79, 79, 79,
		87, 85, 82, 80, 79, 79, 79, 79,
		91, 86, 84, 81, 80, 79, 79, 79,
		94, 87, 85, 82, 80, 79, 79, 79,
		96, 91, 86, 84, 81, 80, 79, 79,
		95, 94, 87, 85, 82, 80, 79, 79,
		95, 96, 91, 86, 84, 81, 80, 79,
		95, 95, 94, 87, 85, 82, 80, 79,
	}
	got := make([]uint8, 64)
	PredIntra8x8WithTopRight(got, Intra4x4VerticalRight, top, left, 87, true)
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("pixel %d = %d, want %d; got=%v", i, got[i], want[i], got)
		}
	}
}
