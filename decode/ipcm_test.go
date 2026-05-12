package decode

import (
	"testing"

	"github.com/rcarmo/go-264/frame"
	"github.com/rcarmo/go-264/syntax"
)

func TestReconstructIPCMWritesRawSamples(t *testing.T) {
	d := &Decoder{}
	f := frame.NewFrame(16, 16)
	mb := &syntax.MBIntra{MBType: syntax.MBTypeIPCM}
	for i := range mb.PCMY {
		mb.PCMY[i] = uint8(i)
	}
	for i := range mb.PCMCb {
		mb.PCMCb[i] = uint8(10 + i)
		mb.PCMCr[i] = uint8(100 + i)
	}
	d.reconstructMB(f, mb, 0, 0, 26, nil)
	if f.PixelY(0, 0) != 0 || f.PixelY(15, 15) != 255 || f.PixelU(0, 0) != 10 || f.PixelU(7, 7) != 73 || f.PixelV(0, 0) != 100 || f.PixelV(7, 7) != 163 {
		t.Fatalf("PCM reconstruction mismatch")
	}
}
