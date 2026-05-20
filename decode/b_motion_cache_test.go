package decode

import (
	"testing"

	"github.com/rcarmo/go-264/frame"
	"github.com/rcarmo/go-264/syntax"
)

func TestBMotionCacheInitializesSplitLists(t *testing.T) {
	c := newBMotionCache(8, 2)
	if len(c.mv4(0)) != 64 || len(c.mv4(1)) != 64 || len(c.ref4(0)) != 64 || len(c.ref4(1)) != 64 {
		t.Fatalf("unexpected cache sizes: mv0=%d mv1=%d ref0=%d ref1=%d", len(c.mv4(0)), len(c.mv4(1)), len(c.ref4(0)), len(c.ref4(1)))
	}
	for list := 0; list < 2; list++ {
		for i, ref := range c.ref4(list) {
			if ref != -2 {
				t.Fatalf("list=%d idx=%d ref=%d want -2", list, i, ref)
			}
		}
	}
}

func TestBMotionCacheHelpersUseListState(t *testing.T) {
	c := newBMotionCache(8, 2)
	c.ref4(0)[0], c.mv4(0)[0] = 0, syntax.MotionVector{X: 1, Y: 2}
	c.ref4(1)[0], c.mv4(1)[0] = 1, syntax.MotionVector{X: 3, Y: 4}
	mv, ref := c.get(1, 0, 0)
	if ref != 1 || mv != (syntax.MotionVector{X: 3, Y: 4}) {
		t.Fatalf("list1 get mv=%+v ref=%d", mv, ref)
	}
	ctx := c.refIdxCtxs(0, 0)
	if ctx[0] != 0 {
		t.Fatalf("top-left ref ctx=%d want 0", ctx[0])
	}
	// Put left/top neighbours around MB (1,1) so skip/direct predictors exercise
	// cache-owned L0 state instead of raw pipeline arrays.
	leftIdx := 4*8 + 3
	topIdx := 3*8 + 4
	c.ref4(0)[leftIdx], c.mv4(0)[leftIdx] = 0, syntax.MotionVector{X: 5, Y: 6}
	c.ref4(0)[topIdx], c.mv4(0)[topIdx] = 0, syntax.MotionVector{X: 0, Y: 0}
	if pred := c.predictSkipL0(4, 4); pred != (syntax.MotionVector{}) {
		t.Fatalf("skip pred=%+v want zero because top neighbour is zero", pred)
	}
	if ref := c.directSpatialL0Ref(4, 4); ref != 0 {
		t.Fatalf("direct spatial ref=%d want 0", ref)
	}
}

func TestBMotionCacheSaveL0ToFrame(t *testing.T) {
	c := newBMotionCache(4, 1)
	c.mv4(0)[0] = syntax.MotionVector{X: 7, Y: -2}
	c.ref4(0)[0] = 1
	f := &frame.Frame{}
	c.saveL0ToFrame(f, []uint32{123})
	if f.MotionStride4 != 4 || len(f.MotionL0) != 16 || len(f.RefIdxL0) != 16 || len(f.MBType) != 1 {
		t.Fatalf("unexpected saved frame sizes/stride: stride=%d motion=%d ref=%d mbtype=%d", f.MotionStride4, len(f.MotionL0), len(f.RefIdxL0), len(f.MBType))
	}
	if f.MotionL0[0] != [2]int16{7, -2} || f.RefIdxL0[0] != 1 || f.MBType[0] != 123 {
		t.Fatalf("unexpected saved frame values: mv=%v ref=%d mbtype=%d", f.MotionL0[0], f.RefIdxL0[0], f.MBType[0])
	}
}

func TestBMotionCacheInitDirect16x16(t *testing.T) {
	c := newBMotionCache(4, 1)
	mb := &syntax.MBBidi{MBType: syntax.BMBTypeDirect16x16}
	mv0 := syntax.MotionVector{X: 1, Y: 2}
	mv1 := syntax.MotionVector{X: -3, Y: 4}
	c.initDirect16x16(mb, 1, mv0, 0, mv1)
	if mb.RefIdxL0[0] != 1 || mb.MVL0[0] != mv0 || mb.MVL1[0] != mv1 {
		t.Fatalf("direct init L0/L1 mismatch: %+v", mb)
	}
	for i, ref := range mb.RefIdxL1 {
		if ref != 0 {
			t.Fatalf("RefIdxL1[%d]=%d want 0", i, ref)
		}
	}
}

func TestBMotionCacheWriteBackInterL0(t *testing.T) {
	c := newBMotionCache(4, 1)
	mb := &syntax.MBInter{MBType: syntax.PMBTypeP16x16, RefIdx: [4]int8{0}, MV: [4]syntax.MotionVector{{X: 2, Y: -1}}}
	c.writeBackInterL0(0, 0, mb)
	for i := 0; i < 16; i++ {
		if c.mv4(0)[i] != mb.MV[0] || c.ref4(0)[i] != 0 {
			t.Fatalf("L0 idx=%d mv=%+v ref=%d", i, c.mv4(0)[i], c.ref4(0)[i])
		}
		if c.ref4(1)[i] != -2 {
			t.Fatalf("L1 idx=%d ref=%d should remain unavailable", i, c.ref4(1)[i])
		}
	}
}

func TestBMotionCacheWriteBackIntraMarksBothLists(t *testing.T) {
	c := newBMotionCache(4, 1)
	for list := 0; list < 2; list++ {
		for i := range c.ref4(list) {
			c.ref4(list)[i] = 0
		}
	}
	c.writeBackIntra(0, 0)
	for list := 0; list < 2; list++ {
		for i, ref := range c.ref4(list) {
			if ref != -1 {
				t.Fatalf("list=%d idx=%d ref=%d want -1", list, i, ref)
			}
		}
	}
}

func TestBMotionCacheWriteBackKeepsListsSeparate(t *testing.T) {
	c := newBMotionCache(4, 1)
	mb := &syntax.MBBidi{MBType: syntax.BMBTypeBi16x16, RefIdxL0: [4]int8{0}, RefIdxL1: [4]int8{1}}
	mb.MVL0[0] = syntax.MotionVector{X: 3, Y: 4}
	mb.MVL1[0] = syntax.MotionVector{X: -2, Y: 1}
	c.writeBackBidi(0, 0, mb)
	for i := 0; i < 16; i++ {
		if c.mv4(0)[i] != mb.MVL0[0] || c.ref4(0)[i] != 0 {
			t.Fatalf("L0 idx=%d mv=%+v ref=%d", i, c.mv4(0)[i], c.ref4(0)[i])
		}
		if c.mv4(1)[i] != mb.MVL1[0] || c.ref4(1)[i] != 1 {
			t.Fatalf("L1 idx=%d mv=%+v ref=%d", i, c.mv4(1)[i], c.ref4(1)[i])
		}
	}
}
