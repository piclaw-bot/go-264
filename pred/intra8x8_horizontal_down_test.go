package pred

import "testing"

func TestPredIntra8x8HorizontalDownMatchesFFmpeg(t *testing.T) {
	top := []uint8{86, 102, 112, 117, 109, 93, 82, 78, 79, 83, 86, 85, 86, 90, 107, 107}
	left := []uint8{78, 84, 94, 84, 74, 80, 82, 80}
	want := []uint8{
		82, 84, 91, 101, 109, 112, 106, 95,
		83, 83, 82, 84, 91, 101, 109, 112,
		87, 85, 83, 83, 82, 84, 91, 101,
		87, 87, 87, 85, 83, 83, 82, 84,
		81, 84, 87, 87, 87, 85, 83, 83,
		79, 80, 81, 84, 87, 87, 87, 85,
		80, 79, 79, 80, 81, 84, 87, 87,
		81, 81, 80, 79, 79, 80, 81, 84,
	}
	got := make([]uint8, 64)
	PredIntra8x8(got, Intra4x4HorizontalDown, top, left, 83)
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("pixel %d = %d, want %d; got=%v", i, got[i], want[i], got)
		}
	}
}
