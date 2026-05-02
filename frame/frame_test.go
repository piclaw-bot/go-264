package frame

import "testing"

func TestNewFrame(t *testing.T) {
	f := NewFrame(320, 240)
	if f.Width != 320 || f.Height != 240 {
		t.Fatalf("size %dx%d want 320x240", f.Width, f.Height)
	}
	// Stride should be >= width and 16-aligned
	if f.StrideY < 320 || f.StrideY%16 != 0 {
		t.Fatalf("strideY=%d want >=320 and 16-aligned", f.StrideY)
	}
	if f.StrideC != f.StrideY/2 {
		t.Fatalf("strideC=%d want %d", f.StrideC, f.StrideY/2)
	}
	t.Logf("Frame 320x240: strideY=%d strideC=%d Y=%d U=%d V=%d bytes",
		f.StrideY, f.StrideC, len(f.Y), len(f.U), len(f.V))
}

func TestFramePixels(t *testing.T) {
	f := NewFrame(16, 16)
	f.SetPixelY(5, 3, 42)
	if v := f.PixelY(5, 3); v != 42 {
		t.Fatalf("pixel(5,3)=%d want 42", v)
	}
}

func TestBlock4x4(t *testing.T) {
	f := NewFrame(32, 32)
	// Fill MB (0,0) with known values
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			f.SetPixelY(x, y, uint8(y*16+x))
		}
	}
	// Extract block 0 (top-left 4x4)
	blk := f.Block4x4Y(0, 0, 0)
	if blk[0] != 0 || blk[3] != 3 || blk[4] != 16 {
		t.Fatalf("block4x4[0]: %v", blk[:8])
	}
}

func TestDPB(t *testing.T) {
	dpb := NewDPB(3)
	for i := 0; i < 5; i++ {
		f := NewFrame(16, 16)
		f.FrameNum = i
		f.IsRef = i%2 == 0
		dpb.Add(f)
	}
	if len(dpb.Frames) > 3 {
		t.Fatalf("DPB size %d want <= 3", len(dpb.Frames))
	}
	t.Logf("DPB has %d frames after adding 5", len(dpb.Frames))
}
