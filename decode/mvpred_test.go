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

func TestWriteBackBidiDirectPreservesChosenL0MV(t *testing.T) {
	mv4 := make([]syntax.MotionVector, 16)
	ref4 := make([]int8, 16)
	mb := &syntax.MBBidi{
		MBType:   syntax.BMBTypeDirect16x16,
		RefIdxL0: [4]int8{0},
		MVL0:     [4]syntax.MotionVector{{X: 3, Y: -2}},
	}
	writeBackBidiL0Context(mv4, ref4, 4, 0, 0, mb)
	for i := 0; i < 16; i++ {
		if mv4[i] != mb.MVL0[0] || ref4[i] != 0 {
			t.Fatalf("direct writeback idx=%d mv=%+v ref=%d, want %+v/ref0", i, mv4[i], ref4[i], mb.MVL0[0])
		}
	}
}

func TestWriteBackBidiB8x8UsesSubPartitionShapes(t *testing.T) {
	mv4 := make([]syntax.MotionVector, 16)
	ref4 := make([]int8, 16)
	for i := range ref4 {
		ref4[i] = -2
	}
	mb := &syntax.MBBidi{MBType: syntax.BMBTypeB8x8}
	mb.SubMBType[0] = 1 // L0_8x8: fills top-left 2x2
	mb.SubMBType[1] = 5 // L0_4x8: fills two vertical 1x2 partitions in top-right
	mb.SubMVL0[0] = syntax.MotionVector{X: 1, Y: 1}
	mb.SubMVL0[4] = syntax.MotionVector{X: 2, Y: 2}
	mb.SubMVL0[5] = syntax.MotionVector{X: 3, Y: 3}
	mb.RefIdxL0[0], mb.RefIdxL0[1] = 0, 1
	writeBackBidiL0Context(mv4, ref4, 4, 0, 0, mb)

	for _, idx := range []int{0, 1, 4, 5} {
		if mv4[idx] != mb.SubMVL0[0] || ref4[idx] != 0 {
			t.Fatalf("8x8 fill idx=%d mv=%+v ref=%d", idx, mv4[idx], ref4[idx])
		}
	}
	for _, idx := range []int{2, 6} {
		if mv4[idx] != mb.SubMVL0[4] || ref4[idx] != 1 {
			t.Fatalf("4x8 left fill idx=%d mv=%+v ref=%d", idx, mv4[idx], ref4[idx])
		}
	}
	for _, idx := range []int{3, 7} {
		if mv4[idx] != mb.SubMVL0[5] || ref4[idx] != 1 {
			t.Fatalf("4x8 right fill idx=%d mv=%+v ref=%d", idx, mv4[idx], ref4[idx])
		}
	}
}
