package syntax

// B-slice macroblock types and bidirectional prediction.
// ITU-T H.264 §7.3.5, Table 7-14

import "github.com/rcarmo/go-264/nal"

// B-slice macroblock types
const (
	BMBTypeDirect16x16 = 0
	BMBTypeL016x16     = 1
	BMBTypeL116x16     = 2
	BMBTypeBi16x16     = 3
	BMBTypeL016x8      = 4
	BMBTypeL016x8b     = 5 // second partition L1
	BMBTypeL116x8      = 6
	BMBTypeL116x8b     = 7
	BMBTypeBi16x8      = 8
	BMBTypeBi16x8b     = 9
	BMBTypeL08x16      = 10
	BMBTypeL18x16      = 11
	BMBTypeBi8x16      = 12
	BMBTypeB8x8        = 22
	BMBTypeIntra       = 23 // I_NxN in B-slice
)

// MBBidi describes a decoded B-slice macroblock.
type MBBidi struct {
	MBType  uint32
	RefIdxL0 [4]int8
	RefIdxL1 [4]int8
	MVL0     [4]MotionVector
	MVL1     [4]MotionVector
	SubMBType [4]uint32
	CBP      uint32
	QPDelta  int32
	Coeffs   [16][16]int16
}

// DecodeMBBidi decodes one macroblock from a B-slice.
func DecodeMBBidi(r *nal.Reader, sliceQP int32, numRefL0, numRefL1 uint32) *MBBidi {
	mb := &MBBidi{}
	mb.MBType = r.ReadUE()

	if mb.MBType >= BMBTypeIntra {
		return mb // intra MB in B-slice
	}

	if mb.MBType == BMBTypeDirect16x16 {
		// Direct mode: MV derived from co-located MB, no explicit MV
		return mb
	}

	// Determine list usage from mb_type
	numParts := 1
	if mb.MBType >= 4 && mb.MBType <= 21 {
		numParts = 2
	}
	if mb.MBType == BMBTypeB8x8 {
		numParts = 4
		for i := 0; i < 4; i++ {
			mb.SubMBType[i] = r.ReadUE()
		}
	}

	// Reference indices
	for i := 0; i < numParts; i++ {
		if usesL0(mb.MBType, i) && numRefL0 > 1 {
			mb.RefIdxL0[i] = int8(r.ReadUE())
		}
	}
	for i := 0; i < numParts; i++ {
		if usesL1(mb.MBType, i) && numRefL1 > 1 {
			mb.RefIdxL1[i] = int8(r.ReadUE())
		}
	}

	// Motion vectors
	for i := 0; i < numParts; i++ {
		if usesL0(mb.MBType, i) {
			mb.MVL0[i] = decodeMVD(r)
		}
	}
	for i := 0; i < numParts; i++ {
		if usesL1(mb.MBType, i) {
			mb.MVL1[i] = decodeMVD(r)
		}
	}

	// CBP + residual
	mb.CBP = decodeCBPInter(r)
	if mb.CBP > 0 {
		mb.QPDelta = r.ReadSE()
	}

	return mb
}

// usesL0 returns true if the partition uses list 0 (forward) prediction.
func usesL0(mbType uint32, partIdx int) bool {
	switch mbType {
	case BMBTypeL016x16, BMBTypeBi16x16:
		return true
	case BMBTypeL116x16:
		return false
	case BMBTypeL016x8, BMBTypeL016x8b:
		return partIdx == 0
	case BMBTypeBi16x8, BMBTypeBi16x8b, BMBTypeBi8x16:
		return true
	case BMBTypeL08x16:
		return true
	}
	if mbType >= 4 && mbType <= 21 {
		return true // simplified
	}
	return false
}

// usesL1 returns true if the partition uses list 1 (backward) prediction.
func usesL1(mbType uint32, partIdx int) bool {
	switch mbType {
	case BMBTypeL116x16, BMBTypeBi16x16:
		return true
	case BMBTypeL016x16:
		return false
	case BMBTypeL116x8, BMBTypeL116x8b:
		return partIdx == 0
	case BMBTypeBi16x8, BMBTypeBi16x8b, BMBTypeBi8x16:
		return true
	case BMBTypeL18x16:
		return true
	}
	if mbType >= 4 && mbType <= 21 {
		return true // simplified
	}
	return false
}

// BiPredBlend blends L0 and L1 predictions for bidirectional prediction.
// out[i] = (predL0[i] + predL1[i] + 1) >> 1
func BiPredBlend(out, predL0, predL1 []uint8, n int) {
	for i := 0; i < n; i++ {
		out[i] = uint8((uint16(predL0[i]) + uint16(predL1[i]) + 1) >> 1)
	}
}
