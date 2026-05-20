package decode

import (
	"testing"

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
	c := newBMotionCache(4, 1)
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
