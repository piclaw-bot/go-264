package cavlc

import (
	"testing"

	"github.com/rcarmo/go-264/nal"
)

func bitsToReader(bits uint32, n int) *nal.Reader {
	var data [4]byte
	v := bits << uint(32-n)
	data[0] = byte(v >> 24)
	data[1] = byte(v >> 16)
	data[2] = byte(v >> 8)
	data[3] = byte(v)
	return nal.NewReader(data[:])
}

func TestCoeffTokenTablesRoundtrip(t *testing.T) {
	for _, tbl := range []struct {
		name string
		lens *[68]uint8
		bits *[68]uint8
		nC   int
	}{
		{"nC0", &ctLen0, &ctBits0, 0},
		{"nC2", &ctLen1, &ctBits1, 2},
		{"nC4", &ctLen2, &ctBits2, 4},
		{"nC8", &ctLen3, &ctBits3, 8},
	} {
		for tc := 0; tc <= 16; tc++ {
			maxTO := 3
			if tc < maxTO {
				maxTO = tc
			}
			for to := 0; to <= maxTO; to++ {
				idx := tc*4 + to
				l := int(tbl.lens[idx])
				if l == 0 {
					continue
				}
				r := bitsToReader(uint32(tbl.bits[idx]), l)
				gotTC, gotTO := decodeCoeffTokenFromTable(r, tbl.nC)
				if gotTC != tc || gotTO != to || r.Position() != l {
					t.Fatalf("%s tc=%d to=%d code=%0*b: got tc=%d to=%d pos=%d want pos=%d",
						tbl.name, tc, to, l, tbl.bits[idx], gotTC, gotTO, r.Position(), l)
				}
			}
		}
	}
}

func TestCoeffTokenLookupMatchesScan(t *testing.T) {
	for _, tbl := range []struct {
		name string
		lens *[68]uint8
		bits *[68]uint8
		nC   int
	}{
		{"nC0", &ctLen0, &ctBits0, 0},
		{"nC2", &ctLen1, &ctBits1, 2},
		{"nC4", &ctLen2, &ctBits2, 4},
		{"nC8", &ctLen3, &ctBits3, 8},
	} {
		for tc := 0; tc <= 16; tc++ {
			maxTO := 3
			if tc < maxTO {
				maxTO = tc
			}
			for to := 0; to <= maxTO; to++ {
				idx := tc*4 + to
				l := int(tbl.lens[idx])
				if l == 0 {
					continue
				}
				for suffix := uint32(0); suffix < 8; suffix++ {
					bits := (uint32(tbl.bits[idx]) << 3) | suffix
					r := bitsToReader(bits, l+3)
					gotTC, gotTO, ok := decodeCoeffTokenLookup(r, tbl.nC)
					if !ok || gotTC != tc || gotTO != to || r.Position() != l {
						t.Fatalf("%s tc=%d to=%d suffix=%03b: got (%d,%d) ok=%v pos=%d want pos=%d",
							tbl.name, tc, to, suffix, gotTC, gotTO, ok, r.Position(), l)
					}
				}
			}
		}
	}
}

func BenchmarkDecodeCoeffTokenFromTable(b *testing.B) {
	data := []byte{0x80, 0x40, 0x20, 0x18, 0xff, 0x55, 0x33, 0x77}
	for i := 0; i < b.N; i++ {
		r := nal.NewReader(data)
		for j := 0; j < 16; j++ {
			_, _ = decodeCoeffTokenFromTable(r, j&7)
		}
	}
}

