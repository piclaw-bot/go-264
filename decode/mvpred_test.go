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

func TestFillMV4HandlesInvalidInputs(t *testing.T) {
	fillMV4(nil, nil, 0, 0, 0, 1, 1, syntax.MotionVector{X: 1}, 0)
	fillMV4(make([]syntax.MotionVector, 1), nil, 4, 0, 0, 2, 2, syntax.MotionVector{X: 1}, 0)
	fillMV4(nil, make([]int8, 1), 4, 0, 0, 2, 2, syntax.MotionVector{X: 1}, 0)
	fillMV4(make([]syntax.MotionVector, 1), make([]int8, 1), 4, -1, -1, 2, 2, syntax.MotionVector{X: 1}, 0)
}

func TestCABACMVDContextVectorClampsMagnitude(t *testing.T) {
	got := cabacMVDContextVector(syntax.MotionVector{X: -99, Y: 42})
	if got.X != 70 || got.Y != 42 {
		t.Fatalf("context vector got %+v want {X:70 Y:42}", got)
	}
	if returned := (syntax.MotionVector{X: -99, Y: 42}); returned.X != -99 {
		t.Fatalf("test invariant: reconstruction MVD should stay signed/full, got %+v", returned)
	}
}

func TestCABACMVContextHelpersHandleInvalidInputs(t *testing.T) {
	if got := cabacRefIdxCtx([]int8{1}, 0, 0, 0); got != 0 {
		t.Fatalf("cabacRefIdxCtx zero stride got %d want 0", got)
	}
	if got := cabacRefIdxCtx([]int8{1}, 4, -1, 0); got != 0 {
		t.Fatalf("cabacRefIdxCtx negative origin got %d want 0", got)
	}
	if got := cabacMVDAMVD([]syntax.MotionVector{{X: 9, Y: -7}}, 0, 0, 0, 0); got != 0 {
		t.Fatalf("cabacMVDAMVD zero stride got %d want 0", got)
	}
	fillMVD4(nil, 0, 0, 0, 1, 1, syntax.MotionVector{X: 1})
	fillMVD4(make([]syntax.MotionVector, 1), 4, -1, -1, 2, 2, syntax.MotionVector{X: 1})
}

func TestGetMV4HandlesInvalidInputs(t *testing.T) {
	if _, ref := getMV4(nil, []int8{0}, 4, 0, 0); ref != -2 {
		t.Fatalf("short mv cache ref=%d want -2", ref)
	}
	if _, ref := getMV4([]syntax.MotionVector{{X: 1}}, []int8{0}, 0, 0, 0); ref != -2 {
		t.Fatalf("zero stride ref=%d want -2", ref)
	}
	if _, ref := getMV4([]syntax.MotionVector{{X: 1}}, []int8{0}, 4, -1, 0); ref != -2 {
		t.Fatalf("negative x ref=%d want -2", ref)
	}
}
