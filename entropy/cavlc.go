package entropy

// CAVLC (Context-Adaptive Variable-Length Coding) decoder.
// ITU-T H.264 §9.2
//
// CAVLC encodes 4×4 blocks of quantized transform coefficients.
// It codes: total coefficients, trailing ones, levels, total zeros, run_before.

import (
	"github.com/rcarmo/go-264/nal"
)

// Block4x4 holds decoded coefficients for a 4×4 block.
type Block4x4 [16]int16

// DecodeCAVLC decodes a 4×4 block of coefficients using CAVLC.
// nC is the predicted number of non-zero coefficients (context).
// Returns the decoded block in zig-zag order.
func DecodeCAVLC(r *nal.Reader, nC int) (Block4x4, int) {
	var block Block4x4

	// 1. Decode coeff_token (total_coeffs, trailing_ones)
	totalCoeff, trailingOnes := decodeCoeffToken(r, nC)
	if totalCoeff == 0 {
		return block, 0
	}

	// 2. Decode trailing ones signs (1 bit each, reverse order)
	signs := make([]int16, trailingOnes)
	for i := trailingOnes - 1; i >= 0; i-- {
		if r.ReadBit() == 1 {
			signs[i] = -1
		} else {
			signs[i] = 1
		}
	}

	// 3. Decode levels (non-trailing-one coefficients)
	levels := make([]int16, totalCoeff)
	levelIdx := totalCoeff - 1

	// Fill trailing ones
	for i := trailingOnes - 1; i >= 0; i-- {
		levels[levelIdx] = signs[i]
		levelIdx--
	}

	// Decode remaining levels
	suffixLength := 0
	if totalCoeff > 10 && trailingOnes < 3 {
		suffixLength = 1
	}

	for i := trailingOnes; i < totalCoeff; i++ {
		level := decodeLevelPrefix(r)

		if suffixLength > 0 {
			levelSuffix := r.ReadBits(suffixLength)
			level = (level << uint(suffixLength)) + int(levelSuffix)
		}

		if i == trailingOnes && trailingOnes < 3 {
			level += 2
		}

		if level%2 == 0 {
			levels[levelIdx] = int16(level/2 + 1)
		} else {
			levels[levelIdx] = int16(-(level + 1) / 2)
		}
		levelIdx--

		// Update suffix length
		if suffixLength == 0 {
			suffixLength = 1
		}
		absLevel := levels[levelIdx+1]
		if absLevel < 0 {
			absLevel = -absLevel
		}
		if int(absLevel) > (3<<uint(suffixLength-1)) && suffixLength < 6 {
			suffixLength++
		}
	}

	// 4. Decode total_zeros
	totalZeros := 0
	if totalCoeff < 16 {
		totalZeros = decodeTotalZeros(r, totalCoeff)
	}

	// 5. Decode run_before and place coefficients
	zerosLeft := totalZeros
	coeffIdx := totalCoeff - 1
	scanIdx := totalCoeff + totalZeros - 1

	for coeffIdx >= 0 {
		if zerosLeft > 0 && coeffIdx > 0 {
			run := decodeRunBefore(r, zerosLeft)
			block[scanIdx] = levels[coeffIdx]
			scanIdx -= run + 1
			zerosLeft -= run
		} else {
			block[scanIdx] = levels[coeffIdx]
			scanIdx--
		}
		coeffIdx--
	}

	return block, totalCoeff
}

// decodeCoeffToken decodes total_coeffs and trailing_ones from VLC tables.
// Simplified: uses lookup tables based on nC range.
func decodeCoeffToken(r *nal.Reader, nC int) (totalCoeff, trailingOnes int) {
	// Table selection based on nC
	if nC < 2 {
		return decodeCoeffTokenTable0(r)
	} else if nC < 4 {
		return decodeCoeffTokenTable1(r)
	} else if nC < 8 {
		return decodeCoeffTokenTable2(r)
	}
	// nC >= 8: fixed 6-bit code
	code := r.ReadBits(6)
	trailingOnes = int(code & 3)
	totalCoeff = int(code>>2) + 1
	if totalCoeff > 16 {
		totalCoeff = 16
	}
	return
}

// decodeLevelPrefix reads the level prefix (unary coded).
func decodeLevelPrefix(r *nal.Reader) int {
	zeros := 0
	for r.ReadBit() == 0 {
		zeros++
		if zeros > 15 {
			return zeros
		}
	}
	return zeros
}

// decodeTotalZeros is a simplified decoder (placeholder — full tables needed).
func decodeTotalZeros(r *nal.Reader, totalCoeff int) int {
	// Simplified: read as truncated unary
	// Real implementation needs VLC tables from spec Table 9-7/9-8
	if totalCoeff >= 16 {
		return 0
	}
	zeros := 0
	maxZeros := 16 - totalCoeff
	for zeros < maxZeros && r.ReadBit() == 0 {
		zeros++
	}
	return zeros
}

// decodeRunBefore decodes run_before value.
func decodeRunBefore(r *nal.Reader, zerosLeft int) int {
	if zerosLeft <= 0 {
		return 0
	}
	// Simplified: read as truncated unary
	// Real implementation needs VLC table from spec Table 9-10
	if zerosLeft == 1 {
		return int(r.ReadBit())
	}
	run := 0
	for run < zerosLeft && r.ReadBit() == 0 {
		run++
	}
	return run
}

// Coeff token VLC table 0 (nC = 0..1) — simplified binary decision tree.
func decodeCoeffTokenTable0(r *nal.Reader) (int, int) {
	// This is a placeholder. Full table has ~60 entries.
	// Most common: 0 coeffs = single 1 bit
	if r.ReadBit() == 1 {
		return 0, 0 // 1 → (0,0)
	}
	if r.ReadBit() == 0 {
		if r.ReadBit() == 1 {
			return 1, 1 // 001 → (1,1)
		}
		// 000...
		if r.ReadBit() == 0 {
			if r.ReadBit() == 1 {
				return 2, 2 // 00001... simplified
			}
			return 1, 0 // 00001 → (1,0) simplified
		}
		return 2, 1 // 0001 → (2,1) simplified
	}
	// 01
	return 1, 1 // 01 → (1,1)
}

func decodeCoeffTokenTable1(r *nal.Reader) (int, int) {
	// Placeholder for nC=2..3
	if r.ReadBit() == 1 {
		if r.ReadBit() == 1 {
			return 0, 0
		}
		return 1, 1
	}
	return 2, 2
}

func decodeCoeffTokenTable2(r *nal.Reader) (int, int) {
	// Placeholder for nC=4..7
	code := r.ReadBits(4)
	if code == 0xF {
		return 0, 0
	}
	return int(code>>2) + 1, int(code & 3)
}
