package decode

import (
	"testing"

	"github.com/rcarmo/go-264/frame"
	"github.com/rcarmo/go-264/syntax"
)

func TestReconstructChromaInterHandlesNilInputs(t *testing.T) {
	d := &Decoder{}
	d.reconstructChromaInter(nil, nil, &syntax.MBInter{}, 0, 0, 26)
	d.reconstructChromaInter(frame.NewFrame(16, 16), nil, nil, 0, 0, 26)
}

func TestReconstructMBBidiHandlesInvalidInputs(t *testing.T) {
	var nilDecoder *Decoder
	nilDecoder.reconstructMBBidi(frame.NewFrame(16, 16), &syntax.MBBidi{}, 0, 0, 26)
	d := &Decoder{}
	d.reconstructMBBidi(nil, &syntax.MBBidi{}, 0, 0, 26)
	d.reconstructMBBidi(frame.NewFrame(16, 16), nil, 0, 0, 26)
	d.reconstructMBBidi(frame.NewFrame(16, 16), &syntax.MBBidi{}, -1, 0, 26)
	d.reconstructMBBidi(frame.NewFrame(16, 16), &syntax.MBBidi{}, 2, 0, 26)
}

func TestReconstructMBBidiAppliesZeroResidualPrediction(t *testing.T) {
	d := &Decoder{}
	f := frame.NewFrame(16, 16)
	for i := range f.Y {
		f.Y[i] = 90
	}
	d.reconstructMBBidi(f, &syntax.MBBidi{MBType: syntax.BMBTypeDirect16x16}, 0, 0, 26)
	for i, got := range f.Y {
		if got != 90 {
			t.Fatalf("pixel %d got %d want blended self prediction 90", i, got)
		}
	}
}

func TestInterResidualWritersHandleOutOfFrameInputs(t *testing.T) {
	d := &Decoder{}
	f := frame.NewFrame(16, 16)
	mb := &syntax.MBInter{MBType: syntax.PMBTypeP16x16}
	var predLuma [256]uint8
	var predChroma [64]uint8
	d.writeInterResidual(f, mb, predLuma[:], -1, 0, 26)
	d.writeInterResidual(f, mb, predLuma[:], 2, 0, 26)
	d.writeChromaInterResidual(f, mb, predChroma[:], 0, -1, 0, 26)
	d.writeChromaInterResidual(f, mb, predChroma[:], 0, 2, 0, 26)
}
