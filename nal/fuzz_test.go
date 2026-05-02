package nal

import "testing"

// Fuzz the bitstream reader with random data
func FuzzReadUE(f *testing.F) {
	f.Add([]byte{0x80})       // UE=0
	f.Add([]byte{0x40})       // UE=1
	f.Add([]byte{0x20})       // UE=3
	f.Add([]byte{0x10})       // UE=7
	f.Add([]byte{0x00, 0x80}) // UE=15
	f.Add([]byte{0xFF})
	f.Add([]byte{0x00, 0x00, 0x03, 0x01}) // emulation prevention

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) == 0 {
			return
		}
		r := NewReader(data)
		// Should not panic on any input
		_ = r.ReadUE()
	})
}

func FuzzReadSE(f *testing.F) {
	f.Add([]byte{0x80})
	f.Add([]byte{0x40})
	f.Add([]byte{0x60})
	f.Add([]byte{0xFF, 0xFF})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) == 0 {
			return
		}
		r := NewReader(data)
		_ = r.ReadSE()
	})
}

func FuzzSplitNALUnits(f *testing.F) {
	f.Add([]byte{0x00, 0x00, 0x00, 0x01, 0x67, 0x42, 0x00})
	f.Add([]byte{0x00, 0x00, 0x01, 0x68, 0xCE})
	f.Add([]byte{0x00, 0x00, 0x00, 0x01, 0x65, 0x88})
	f.Add([]byte{}) // empty

	f.Fuzz(func(t *testing.T, data []byte) {
		// Should not panic
		units := SplitNALUnits(data)
		for _, u := range units {
			_ = u.TypeName()
			_ = u.IsSlice()
		}
	})
}

func FuzzParseSPS(f *testing.F) {
	f.Add([]byte{0x42, 0xc0, 0x1e, 0xd9, 0x01, 0x41, 0xfb, 0x01})
	f.Add([]byte{0x64, 0x00, 0x28, 0xac, 0xd1, 0x00, 0x78})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 3 {
			return
		}
		// Should not panic
		_, _ = ParseSPS(data)
	})
}

func FuzzParsePPS(f *testing.F) {
	f.Add([]byte{0xcb, 0x83, 0xcb, 0x20})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 1 {
			return
		}
		_, _ = ParsePPS(data)
	})
}
