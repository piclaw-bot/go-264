package decode

import (
	"testing"

	"github.com/rcarmo/go-264/frame"
	"github.com/rcarmo/go-264/syntax"
)

func patternedPrediction16() [256]uint8 {
	var predicted [256]uint8
	for i := range predicted {
		predicted[i] = uint8((i*17 + 23) & 0xff)
	}
	return predicted
}

func assertPredicted16x16(t *testing.T, f *frame.Frame, predicted []uint8, mbX, mbY int) {
	t.Helper()
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			got := f.Y[(mbY*16+y)*f.StrideY+mbX*16+x]
			want := predicted[y*16+x]
			if got != want {
				t.Fatalf("pixel (%d,%d) got %d want %d", x, y, got, want)
			}
		}
	}
}

func TestWriteInterResidualZeroCBPCopiesPrediction4x4(t *testing.T) {
	var d Decoder
	f := frame.NewFrame(16, 16)
	predicted := patternedPrediction16()
	mb := &syntax.MBInter{CBP: 0}

	d.writeInterResidual(f, mb, predicted[:], 0, 0, 26)
	assertPredicted16x16(t, f, predicted[:], 0, 0)
}

func TestWriteInterResidualZeroCBPCopiesPrediction8x8(t *testing.T) {
	var d Decoder
	f := frame.NewFrame(16, 16)
	predicted := patternedPrediction16()
	mb := &syntax.MBInter{CBP: 0, Use8x8Transform: true}

	d.writeInterResidual(f, mb, predicted[:], 0, 0, 26)
	assertPredicted16x16(t, f, predicted[:], 0, 0)
}

func TestCoeff4x4NonZero(t *testing.T) {
	var zero [16]int16
	if coeff4x4NonZero(zero) {
		t.Fatal("zero block reported non-zero")
	}
	for i := range zero {
		block := zero
		block[i] = -1
		if !coeff4x4NonZero(block) {
			t.Fatalf("block with coeff %d set reported zero", i)
		}
	}
}

func TestWriteInterResidualPartialCBPCopiesUncoded4x4Blocks(t *testing.T) {
	var d Decoder
	f := frame.NewFrame(16, 16)
	predicted := patternedPrediction16()
	mb := &syntax.MBInter{CBP: 0x1}
	mb.Coeffs[0][0] = 64

	d.writeInterResidual(f, mb, predicted[:], 0, 0, 26)
	// Groups 1..3 are uncoded and must be exact copies of prediction.
	for y := 4; y < 16; y++ {
		for x := 0; x < 16; x++ {
			got := f.Y[y*f.StrideY+x]
			want := predicted[y*16+x]
			if got != want {
				t.Fatalf("uncoded lower pixel (%d,%d) got %d want %d", x, y, got, want)
			}
		}
	}
}

func TestWriteInterResidualCodedGroupCopiesZeroCoeff4x4Blocks(t *testing.T) {
	var d Decoder
	f := frame.NewFrame(16, 16)
	predicted := patternedPrediction16()
	mb := &syntax.MBInter{CBP: 0x1}
	mb.Coeffs[0][0] = 64

	d.writeInterResidual(f, mb, predicted[:], 0, 0, 26)
	// Block 0 has residual. Blocks 1..3 are in a coded CBP group but have all-zero
	// coefficients and must still be exact prediction copies.
	for _, blk := range []int{1, 2, 3} {
		bx := blk4x4X[blk]
		by := blk4x4Y[blk]
		for y := 0; y < 4; y++ {
			for x := 0; x < 4; x++ {
				got := f.Y[(by+y)*f.StrideY+bx+x]
				want := predicted[(by+y)*16+bx+x]
				if got != want {
					t.Fatalf("zero-coeff block %d pixel (%d,%d) got %d want %d", blk, bx+x, by+y, got, want)
				}
			}
		}
	}
}

func TestWriteInterResidualPartialCBPCopiesUncoded8x8Groups(t *testing.T) {
	var d Decoder
	f := frame.NewFrame(16, 16)
	predicted := patternedPrediction16()
	mb := &syntax.MBInter{CBP: 0x1, Use8x8Transform: true}
	mb.Coeffs[0][0] = 64

	d.writeInterResidual(f, mb, predicted[:], 0, 0, 26)
	// Groups 1..3 are uncoded and must be exact copies of prediction.
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			if x < 8 && y < 8 {
				continue
			}
			got := f.Y[y*f.StrideY+x]
			want := predicted[y*16+x]
			if got != want {
				t.Fatalf("uncoded 8x8 pixel (%d,%d) got %d want %d", x, y, got, want)
			}
		}
	}
}
