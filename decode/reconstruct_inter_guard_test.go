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
