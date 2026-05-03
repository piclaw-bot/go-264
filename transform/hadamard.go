package transform

// Hadamard4x4DC performs the 4×4 inverse Hadamard transform
// on the DC coefficients of an Intra_16x16 macroblock.
// ITU-T H.264 §8.5.6
//
// Input: 16 DC coefficients (one per 4×4 sub-block, in raster scan order).
// Output: 16 scaled DC values to be placed as block[0] of each sub-block.
func Hadamard4x4DC(dc []int16, qp int) {
	// 4×4 Hadamard inverse transform
	// Horizontal
	for i := 0; i < 4; i++ {
		r := dc[i*4 : i*4+4]
		a0 := r[0] + r[2]
		a1 := r[0] - r[2]
		a2 := r[1] - r[3]
		a3 := r[1] + r[3]
		r[0] = a0 + a3
		r[1] = a1 + a2
		r[2] = a1 - a2
		r[3] = a0 - a3
	}
	// Vertical
	for j := 0; j < 4; j++ {
		a0 := dc[j] + dc[8+j]
		a1 := dc[j] - dc[8+j]
		a2 := dc[4+j] - dc[12+j]
		a3 := dc[4+j] + dc[12+j]
		dc[j] = a0 + a3
		dc[4+j] = a1 + a2
		dc[8+j] = a1 - a2
		dc[12+j] = a0 - a3
	}

	// Dequantize DC coefficients (different from AC dequant)
	// For qp < 36: result = (dc * dequantV * 2) >> (6 - qp/6) ... simplified:
	// The DC dequant uses: levelScale = dequantV[qpMod6][0]
	// result = (coeff * levelScale) << (qp/6 - 2) for qp/6 >= 2
	// result = (coeff * levelScale + (1 << (1-qp/6))) >> (2 - qp/6) for qp/6 < 2
	qpDiv6 := qp / 6
	qpMod6 := qp % 6
	scale := int32(dequantV[qpMod6][0])

	if qpDiv6 >= 2 {
		shift := uint(qpDiv6 - 2)
		for i := range dc {
			dc[i] = int16((int32(dc[i]) * scale) << shift)
		}
	} else {
		shift := uint(2 - qpDiv6)
		round := int32(1 << (shift - 1))
		for i := range dc {
			dc[i] = int16((int32(dc[i])*scale + round) >> shift)
		}
	}
}

// Hadamard2x2DC performs the 2×2 inverse Hadamard transform
// for chroma DC coefficients.
func Hadamard2x2DC(dc []int16, qp int) {
	a := dc[0] + dc[1]
	b := dc[0] - dc[1]
	c := dc[2] + dc[3]
	d := dc[2] - dc[3]
	dc[0] = a + c
	dc[1] = b + d
	dc[2] = a - c
	dc[3] = b - d

	// Dequant
	qpMod6 := qp % 6
	qpDiv6 := qp / 6
	scale := int32(dequantV[qpMod6][0])

	if qpDiv6 >= 1 {
		shift := uint(qpDiv6 - 1)
		for i := range dc {
			dc[i] = int16((int32(dc[i]) * scale) << shift)
		}
	} else {
		for i := range dc {
			dc[i] = int16((int32(dc[i]) * scale) >> 1)
		}
	}
}
