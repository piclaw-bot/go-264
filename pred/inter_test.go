package pred

import "testing"

func TestInterPred16x16(t *testing.T) {
	// Create a simple 32x32 reference frame
	stride := 32
	ref := make([]uint8, stride*32)
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			ref[y*stride+x] = uint8((x + y) * 4)
		}
	}

	// Zero MV: should copy the top-left 16x16
	out := make([]uint8, 256)
	InterPred16x16(out, ref, stride, MotionVector{0, 0})
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			want := uint8((x + y) * 4)
			if out[y*16+x] != want {
				t.Fatalf("out[%d,%d]=%d want %d", y, x, out[y*16+x], want)
			}
		}
	}

	// MV = (4,4) in quarter-pixel = (1,1) full pixel
	InterPred16x16(out, ref, stride, MotionVector{4, 4})
	if out[0] != ref[1*stride+1] {
		t.Errorf("MV(4,4): out[0]=%d want %d", out[0], ref[1*stride+1])
	}
}

func TestSubpelFilter(t *testing.T) {
	// Test with constant input: should return same value
	samples := [6]uint8{100, 100, 100, 100, 100, 100}
	v := SubpelFilter6Tap(samples)
	if v != 100 {
		t.Errorf("constant input: got %d want 100", v)
	}

	// Test with edge: ramp up
	samples = [6]uint8{0, 50, 100, 150, 200, 250}
	v = SubpelFilter6Tap(samples)
	// (0 - 250 + 2000 + 3000 - 1000 + 250) / 32 = 4000/32 = 125
	t.Logf("ramp: %d (expect ~125)", v)
}
