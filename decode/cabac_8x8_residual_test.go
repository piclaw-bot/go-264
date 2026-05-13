package decode

import "testing"

func TestLuma8x8ResidualSplitJoinPreservesQuadrants(t *testing.T) {
	var src [64]int16
	for i := range src {
		src[i] = int16(i + 1)
	}
	var coeffs [16][16]int16
	splitLuma8x8Residual(&coeffs, 2, src)

	checks := []struct {
		blk, coeff int
		want       int16
	}{
		{8, 0, 1},   // top-left quadrant row 0 col 0
		{8, 15, 28}, // top-left quadrant row 3 col 3 -> 8x8 row 3 col 3
		{9, 0, 5},   // top-right quadrant row 0 col 4
		{9, 15, 32}, // top-right quadrant row 3 col 7
		{10, 0, 33}, // bottom-left quadrant row 4 col 0
		{10, 15, 60},
		{11, 0, 37}, // bottom-right quadrant row 4 col 4
		{11, 15, 64},
	}
	for _, c := range checks {
		if got := coeffs[c.blk][c.coeff]; got != c.want {
			t.Fatalf("coeffs[%d][%d] got %d want %d", c.blk, c.coeff, got, c.want)
		}
	}
	if got := joinLuma8x8Residual(coeffs, 2); got != src {
		t.Fatalf("split/join changed 8x8 residual:\n got %v\nwant %v", got, src)
	}
}

func TestLuma8x8ResidualHelpersIgnoreInvalidInputs(t *testing.T) {
	var src [64]int16
	src[0] = 99
	splitLuma8x8Residual(nil, 0, src)
	var coeffs [16][16]int16
	splitLuma8x8Residual(&coeffs, -1, src)
	splitLuma8x8Residual(&coeffs, 4, src)
	if coeffs != [16][16]int16{} {
		t.Fatalf("invalid split mutated coeffs: %v", coeffs)
	}
	if got := joinLuma8x8Residual(coeffs, -1); got != [64]int16{} {
		t.Fatalf("invalid join got %v", got)
	}
}
