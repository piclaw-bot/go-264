package gpu

import "testing"

func TestBatchIDCT4x4(t *testing.T) {
	// 4 blocks, each with DC-only coefficient
	blocks := make([]int16, 4*16)
	for i := 0; i < 4; i++ {
		blocks[i*16] = 256 // DC coefficient
	}

	BatchIDCT4x4(blocks, 4)

	// Each block should decode to all 4's (256/64=4)
	for i := 0; i < 4; i++ {
		for j := 0; j < 16; j++ {
			if blocks[i*16+j] != 4 {
				t.Fatalf("block[%d][%d]=%d want 4", i, j, blocks[i*16+j])
			}
		}
	}
	t.Log("BatchIDCT4x4: 4 blocks, all DC-only → correct ✓")
}

func TestBatchSAD16x16(t *testing.T) {
	stride := 32
	ref := make([]uint8, stride*32)
	cur := make([]uint8, stride*32)
	for i := range ref {
		ref[i] = 100
		cur[i] = 200
	}

	mvs := [][2]int{{0, 0}, {1, 0}, {0, 1}}
	results := make([]uint32, 3)
	BatchSAD16x16(results, ref, cur, stride, mvs, 3)

	// All differences = 100, 256 pixels → SAD = 25600
	if results[0] != 25600 {
		t.Fatalf("SAD[0]=%d want 25600", results[0])
	}
	t.Logf("BatchSAD: %v", results)
}

func BenchmarkBatchIDCT4x4(b *testing.B) {
	// 400 blocks = one 1080p frame's worth of 4x4 blocks (120 MBs × ~3.3 blocks)
	count := 400
	blocks := make([]int16, count*16)
	for i := range blocks {
		blocks[i] = int16(i % 500)
	}
	for i := 0; i < b.N; i++ {
		BatchIDCT4x4(blocks, count)
	}
}

func BenchmarkBatchSAD(b *testing.B) {
	stride := 64
	ref := make([]uint8, stride*64)
	cur := make([]uint8, stride*64)
	for i := range ref {
		ref[i] = uint8(i * 3)
		cur[i] = uint8(i * 7)
	}
	mvs := make([][2]int, 100)
	for i := range mvs {
		mvs[i] = [2]int{i % 16, i / 16}
	}
	results := make([]uint32, 100)
	for i := 0; i < b.N; i++ {
		BatchSAD16x16(results, ref, cur, stride, mvs, 100)
	}
}
