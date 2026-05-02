package filter

import "testing"

func TestClip(t *testing.T) {
	if Clip3(0, 255, -5) != 0 { t.Fatal("clip low") }
	if Clip3(0, 255, 300) != 255 { t.Fatal("clip high") }
	if Clip3(0, 255, 128) != 128 { t.Fatal("clip mid") }
	if Clip1(-1) != 0 { t.Fatal("clip1 low") }
	if Clip1(256) != 255 { t.Fatal("clip1 high") }
}

func TestAlphaBetaTables(t *testing.T) {
	// Verify table sizes and monotonicity
	if len(alphaTable) != 52 { t.Fatal("alpha table size") }
	if len(betaTable) != 52 { t.Fatal("beta table size") }
	for i := 1; i < 52; i++ {
		if alphaTable[i] < alphaTable[i-1] {
			t.Errorf("alpha not monotonic at %d: %d < %d", i, alphaTable[i], alphaTable[i-1])
		}
	}
	// Alpha[0..15] should be 0
	if alphaTable[0] != 0 || alphaTable[15] != 0 {
		t.Error("alpha should be 0 for QP < 16")
	}
	if alphaTable[51] != 255 {
		t.Errorf("alpha[51]=%d want 255", alphaTable[51])
	}
}
