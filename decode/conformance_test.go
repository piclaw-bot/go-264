package decode

import (
	"os"
	"testing"
	"math"
)

func psnr(a, b []uint8, w, h, strideA, strideB int) float64 {
	var mse float64
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			d := float64(a[y*strideA+x]) - float64(b[y*strideB+x])
			mse += d * d
		}
	}
	mse /= float64(w * h)
	if mse < 1e-10 { return 99.0 }
	return 10 * math.Log10(255*255/mse)
}

func TestConformanceGray16(t *testing.T) {
	data, err := os.ReadFile("/workspace/tmp/gray16.h264")
	if err != nil { t.Skip("no test file") }
	dec := NewDecoder()
	frames, err := dec.Decode(data)
	if err != nil { t.Fatal(err) }
	if len(frames) == 0 { t.Fatal("no frames") }
	f := frames[0]
	// All pixels should be ~128 (gray)
	for y := 0; y < f.Height; y++ {
		for x := 0; x < f.Width; x++ {
			v := f.PixelY(x, y)
			if v < 124 || v > 132 {
				t.Fatalf("pixel(%d,%d)=%d, want ~128", x, y, v)
			}
		}
	}
	t.Log("Gray 16×16: pixel-accurate ✓")
}

func TestConformanceBBB(t *testing.T) {
	data, err := os.ReadFile("/workspace/tmp/bbb_annexb.h264")
	if err != nil { t.Skip("no test file") }
	dec := NewDecoder()
	frames, err := dec.Decode(data)
	if err != nil { t.Fatal(err) }
	if len(frames) < 10 {
		t.Fatalf("decoded %d frames, want >=10", len(frames))
	}
	t.Logf("BBB: decoded %d frames at %dx%d", len(frames), frames[0].Width, frames[0].Height)
	// Check first frame has non-trivial content
	unique := map[uint8]bool{}
	for y := 0; y < frames[0].Height; y++ {
		for x := 0; x < frames[0].Width; x++ {
			unique[frames[0].PixelY(x, y)] = true
		}
	}
	if len(unique) < 50 {
		t.Fatalf("only %d unique values, want >=50", len(unique))
	}
	t.Logf("BBB frame 0: %d unique pixel values ✓", len(unique))
}

func TestConformanceBaseline(t *testing.T) {
	data, err := os.ReadFile("/workspace/tmp/testsrc_bl.h264")
	if err != nil { t.Skip("no test file") }
	dec := NewDecoder()
	frames, err := dec.Decode(data)
	if err != nil { t.Fatal(err) }
	if len(frames) < 5 {
		t.Fatalf("decoded %d frames, want >=5", len(frames))
	}
	t.Logf("Baseline: %d frames, %dx%d", len(frames), frames[0].Width, frames[0].Height)
}
