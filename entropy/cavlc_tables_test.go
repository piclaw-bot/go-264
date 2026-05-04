package entropy

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