func TestTotalZerosTablesRoundtrip(t *testing.T) {
	for totalCoeff := 1; totalCoeff < 16; totalCoeff++ {
		tableIdx := totalCoeff - 1
		for totalZeros := 0; totalZeros <= totalZerosMaxVal[tableIdx]; totalZeros++ {
			l := int(totalZerosLen[tableIdx][totalZeros])
			if l == 0 {
				continue
			}
			r := bitsToReader(uint32(totalZerosBits[tableIdx][totalZeros]), l)
			got := DecodeTotalZeros(r, totalCoeff)
			if got != totalZeros || r.Position() != l {
				t.Fatalf("totalCoeff=%d totalZeros=%d code=%0*b: got %d pos=%d want pos=%d",
					totalCoeff, totalZeros, l, totalZerosBits[tableIdx][totalZeros], got, r.Position(), l)
			}
		}
	}
}

func TestChromaDCTablesRoundtrip(t *testing.T) {
	for tc := 0; tc <= 4; tc++ {
		maxTO := 3
		if tc < maxTO {
			maxTO = tc
		}
		for to := 0; to <= maxTO; to++ {
			idx := tc*4 + to
			l := int(chromaDCCoeffTokenLen[idx])
			if l == 0 {
				continue
			}
			r := bitsToReader(uint32(chromaDCCoeffTokenBits[idx]), l)
			gotTC, gotTO := decodeCoeffTokenChromaDCTable(r)
			if gotTC != tc || gotTO != to || r.Position() != l {
				t.Fatalf("chromaDC coeff tc=%d to=%d code=%0*b: got tc=%d to=%d pos=%d want pos=%d",
					tc, to, l, chromaDCCoeffTokenBits[idx], gotTC, gotTO, r.Position(), l)
			}
		}
	}
	for tc := 1; tc < 4; tc++ {
		idx := tc - 1
		for totalZeros := 0; totalZeros < 4; totalZeros++ {
			l := int(chromaDCTotalZerosLen[idx][totalZeros])
			if l == 0 {
				continue
			}
			r := bitsToReader(uint32(chromaDCTotalZerosBits[idx][totalZeros]), l)
			got := decodeChromaDCTotalZerosTable(r, tc)
			if got != totalZeros || r.Position() != l {
				t.Fatalf("chromaDC totalZeros tc=%d z=%d code=%0*b: got %d pos=%d want pos=%d",
					tc, totalZeros, l, chromaDCTotalZerosBits[idx][totalZeros], got, r.Position(), l)
			}
		}
	}
}

func TestRunBeforeTablesRoundtrip(t *testing.T) {
	for zerosLeft := 1; zerosLeft <= 15; zerosLeft++ {
		tableIdx := zerosLeft - 1
		if tableIdx > 6 {
			tableIdx = 6
		}
		for run := 0; run <= zerosLeft && run < 16; run++ {
			l := int(runBeforeLen[tableIdx][run])
			if l == 0 {
				continue
			}
			r := bitsToReader(uint32(runBeforeBits[tableIdx][run]), l)
			got := DecodeRunBefore(r, zerosLeft)
			if got != run || r.Position() != l {
				t.Fatalf("zerosLeft=%d run=%d code=%0*b: got %d pos=%d want pos=%d",
					zerosLeft, run, l, runBeforeBits[tableIdx][run], got, r.Position(), l)
			}
		}
	}
}

func TestVLCTableAdvanceSkipsEmulationPrevention(t *testing.T) {
	// Start three bits before an emulation-prevention byte. The RBSP bits seen by
	// DecodeRunBefore are 0001 (zerosLeft=7, run_before=7): three zero bits at
	// the end of byte 1, then byte 2 (0x03) must be skipped and the one bit is
	// read from byte 3. Table decoders must advance by reading bits, not by raw
	// Seek(pos+n), otherwise they can land inside the 0x03 EBSP byte.
	r := nal.NewReader([]byte{0x00, 0x00, 0x03, 0x80})
	r.Seek(13)
	got := DecodeRunBefore(r, 7)
	if got != 7 {
		t.Fatalf("DecodeRunBefore across emulation-prevention byte = %d, want 7", got)
	}
	if pos := r.Position(); pos != 25 {
		t.Fatalf("reader position after VLC across emulation-prevention byte = %d, want 25", pos)
	}
}
