package decode

import "testing"

func TestCountNonZero16(t *testing.T) {
	var coeffs [16]int16
	if got := countNonZero16(coeffs); got != 0 {
		t.Fatalf("empty count got %d want 0", got)
	}
	coeffs[0] = 3
	coeffs[7] = -2
	coeffs[15] = 1
	if got := countNonZero16(coeffs); got != 3 {
		t.Fatalf("non-zero count got %d want 3", got)
	}
}
