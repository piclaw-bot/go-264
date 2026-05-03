package slice

import (
	"github.com/rcarmo/go-264/entropy"
	"github.com/rcarmo/go-264/nal"
)

// Macroblock types for I-slices (ITU-T H.264 Table 7-11)
const (
	MBTypeINxN      = 0  // Intra_4x4 or Intra_8x8
	MBTypeI16x16_0  = 1  // Intra_16x16 (pred=0, CBP luma=0, CBP chroma=0)
	MBTypeI16x16_25 = 25 // last I_16x16 variant
	MBTypeIPCM      = 25 // I_PCM (raw samples)
)

// MBIntra describes a decoded intra macroblock.
type MBIntra struct {
	MBType        uint32
	IntraPredMode [16]int8 // 4x4 prediction modes (if MBTypeINxN)
	Intra16x16PredMode int8
	CodedBlockPattern  uint32 // CBP
	QPDelta            int32
	Coeffs             [16][16]int16 // 4x4 luma blocks in raster scan
	CoeffsChroma       [2][4][16]int16 // chroma blocks [U/V][4 blocks][16 coeffs]
}

// DecodeMBIntra decodes one intra macroblock from the bitstream.
// Returns the macroblock data needed for reconstruction.
func DecodeMBIntra(r *nal.Reader, sliceQP int32, ppsEntropy uint32, transform8x8 bool) *MBIntra {
	mb := &MBIntra{}

	mb.MBType = r.ReadUE()

	if mb.MBType == 0 {
		// I_NxN: decode prediction modes
		// transform_8x8_mode_flag=1 → I_8x8 (4 modes), else I_4x4 (16 modes)
		numModes := 16
		if transform8x8 { numModes = 4 }
		for i := 0; i < numModes; i++ {
			if r.ReadBool() { // prev_intra_pred_mode_flag
				mb.IntraPredMode[i] = -1 // use predicted mode
			} else {
				mb.IntraPredMode[i] = int8(r.ReadBits(3)) // rem_intra_pred_mode
			}
		}
	} else if mb.MBType >= 1 && mb.MBType <= 24 {
		// I_16x16: prediction mode, CBP coded in mb_type
		mb.Intra16x16PredMode = int8((mb.MBType - 1) % 4)
		cbpChroma := (mb.MBType - 1) / 4 % 3
		cbpLuma := uint32(0)
		if (mb.MBType-1)/12 > 0 {
			cbpLuma = 15
		}
		mb.CodedBlockPattern = cbpLuma | (cbpChroma << 4)
	}

	// Chroma intra pred mode (for NxN and 16x16)
	if mb.MBType != MBTypeIPCM {
		_ = r.ReadUE() // intra_chroma_pred_mode
	}

	// Coded block pattern (only for I_NxN, I_16x16 has it in mb_type)
	if mb.MBType == 0 {
		mb.CodedBlockPattern = decodeCBPIntra(r)
	}

	// QP delta
	if mb.CodedBlockPattern > 0 || (mb.MBType >= 1 && mb.MBType <= 24) {
		mb.QPDelta = r.ReadSE()
	}

	if ppsEntropy == 0 {
		if mb.MBType >= 1 && mb.MBType <= 24 {
			// I_16x16: decode DC block (16 DC coefficients) via CAVLC
			// nC = -1 signals DC block (uses special ChromaDC-like table)
			// For simplicity, use nC=0
			dcBlock, _ := entropy.DecodeCAVLCBlock(r, 0)
			for i := 0; i < 16; i++ {
				mb.Coeffs[i][0] = dcBlock[i]
			}
			// Decode AC coefficients for each 4x4 block if CBP indicates
			cbpLuma := mb.CodedBlockPattern & 0xF
			if cbpLuma != 0 {
				for blk := 0; blk < 16; blk++ {
					acBlock, _ := entropy.DecodeCAVLCBlock(r, 0)
					// AC coefficients go into positions 1..15
					for j := 1; j < 16; j++ {
						mb.Coeffs[blk][j] = acBlock[j-1]
					}
				}
			}
		} else if mb.MBType == 0 && mb.CodedBlockPattern > 0 {
			cbpLuma := mb.CodedBlockPattern & 0xF
			var nzCoeffs [16]int
			if transform8x8 {
				// I_8x8: each 8x8 block decoded as 4 sub-blocks
				for blk8 := 0; blk8 < 4; blk8++ {
					if cbpLuma&(1<<uint(blk8)) != 0 {
						// 4 sub-blocks per 8x8 block
						for sub := 0; sub < 4; sub++ {
							blk4 := blk8*4 + sub
							nC := computeNC4x4(blk4, nzCoeffs[:])
							block, tc := entropy.DecodeCAVLCBlock(r, nC)
							mb.Coeffs[blk4] = [16]int16(block)
							nzCoeffs[blk4] = tc
						}
					}
				}
			} else {
				// I_4x4: decode each 4x4 block independently
				for blk := 0; blk < 16; blk++ {
					group := blk / 4
					if cbpLuma&(1<<uint(group)) != 0 {
						nC := computeNC4x4(blk, nzCoeffs[:])
						block, tc := entropy.DecodeCAVLCBlock(r, nC)
						mb.Coeffs[blk] = [16]int16(block)
						nzCoeffs[blk] = tc
					}
				}
			}
		}
	}

	// Decode chroma residual if CBP indicates
	cbpChroma := mb.CodedBlockPattern >> 4
	if ppsEntropy == 0 && cbpChroma > 0 {
		// Chroma DC: 2×2 block for each Cb and Cr
		for comp := 0; comp < 2; comp++ {
			dcBlock, _ := entropy.DecodeCAVLCBlock(r, -1) // nC=-1 for chroma DC
			// Store DC values
			for i := 0; i < 4; i++ {
				mb.CoeffsChroma[comp][i][0] = dcBlock[i]
			}
		}
		// Chroma AC (if cbpChroma == 2)
		if cbpChroma == 2 {
			for comp := 0; comp < 2; comp++ {
				for blk := 0; blk < 4; blk++ {
					acBlock, _ := entropy.DecodeCAVLCBlock(r, 0)
					for j := 1; j < 16; j++ {
						mb.CoeffsChroma[comp][blk][j] = acBlock[j-1]
					}
				}
			}
		}
	}
	
	return mb
}

