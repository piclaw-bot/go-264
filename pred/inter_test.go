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

func TestInterPred16x16At(t *testing.T) {
	stride := 48
	ref := make([]uint8, stride*48)
	for y := 0; y < 48; y++ {
		for x := 0; x < 48; x++ {
			ref[y*stride+x] = uint8((x*3 + y*5) & 0xff)
		}
	}

	out := make([]uint8, 256)
	InterPred16x16At(out, ref, stride, 16, 12, MotionVector{4, 8}) // +1,+2 px
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			want := ref[(12+2+y)*stride+(16+1+x)]
			if out[y*16+x] != want {
				t.Fatalf("out[%d,%d]=%d want %d", y, x, out[y*16+x], want)
			}
		}
	}

	// Clipped edge path must remain scalar-correct.
	InterPred16x16At(out, ref, stride, -4, -3, MotionVector{0, 0})
	if out[0] != ref[0] {
		t.Fatalf("clipped top-left: got %d want %d", out[0], ref[0])
	}
}

func TestInterPred16x16AtFractionalInteriorMatchesReference(t *testing.T) {
	stride := 40
	ref := make([]uint8, stride*40)
	for y := 0; y < 40; y++ {
		for x := 0; x < 40; x++ {
			ref[y*stride+x] = uint8((x*11 + y*17) & 0xff)
		}
	}
	for _, mv := range []MotionVector{{1, 2}, {2, 0}, {0, 3}} {
		var fast, want [256]uint8
		InterPred16x16At(fast[:], ref, stride, 8, 9, mv)
		fx, fy := int(mv.X)&3, int(mv.Y)&3
		w00 := (4 - fx) * (4 - fy)
		w10 := fx * (4 - fy)
		w01 := (4 - fx) * fy
		w11 := fx * fy
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				sx, sy := 8+x, 9+y
				a := int(ref[sy*stride+sx])
				b := int(ref[sy*stride+sx+1])
				c := int(ref[(sy+1)*stride+sx])
				d := int(ref[(sy+1)*stride+sx+1])
				want[y*16+x] = uint8((a*w00 + b*w10 + c*w01 + d*w11 + 8) >> 4)
			}
		}
		if fast != want {
			t.Fatalf("fractional interior fast path mismatch for mv=%+v", mv)
		}
	}
}

func BenchmarkInterPred16x16At(b *testing.B) {
	stride := 1920
	ref := make([]uint8, stride*1080)
	out := make([]uint8, 256)
	for i := range ref {
		ref[i] = uint8(i)
	}
	mv := MotionVector{4, 4}
	b.ReportAllocs()
	b.SetBytes(256)
	for i := 0; i < b.N; i++ {
		InterPred16x16At(out, ref, stride, 256, 256, mv)
	}
}

func BenchmarkInterPred16x16AtFractionalHorizontal(b *testing.B) {
	stride := 1920
	ref := make([]uint8, stride*1080)
	out := make([]uint8, 256)
	for i := range ref {
		ref[i] = uint8(i)
	}
	mv := MotionVector{2, 0}
	b.ReportAllocs()
	b.SetBytes(256)
	for i := 0; i < b.N; i++ {
		InterPred16x16At(out, ref, stride, 256, 256, mv)
	}
}

func BenchmarkInterPred16x16AtFractionalVertical(b *testing.B) {
	stride := 1920
	ref := make([]uint8, stride*1080)
	out := make([]uint8, 256)
	for i := range ref {
		ref[i] = uint8(i)
	}
	mv := MotionVector{0, 2}
	b.ReportAllocs()
	b.SetBytes(256)
	for i := 0; i < b.N; i++ {
		InterPred16x16At(out, ref, stride, 256, 256, mv)
	}
}

func BenchmarkInterPred16x16AtFractional(b *testing.B) {
	stride := 1920
	ref := make([]uint8, stride*1080)
	out := make([]uint8, 256)
	for i := range ref {
		ref[i] = uint8(i)
	}
	mv := MotionVector{1, 2}
	b.ReportAllocs()
	b.SetBytes(256)
	for i := 0; i < b.N; i++ {
		InterPred16x16At(out, ref, stride, 256, 256, mv)
	}
}

func BenchmarkInterPred16x16AtFractionalClipped(b *testing.B) {
	stride := 1920
	ref := make([]uint8, stride*1080)
	out := make([]uint8, 256)
	for i := range ref {
		ref[i] = uint8(i)
	}
	mv := MotionVector{1, 2}
	b.ReportAllocs()
	b.SetBytes(256)
	for i := 0; i < b.N; i++ {
		InterPred16x16At(out, ref, stride, -1, -1, mv)
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
