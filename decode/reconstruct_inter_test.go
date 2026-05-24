package decode

import (
	"testing"

	"github.com/rcarmo/go-264/frame"
	"github.com/rcarmo/go-264/syntax"
)

func TestReconstructMBInterHandlesNilInputs(t *testing.T) {
	var d Decoder
	d.reconstructMBInter(nil, nil, 0, 0, 26)
	d.reconstructMBInter(frame.NewFrame(16, 16), nil, 0, 0, 26)
}

func TestReconstructMBInterNoReferenceFillsLumaAndLeavesNeutralChroma(t *testing.T) {
	var d Decoder
	f := frame.NewFrame(16, 16)
	mb := &syntax.MBInter{MBType: syntax.PMBTypeP16x16}
	d.reconstructMBInter(f, mb, 0, 0, 26)
	for i, got := range f.Y[:16*16] {
		if got != 128 {
			t.Fatalf("Y[%d] got %d want 128", i, got)
		}
	}
	for i, got := range f.U[:8*8] {
		if got != 128 {
			t.Fatalf("U[%d] got %d want neutral 128", i, got)
		}
	}
	for i, got := range f.V[:8*8] {
		if got != 128 {
			t.Fatalf("V[%d] got %d want neutral 128", i, got)
		}
	}
}

func TestDirect16HasSubMVsDetectsPer8x8DirectMotion(t *testing.T) {
	mb := &syntax.MBBidi{MBType: syntax.BMBTypeDirect16x16}
	mb.MVL0[0] = syntax.MotionVector{X: 1, Y: 2}
	mb.MVL1[0] = syntax.MotionVector{X: -1, Y: -2}
	for part := 0; part < 4; part++ {
		mb.RefIdxL0[part], mb.RefIdxL1[part] = 0, 0
		mb.SubMVL0[part*4] = mb.MVL0[0]
		mb.SubMVL1[part*4] = mb.MVL1[0]
	}
	if direct16HasSubMVs(mb) {
		t.Fatalf("uniform Direct16x16 motion should use the regular 16x16 path")
	}
	mb.SubMVL0[4] = syntax.MotionVector{X: 3, Y: 2}
	if !direct16HasSubMVs(mb) {
		t.Fatalf("per-8x8 Direct motion should use sub-block reconstruction")
	}
}