// decodeCBPIntra decodes coded_block_pattern for intra macroblocks.
// Uses Table 9-4 mapping from codeNum to CBP.
func decodeCBPIntra(r *nal.Reader) uint32 {
	codeNum := r.ReadUE()
	// Table 9-4: Intra CBP mapping (subset)
	cbpIntraTable := [48]uint32{
		47, 31, 15, 0, 23, 27, 29, 30, 7, 11, 13, 14, 39, 43, 45, 46,
		16, 3, 5, 10, 12, 19, 21, 26, 28, 35, 37, 42, 44, 1, 2, 4,
		8, 17, 18, 20, 24, 6, 9, 22, 25, 32, 33, 34, 36, 40, 38, 41,
	}
	if codeNum < 48 {
		return cbpIntraTable[codeNum]
	}
	return 0
}


// computeNC4x4 computes the nC context for a 4x4 block within a macroblock.
// Uses the totalCoeff of the left and top neighboring 4x4 blocks.
// Block layout within MB (raster scan):
//  0  1  4  5
//  2  3  6  7
//  8  9 12 13
// 10 11 14 15
func computeNC4x4(blkIdx int, nz []int) int {
	// Map block index to (x,y) within 4x4 grid
	// Using H.264 inverse raster scan
	x := (blkIdx % 4)
	y := (blkIdx / 4)
	
	nA, nB := -1, -1 // -1 = not available
	
	// Left neighbor
	if x > 0 {
		leftIdx := blkIdx - 1
		nA = nz[leftIdx]
	}
	
	// Top neighbor
	if y > 0 {
		topIdx := blkIdx - 4
		nB = nz[topIdx]
	}
	
	if nA >= 0 && nB >= 0 {
		return (nA + nB + 1) >> 1
	}
	if nA >= 0 { return nA }
	if nB >= 0 { return nB }
	return 0
}
