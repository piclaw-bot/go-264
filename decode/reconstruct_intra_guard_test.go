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
