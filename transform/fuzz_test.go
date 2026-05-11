package transform

import "testing"

func FuzzIDCT4x4(f *testing.F) {
	f.Add(int16(256), int16(0), int16(0), int16(0))
	f.Add(int16(100), int16(50), int16(-30), int16(10))
	f.Add(int16(0), int16(0), int16(0), int16(0))
	f.Add(int16(32767), int16(-32768), int16(1), int16(-1))

	f.Fuzz(func(t *testing.T, c0, c1, c2, c3 int16) {
		block := [16]int16{c0, c1, c2, c3}
		// Should not panic
		IDCT4x4(block[:])
	})
}

func FuzzDCTRoundtrip(f *testing.F) {
	f.Add(int16(52), int16(55), int16(61), int16(66), int(10))
	f.Add(int16(0), int16(0), int16(0), int16(0), int(26))
	f.Add(int16(128), int16(128), int16(128), int16(128), int(0))

	f.Fuzz(func(t *testing.T, v0, v1, v2, v3 int16, qp int) {
		block := [16]int16{v0, v1, v2, v3}
		DCT4x4(block[:])
		Quant4x4(block[:], qp)
		Dequant4x4(block[:], qp)
		IDCT4x4(block[:])
		short := []int16{v0, v1, v2}
		Quant4x4(short, qp)
		Dequant4x4(short, qp)
		Dequant4x4AC(short, qp)
		// Should not panic — values may overflow but that's OK
	})
}
