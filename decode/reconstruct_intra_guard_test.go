package decode

import (
	"testing"

	"github.com/rcarmo/go-264/frame"
	"github.com/rcarmo/go-264/syntax"
)

func TestReconstructMBHandlesNilInputs(t *testing.T) {
	d := &Decoder{}
	d.reconstructMB(nil, &syntax.MBIntra{MBType: syntax.MBTypeINxN}, 0, 0, 26, nil)
	d.reconstructMB(frame.NewFrame(16, 16), nil, 0, 0, 26, nil)
}

func TestIntraLumaReconstructorsHandleInvalidInputs(t *testing.T) {
	d := &Decoder{}
	f := frame.NewFrame(16, 16)
	mb := &syntax.MBIntra{MBType: syntax.MBTypeINxN}
	d.reconstruct16x16(nil, mb, 0, 0, 26)
	d.reconstruct16x16(f, nil, 0, 0, 26)
	d.reconstruct16x16(f, mb, -1, 0, 26)
	d.reconstruct16x16(f, mb, 2, 0, 26)
	d.reconstruct4x4(nil, mb, 0, 0, 26)
	d.reconstruct4x4(f, nil, 0, 0, 26)
	d.reconstruct4x4(f, mb, -1, 0, 26)
	d.reconstruct4x4(f, mb, 2, 0, 26)
	d.reconstruct8x8(nil, mb, 0, 0, 26)
	d.reconstruct8x8(f, nil, 0, 0, 26)
	d.reconstruct8x8(f, mb, -1, 0, 26)
	d.reconstruct8x8(f, mb, 2, 0, 26)
}

func TestReconstructIPCMHandlesOutOfFrameInputs(t *testing.T) {
	d := &Decoder{}
	f := frame.NewFrame(16, 16)
	mb := &syntax.MBIntra{MBType: syntax.MBTypeIPCM}
	d.reconstructIPCM(nil, mb, 0, 0)
	d.reconstructIPCM(f, nil, 0, 0)
	d.reconstructIPCM(f, mb, -1, 0)
	d.reconstructIPCM(f, mb, 2, 0)
}

func TestReconstructChromaIntraHandlesInvalidInputs(t *testing.T) {
	d := &Decoder{}
	d.reconstructChromaIntra(nil, &syntax.MBIntra{}, 0, 0, 26)
	d.reconstructChromaIntra(frame.NewFrame(16, 16), nil, 0, 0, 26)
	d.reconstructChromaIntra(frame.NewFrame(16, 16), &syntax.MBIntra{}, -1, 0, 26)
	d.reconstructChromaIntra(frame.NewFrame(16, 16), &syntax.MBIntra{}, 2, 0, 26)
}

func TestPredictChroma8x8HandlesInvalidFrames(t *testing.T) {
	d := &Decoder{}
	cases := []struct {
		name string
		f    *frame.Frame
		mbX  int
		mbY  int
	}{
		{"nil", nil, 1, 1},
		{"negative", frame.NewFrame(16, 16), -1, 0},
		{"outside", frame.NewFrame(16, 16), 2, 0},
	}
	for _, tc := range cases {
		got := d.predictChroma8x8(tc.f, 0, tc.mbX, tc.mbY, 0)
		for i, v := range got {
			if v != 128 {
				t.Fatalf("%s pred[%d] got %d want 128", tc.name, i, v)
			}
		}
	}
}
