package decode

import (
	"testing"

	"github.com/rcarmo/go-264/syntax"
)

func TestWriteBackInter4x4HandlesInvalidInputs(t *testing.T) {
	writeBackInter4x4(nil, nil, 0, 0, 0, nil)
	writeBackInter4x4(make([]syntax.MotionVector, 1), nil, 4, 0, 0, &syntax.MBInter{MBType: syntax.PMBTypeP16x16})
	writeBackInter4x4(nil, make([]int8, 1), 4, 0, 0, &syntax.MBInter{MBType: syntax.PMBTypeP16x16})
	writeBackInter4x4(make([]syntax.MotionVector, 1), make([]int8, 1), 4, -1, -1, &syntax.MBInter{MBType: syntax.PMBTypeP16x16})
}

func TestWriteBackIntra4x4HandlesInvalidInputs(t *testing.T) {
	writeBackIntra4x4(nil, 0, 0, 0)
	writeBackIntra4x4(make([]int8, 1), 4, -1, -1)
}
