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
	if !usesL0(BMBTypeL016x16, 0) {
		t.Error("L016x16 should use L0")
	}
	if usesL1(BMBTypeL016x16, 0) {
		t.Error("L016x16 should not use L1")
	}

	// L1 16x16: uses L1 only
	if usesL0(BMBTypeL116x16, 0) {
		t.Error("L116x16 should not use L0")
	}
	if !usesL1(BMBTypeL116x16, 0) {
		t.Error("L116x16 should use L1")
	}

	// Bi 16x16: uses both
	if !usesL0(BMBTypeBi16x16, 0) {
		t.Error("Bi16x16 should use L0")
	}
	if !usesL1(BMBTypeBi16x16, 0) {
		t.Error("Bi16x16 should use L1")
	}
}

func TestUsesL0L1MatchesFFmpegBMBTypeTable(t *testing.T) {
	cases := []struct {
		mbType uint32
		part   int
		wantL0 bool
		wantL1 bool
	}{
		{4, 0, true, false}, {4, 1, true, false}, // L0/L0 16x8
		{6, 0, false, true}, {6, 1, false, true}, // L1/L1 16x8
		{8, 0, true, false}, {8, 1, false, true}, // L0/L1 16x8
		{10, 0, false, true}, {10, 1, true, false}, // L1/L0 16x8
		{12, 0, true, false}, {12, 1, true, true}, // L0/Bi 16x8
		{14, 0, false, true}, {14, 1, true, true}, // L1/Bi 16x8
		{16, 0, true, true}, {16, 1, true, false}, // Bi/L0 16x8
		{18, 0, true, true}, {18, 1, false, true}, // Bi/L1 16x8
		{20, 0, true, true}, {20, 1, true, true}, // Bi/Bi 16x8
	}
	for _, c := range cases {
		if got := usesL0(c.mbType, c.part); got != c.wantL0 {
			t.Fatalf("usesL0(type=%d, part=%d) got %v want %v", c.mbType, c.part, got, c.wantL0)
		}
		if got := usesL1(c.mbType, c.part); got != c.wantL1 {
			t.Fatalf("usesL1(type=%d, part=%d) got %v want %v", c.mbType, c.part, got, c.wantL1)
		}
	}
}

func TestUsesL0L1RejectsInvalidInputs(t *testing.T) {
	if usesL0(99, 0) || usesL1(99, 0) || usesL0(1, -1) || usesL1(1, 2) {
		t.Fatal("invalid B-slice list-use query returned true")
	}
}
