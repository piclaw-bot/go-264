package decode

import (
	"testing"

	"github.com/rcarmo/go-264/frame"
)

func TestPredictChroma8x8Plane(t *testing.T) {
	d := &Decoder{}
	f := frame.NewFrame(32, 32)
	f.SetPixelU(7, 7, 17)
	for i := 0; i < 8; i++ {
		f.SetPixelU(8+i, 7, uint8(20+i*3))
		f.SetPixelU(7, 8+i, uint8(40+i*2))
	}
	got := d.predictChroma8x8(f, 0, 1, 1, 3)

	topLeft := 17
	top := [8]int{20, 23, 26, 29, 32, 35, 38, 41}
	left := [8]int{40, 42, 44, 46, 48, 50, 52, 54}
	h, v := 0, 0
	for i := 0; i < 4; i++ {
		w := i + 1
		leftRef := topLeft
		topRef := topLeft
		if i < 3 {
			leftRef = top[2-i]
			topRef = left[2-i]
		}
		h += w * (top[4+i] - leftRef)
		v += w * (left[4+i] - topRef)
	}
	a := 16 * (left[7] + top[7])
	b := (17*h + 16) >> 5
	c := (17*v + 16) >> 5
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			want := clip8((a + b*(x-3) + c*(y-3) + 16) >> 5)
			if got[y*8+x] != want {
				t.Fatalf("plane[%d,%d] got %d want %d", x, y, got[y*8+x], want)
			}
		}
	}
}

func TestPredictChroma8x8PlaneUnavailableFallsBackNeutral(t *testing.T) {
	d := &Decoder{}
	f := frame.NewFrame(16, 16)
	got := d.predictChroma8x8(f, 0, 0, 0, 3)
	for i, v := range got {
		if v != 128 {
			t.Fatalf("plane fallback[%d] got %d want 128", i, v)
		}
	}
}

func TestPredictChroma8x8DCUsesFFmpegQuadrants(t *testing.T) {
	d := &Decoder{}
	f := frame.NewFrame(32, 32)
	for i := 0; i < 8; i++ {
		f.SetPixelU(8+i, 7, uint8(10+i))
		f.SetPixelU(7, 8+i, uint8(50+i*2))
	}

	got := d.predictChroma8x8(f, 0, 1, 1, 0)
	want := [4]uint8{
		uint8((50 + 52 + 54 + 56 + 10 + 11 + 12 + 13 + 4) >> 3),
		uint8((14 + 15 + 16 + 17 + 2) >> 2),
		uint8((58 + 60 + 62 + 64 + 2) >> 2),
		uint8((14 + 15 + 16 + 17 + 58 + 60 + 62 + 64 + 4) >> 3),
	}
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			quad := 0
			if x >= 4 {
				quad++
			}
			if y >= 4 {
				quad += 2
			}
			if got[y*8+x] != want[quad] {
				t.Fatalf("dc quadrant[%d,%d] got %d want %d", x, y, got[y*8+x], want[quad])
			}
		}
	}
}

func TestPredictChroma8x8DCEdgeUsesFFmpegHalves(t *testing.T) {
	d := &Decoder{}

	leftOnlyFrame := frame.NewFrame(32, 32)
	for i := 0; i < 8; i++ {
		leftOnlyFrame.SetPixelV(7, i, uint8(20+i*4))
	}
	leftOnly := d.predictChroma8x8(leftOnlyFrame, 1, 1, 0, 0)
	leftTop := uint8((20 + 24 + 28 + 32 + 2) >> 2)
	leftBottom := uint8((36 + 40 + 44 + 48 + 2) >> 2)
	for y := 0; y < 8; y++ {
		want := leftTop
		if y >= 4 {
			want = leftBottom
		}
		for x := 0; x < 8; x++ {
			if leftOnly[y*8+x] != want {
				t.Fatalf("left-only dc[%d,%d] got %d want %d", x, y, leftOnly[y*8+x], want)
			}
		}
	}

	topOnlyFrame := frame.NewFrame(32, 32)
	for i := 0; i < 8; i++ {
		topOnlyFrame.SetPixelU(i, 7, uint8(30+i*3))
	}
	topOnly := d.predictChroma8x8(topOnlyFrame, 0, 0, 1, 0)
	topLeft := uint8((30 + 33 + 36 + 39 + 2) >> 2)
	topRight := uint8((42 + 45 + 48 + 51 + 2) >> 2)
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			want := topLeft
			if x >= 4 {
				want = topRight
			}
			if topOnly[y*8+x] != want {
				t.Fatalf("top-only dc[%d,%d] got %d want %d", x, y, topOnly[y*8+x], want)
			}
		}
	}
}
