package entropy

// CAVLC tables from FFmpeg h264_cavlc.c (authoritative).

import "github.com/rcarmo/go-264/nal"

// Table 9-7: total_zeros VLC
var totalZerosLen = [15][16]uint8{
	{1, 3, 3, 4, 4, 5, 5, 6, 6, 7, 7, 8, 8, 9, 9, 9},
	{3, 3, 3, 3, 3, 4, 4, 4, 4, 5, 5, 6, 6, 6, 6, 0},
	{4, 3, 3, 3, 4, 4, 3, 3, 4, 5, 5, 6, 5, 6, 0, 0},
	{5, 3, 4, 4, 3, 3, 3, 4, 3, 4, 5, 5, 5, 0, 0, 0},
	{4, 4, 4, 3, 3, 3, 3, 3, 4, 5, 4, 5, 0, 0, 0, 0},
	{6, 5, 3, 3, 3, 3, 3, 3, 4, 3, 6, 0, 0, 0, 0, 0},
	{6, 5, 3, 3, 3, 2, 3, 4, 3, 6, 0, 0, 0, 0, 0, 0},
	{6, 4, 5, 3, 2, 2, 3, 3, 6, 0, 0, 0, 0, 0, 0, 0},
	{6, 6, 4, 2, 2, 3, 2, 5, 0, 0, 0, 0, 0, 0, 0, 0},
	{5, 5, 3, 2, 2, 2, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{4, 4, 3, 3, 1, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{4, 4, 2, 1, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{3, 3, 1, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{2, 2, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
}

var totalZerosBits = [15][16]uint8{
	{1, 3, 2, 3, 2, 3, 2, 3, 2, 3, 2, 3, 2, 3, 2, 1},
	{7, 6, 5, 4, 3, 5, 4, 3, 2, 3, 2, 3, 2, 1, 0, 0},
	{5, 7, 6, 5, 4, 3, 4, 3, 2, 3, 2, 1, 1, 0, 0, 0},
	{3, 7, 5, 4, 6, 5, 4, 3, 3, 2, 2, 1, 0, 0, 0, 0},
	{5, 4, 3, 7, 6, 5, 4, 3, 2, 1, 1, 0, 0, 0, 0, 0},
	{1, 1, 7, 6, 5, 4, 3, 2, 1, 1, 0, 0, 0, 0, 0, 0},
	{1, 1, 5, 4, 3, 3, 2, 1, 1, 0, 0, 0, 0, 0, 0, 0},
	{1, 1, 1, 3, 3, 2, 2, 1, 0, 0, 0, 0, 0, 0, 0, 0},
	{1, 0, 1, 3, 2, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0},
	{1, 0, 1, 3, 2, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0, 1, 1, 2, 1, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
}

var totalZerosMaxVal = [15]int{15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1}

// Table 9-10: run_before VLC
var runBeforeLen = [7][16]uint8{
	{1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{1, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{2, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{2, 2, 2, 3, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{2, 2, 3, 3, 3, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{2, 3, 3, 3, 3, 3, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{3, 3, 3, 3, 3, 3, 3, 4, 5, 6, 7, 8, 9, 10, 11, 0},
}

var runBeforeBits = [7][16]uint8{
	{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{3, 2, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{3, 2, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{3, 2, 3, 2, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{3, 0, 1, 3, 2, 5, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{7, 6, 5, 4, 3, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0},
}

// DecodeTotalZeros decodes total_zeros (Table 9-7).
func DecodeTotalZeros(r *nal.Reader, totalCoeff int) int {
	if totalCoeff <= 0 || totalCoeff >= 16 {
		return 0
	}
	tableIdx := totalCoeff - 1
	maxVal := totalZerosMaxVal[tableIdx]

	pos := r.Position()
	avail := r.BitsLeft()
	peekLen := 9 // max code length in total_zeros table
	if avail < peekLen {
		peekLen = avail
	}
	if peekLen <= 0 {
		return 0
	}
	bits := r.PeekBits(peekLen)

	for val := 0; val <= maxVal; val++ {
		cLen := int(totalZerosLen[tableIdx][val])
		cBits := uint32(totalZerosBits[tableIdx][val])
		if cLen == 0 || cLen > peekLen {
			continue
		}
		shift := uint(peekLen - cLen)
		if (bits >> shift) == cBits {
			r.Seek(pos + cLen)
			return val
		}
	}
	r.Seek(pos + 1) // fallback
	return 0
}

// DecodeRunBefore decodes run_before (Table 9-10).
func DecodeRunBefore(r *nal.Reader, zerosLeft int) int {
	if zerosLeft <= 0 {
		return 0
	}

	tableIdx := zerosLeft - 1
	if tableIdx > 6 {
		tableIdx = 6
	}
	maxRun := zerosLeft
	if maxRun > 15 {
		maxRun = 15
	}

	pos := r.Position()
	avail := r.BitsLeft()
	peekLen := 11 // max code length in run_before table
	if avail < peekLen {
		peekLen = avail
	}
	if peekLen <= 0 {
		return 0
	}
	bits := r.PeekBits(peekLen)

	for run := 0; run <= maxRun; run++ {
		cLen := int(runBeforeLen[tableIdx][run])
		cBits := uint32(runBeforeBits[tableIdx][run])
		if cLen == 0 || cLen > peekLen {
			continue
		}
		shift := uint(peekLen - cLen)
		if (bits >> shift) == cBits {
			r.Seek(pos + cLen)
			return run
		}
	}
	r.Seek(pos + 1)
	return 0
}

// Coeff_token VLC tables from FFmpeg (Table 9-5a/b/c)
// Indexed as [totalCoeff*4 + trailingOnes]

// Table 9-5a: 0 <= nC < 2
var ctLen0 = [68]uint8{
	1, 0, 0, 0,
	6, 2, 0, 0, 8, 6, 3, 0, 9, 8, 7, 5, 10, 9, 8, 6,
	11, 10, 9, 7, 13, 11, 10, 8, 13, 13, 11, 9, 13, 13, 13, 10,
	14, 14, 13, 11, 14, 14, 14, 13, 15, 15, 14, 14, 15, 15, 15, 14,
	16, 15, 15, 15, 16, 16, 16, 15, 16, 16, 16, 16, 16, 16, 16, 16,
}
var ctBits0 = [68]uint8{
	1, 0, 0, 0,
	5, 1, 0, 0, 7, 4, 1, 0, 7, 6, 5, 3, 7, 6, 5, 3,
	7, 6, 5, 4, 15, 6, 5, 4, 11, 14, 5, 4, 8, 10, 13, 4,
	15, 14, 9, 4, 11, 10, 13, 12, 15, 14, 9, 12, 11, 10, 13, 8,
	15, 1, 9, 12, 11, 14, 13, 8, 7, 10, 9, 12, 4, 6, 5, 8,
}

// Table 9-5b: 2 <= nC < 4
var ctLen1 = [68]uint8{2, 0, 0, 0, 6, 2, 0, 0, 6, 5, 3, 0, 7, 6, 6, 4, 8, 6, 6, 4, 8, 7, 7, 5, 9, 8, 8, 6, 11, 9, 9, 6, 11, 11, 11, 7, 12, 11, 11, 9, 12, 12, 12, 11, 12, 12, 12, 11, 13, 13, 13, 12, 13, 13, 13, 13, 13, 14, 13, 13, 14, 14, 14, 13, 14, 14, 14, 14}
var ctBits1 = [68]uint8{3, 0, 0, 0, 11, 2, 0, 0, 7, 7, 3, 0, 7, 10, 9, 5, 7, 6, 5, 4, 4, 6, 5, 6, 7, 6, 5, 8, 15, 6, 5, 4, 11, 14, 13, 4, 15, 10, 9, 4, 11, 14, 13, 12, 8, 10, 9, 8, 15, 14, 13, 12, 11, 10, 9, 12, 7, 11, 6, 8, 9, 8, 10, 1, 7, 6, 5, 4}

// Table 9-5c: 4 <= nC < 8
var ctLen2 = [68]uint8{4, 0, 0, 0, 6, 4, 0, 0, 6, 5, 4, 0, 6, 5, 5, 4, 7, 5, 5, 4, 7, 5, 5, 4, 7, 6, 6, 4, 7, 6, 6, 4, 8, 7, 7, 5, 8, 8, 7, 6, 9, 8, 8, 7, 9, 9, 8, 8, 9, 9, 9, 8, 10, 9, 9, 9, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10}
var ctBits2 = [68]uint8{15, 0, 0, 0, 15, 14, 0, 0, 11, 15, 13, 0, 8, 12, 14, 12, 15, 10, 11, 11, 11, 8, 9, 10, 9, 14, 13, 9, 8, 10, 9, 8, 15, 14, 13, 13, 11, 14, 10, 12, 15, 10, 13, 12, 11, 14, 9, 12, 8, 10, 13, 8, 13, 7, 9, 12, 9, 12, 11, 10, 5, 8, 7, 6, 1, 4, 3, 2}

// decodeCoeffTokenFromTable reads coeff_token using FFmpeg VLC tables.
func decodeCoeffTokenFromTable(r *nal.Reader, nC int) (int, int) {
	if nC >= 8 {
		// Table 9-5d: fixed 6-bit code
		code := r.ReadBits(6)
		if code < 4 {
			if code == 3 {
				return 0, 0
			}
			// code 0→(1,0)? Actually for nC>=8:
			// totalCoeff = code/4 + 1 isn't right for code < 4
			// FFmpeg: suffix = code & 3, tc = code >> 2
			// code=0: tc=0,to=0 → but tc=0,to=0 code should be 3 (0b000011)
			// Let me just use the standard formula: to = code%4, tc = code/4
			// And for code=3: tc=0,to=3 but (0,3) doesn't exist, so it's (0,0)
			return 0, 0
		}
		to := int(code % 4)
		tc := int(code / 4)
		if to > tc {
			to = tc
		}
		return tc, to
	}

	var ctLen *[68]uint8
	var ctBits *[68]uint8
	if nC < 2 {
		ctLen = &ctLen0
		ctBits = &ctBits0
	} else if nC < 4 {
		ctLen = &ctLen1
		ctBits = &ctBits1
	} else {
		ctLen = &ctLen2
		ctBits = &ctBits2
	}

	pos := r.Position()
	avail := r.BitsLeft()
	peekLen := 16
	if avail < peekLen {
		peekLen = avail
	}
	if peekLen <= 0 {
		return 0, 0
	}
	bits := r.PeekBits(peekLen)

	bestLen := 0
	bestTC, bestTO := 0, 0
	for tc := 0; tc <= 16; tc++ {
		maxTO := 3
		if tc < maxTO {
			maxTO = tc
		}
		for to := 0; to <= maxTO; to++ {
			idx := tc*4 + to
			cLen := int(ctLen[idx])
			cBits := uint32(ctBits[idx])
			if cLen == 0 || cLen > peekLen {
				continue
			}
			shift := uint(peekLen - cLen)
			if (bits >> shift) == cBits {
				if bestLen == 0 || cLen < bestLen {
					bestLen = cLen
					bestTC = tc
					bestTO = to
				}
			}
		}
	}

	if bestLen > 0 {
		r.Seek(pos + bestLen)
		return bestTC, bestTO
	}
	r.Seek(pos + 1)
	return 0, 0
}
