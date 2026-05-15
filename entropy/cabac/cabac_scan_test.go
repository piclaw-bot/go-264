package cabac

import "testing"

func TestCABACScan8x8UsesDirectZigZagOrder(t *testing.T) {
	// CABAC luma 8x8 residual decode writes scan positions through the direct
	// H.264 8x8 zig-zag into Go's row-major coefficient matrix. FFmpeg later
	// stores coefficients in a transposed/permuted layout for its IDCT/qmul
	// tables, but blindly using that storage permutation here regresses the
	// decoder because Go's transform path expects row-major coefficients.
	want := [64]int{
		0, 1, 8, 16, 9, 2, 3, 10,
		17, 24, 32, 25, 18, 11, 4, 5,
		12, 19, 26, 33, 40, 48, 41, 34,
		27, 20, 13, 6, 7, 14, 21, 28,
		35, 42, 49, 56, 57, 50, 43, 36,
		29, 22, 15, 23, 30, 37, 44, 51,
		58, 59, 52, 45, 38, 31, 39, 46,
		53, 60, 61, 54, 47, 55, 62, 63,
	}
	if cabacScan8x8 != want {
		t.Fatalf("CABAC 8x8 scan changed\ngot  %v\nwant %v", cabacScan8x8, want)
	}
}
