package gpu

import (
	"testing"

	"github.com/rcarmo/go-264/me"
	"github.com/rcarmo/go-264/transform"
)

func TestBatchIDCT4x4MatchesTransform(t *testing.T) {
	const count = 32
	blocks := make([]int16, count*16)
	want := make([]int16, count*16)
	for i := range blocks {
		v := int16((i*37)%511 - 255)
		blocks[i] = v
		want[i] = v
	}
	BatchIDCT4x4(blocks, count)
	for i := 0; i < count; i++ {
		transform.IDCT4x4(want[i*16 : (i+1)*16])
	}
	for i := range blocks {
		if blocks[i] != want[i] {
			t.Fatalf("block value %d: got %d want %d", i, blocks[i], want[i])
		}
	}
}

func TestBatchSAD16x16MatchesSAD(t *testing.T) {
	stride := 64
	ref := make([]uint8, stride*64)
	cur := make([]uint8, stride*64)
	for i := range ref {
		ref[i] = uint8((i*13 + 7) & 0xff)
		cur[i] = uint8((i*5 + 29) & 0xff)
	}
	mvs := [][2]int{{0, 0}, {1, 0}, {0, 1}, {7, 3}, {15, 15}}
	results := make([]uint32, len(mvs))
	BatchSAD16x16(results, ref, cur, stride, mvs, len(mvs))
	for i, mv := range mvs {
		want := me.SAD16x16(cur, ref[mv[1]*stride+mv[0]:], stride, stride)
		if results[i] != want {
			t.Fatalf("mv[%d]=%v SAD=%d want %d", i, mv, results[i], want)
		}
	}
}
