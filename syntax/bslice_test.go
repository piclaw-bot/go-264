package syntax

import "testing"

func TestBiPredBlend(t *testing.T) {
	l0 := []uint8{100, 200, 50, 0}
	l1 := []uint8{200, 100, 50, 255}
	out := make([]uint8, 4)
	BiPredBlend(out, l0, l1, 4)

	want := []uint8{150, 150, 50, 128}
	for i, v := range out {
		if v != want[i] {
			t.Errorf("out[%d]=%d want %d", i, v, want[i])
		}
	}
}

func TestUsesL0L1(t *testing.T) {
	// L0 16x16: uses L0 only
	if !usesL0(BMBTypeL016x16, 0) { t.Error("L016x16 should use L0") }
	if usesL1(BMBTypeL016x16, 0) { t.Error("L016x16 should not use L1") }

	// L1 16x16: uses L1 only
	if usesL0(BMBTypeL116x16, 0) { t.Error("L116x16 should not use L0") }
	if !usesL1(BMBTypeL116x16, 0) { t.Error("L116x16 should use L1") }

	// Bi 16x16: uses both
	if !usesL0(BMBTypeBi16x16, 0) { t.Error("Bi16x16 should use L0") }
	if !usesL1(BMBTypeBi16x16, 0) { t.Error("Bi16x16 should use L1") }
}
