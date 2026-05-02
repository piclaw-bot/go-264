package nal

import "testing"

func TestReadBits(t *testing.T) {
	// 0xAB = 10101011
	r := NewReader([]byte{0xAB})
	if v := r.ReadBits(4); v != 0xA {
		t.Fatalf("got 0x%X want 0xA", v)
	}
	if v := r.ReadBits(4); v != 0xB {
		t.Fatalf("got 0x%X want 0xB", v)
	}
}

func TestReadUE(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want uint32
	}{
		{"0 (1)", []byte{0x80}, 0},       // 1... → 0
		{"1 (010)", []byte{0x40}, 1},     // 010... → 1
		{"2 (011)", []byte{0x60}, 2},     // 011... → 2
		{"3 (00100)", []byte{0x20}, 3},   // 00100... → 3
		{"4 (00101)", []byte{0x28}, 4},   // 00101... → 4
		{"5 (00110)", []byte{0x30}, 5},   // 00110... → 5
		{"6 (00111)", []byte{0x38}, 6},   // 00111... → 6
		{"7 (0001000)", []byte{0x10}, 7}, // 0001000... → 7
	}
	for _, tt := range tests {
		r := NewReader(tt.data)
		got := r.ReadUE()
		if got != tt.want {
			t.Errorf("ReadUE(%s) = %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestReadSE(t *testing.T) {
	tests := []struct {
		data []byte
		want int32
	}{
		{[]byte{0x80}, 0},  // UE=0 → SE=0
		{[]byte{0x40}, 1},  // UE=1 → SE=1
		{[]byte{0x60}, -1}, // UE=2 → SE=-1
		{[]byte{0x20}, 2},  // UE=3 → SE=2
		{[]byte{0x28}, -2}, // UE=4 → SE=-2
	}
	for _, tt := range tests {
		r := NewReader(tt.data)
		got := r.ReadSE()
		if got != tt.want {
			t.Errorf("ReadSE(%x) = %d, want %d", tt.data, got, tt.want)
		}
	}
}

func TestEmulationPrevention(t *testing.T) {
	// 0x00 0x00 0x03 0x01 → should read as 0x00 0x00 0x01
	data := []byte{0x00, 0x00, 0x03, 0x01}
	r := NewReader(data)
	b0 := r.ReadU8()
	b1 := r.ReadU8()
	b2 := r.ReadU8()
	if b0 != 0x00 || b1 != 0x00 || b2 != 0x01 {
		t.Fatalf("got %02x %02x %02x, want 00 00 01", b0, b1, b2)
	}
}
