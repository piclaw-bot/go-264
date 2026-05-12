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
