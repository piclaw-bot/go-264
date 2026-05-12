package decode

import (
	"testing"

	"github.com/rcarmo/go-264/syntax"
)

func TestStoreCABACChromaDCDistributesAcrossBlocks(t *testing.T) {
	mb := &syntax.MBInter{}
	storeCABACChromaDC(mb, 0, [4]int16{11, 22, 33, 44})
	for blk, want := range []int16{11, 22, 33, 44} {
		if got := mb.CoeffsChroma[0][blk][0]; got != want {
			t.Fatalf("block %d DC got %d want %d", blk, got, want)
		}
	}
}

func TestStoreCABACChromaACPreservesDC(t *testing.T) {
	mb := &syntax.MBInter{}
	mb.CoeffsChroma[1][2][0] = 99
	var ac [16]int16
	for i := range ac {
		ac[i] = int16(i + 1)
	}
	storeCABACChromaAC(mb, 1, 2, ac)
	if got := mb.CoeffsChroma[1][2][0]; got != 99 {
		t.Fatalf("DC overwritten: got %d want 99", got)
	}
	for i := 1; i < 16; i++ {
		if got, want := mb.CoeffsChroma[1][2][i], ac[i]; got != want {
			t.Fatalf("AC[%d] got %d want %d", i, got, want)
		}
	}
}

func TestCABACTransform8x8CtxClampsToSpecRange(t *testing.T) {
	cases := map[int]int{-3: 0, -1: 0, 0: 0, 1: 1, 2: 2, 3: 2, 99: 2}
	for in, want := range cases {
		if got := cabacTransform8x8Ctx(in); got != want {
			t.Fatalf("cabacTransform8x8Ctx(%d) got %d want %d", in, got, want)
		}
	}
}

func TestStoreCABACChromaHelpersIgnoreInvalidInputs(t *testing.T) {
	storeCABACChromaDC(nil, 0, [4]int16{1, 2, 3, 4})
	storeCABACChromaAC(nil, 0, 0, [16]int16{})
	mb := &syntax.MBInter{}
	storeCABACChromaDC(mb, -1, [4]int16{1, 2, 3, 4})
	storeCABACChromaDC(mb, 2, [4]int16{1, 2, 3, 4})
	storeCABACChromaAC(mb, -1, 0, [16]int16{1})
	storeCABACChromaAC(mb, 2, 0, [16]int16{1})
	storeCABACChromaAC(mb, 0, -1, [16]int16{1})
	storeCABACChromaAC(mb, 0, 4, [16]int16{1})
	if mb.CoeffsChroma != [2][4][16]int16{} {
		t.Fatalf("invalid helper inputs mutated MB: %+v", mb.CoeffsChroma)
	}
}
