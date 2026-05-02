package transform

// H.264 4×4 integer transform and quantization.
// ITU-T H.264 §8.5.12
//
// The H.264 "DCT" is not a true DCT — it's a scaled integer transform
// that avoids floating-point entirely. The core transform matrix is:
//
//   Cf = [ 1  1  1  1 ]
//        [ 2  1 -1 -2 ]
//        [ 1 -1 -1  1 ]
//        [ 1 -2  2 -1 ]
//
// Forward: Y = Cf * X * Cf^T (with post-scaling)
// Inverse: X = Ci^T * Y * Ci (with pre-scaling)
//
// All arithmetic is 16-bit integer.

// IDCT4x4 performs the inverse 4×4 integer transform (in-place).
// Input: dequantized coefficients in block[0:16].
// Output: residual pixel values.
func IDCT4x4(block []int16) {
	if HasAVX2 && len(block) >= 16 {
		IDCT4x4_AVX2(&block[0])
		return
	}
	if HasNEON && len(block) >= 16 {
		IDCT4x4_NEON(&block[0])
		return
	}
	IDCT4x4Scalar(block)
}

// Output: transform coefficients (before quantization).
func DCT4x4(block []int16) {
	if HasAVX2 && len(block) >= 16 {
		DCT4x4_AVX2(&block[0])
		return
	}
	if HasNEON && len(block) >= 16 {
		DCT4x4_NEON(&block[0])
		return
	}
	DCT4x4Scalar(block)
}

// Quantization tables (ITU-T H.264 §8.5.8)
// MF[qp%6] for 4×4 blocks.
var quantMF = [6][3]int16{
	{13107, 5243, 8066},
	{11916, 4660, 7490},
	{10082, 4194, 6554},
	{9362, 3647, 5825},
	{8192, 3355, 5243},
	{7282, 2893, 4559},
}

// Dequantization scale factors.
var dequantV = [6][3]int16{
	{10, 16, 13},
	{11, 18, 14},
	{13, 20, 16},
	{14, 23, 18},
	{16, 25, 20},
	{18, 29, 23},
}

// Position-to-V-index mapping for 4×4 block.
var posToV = [16]int{
	0, 2, 0, 2,
	2, 1, 2, 1,
	0, 2, 0, 2,
	2, 1, 2, 1,
}

// Dequant4x4 dequantizes a 4×4 block of transform coefficients.
// QP range: 0-51.
func Dequant4x4(block []int16, qp int) {
	qpDiv6 := uint(qp / 6)
	qpMod6 := qp % 6
	for i := 0; i < 16; i++ {
		if block[i] != 0 {
			v := int32(dequantV[qpMod6][posToV[i]])
			block[i] = int16(int32(block[i]) * v << qpDiv6)
		}
	}
}

// Quant4x4 quantizes a 4×4 block of transform coefficients.
func Quant4x4(block []int16, qp int) {
	qpDiv6 := qp / 6
	qpMod6 := qp % 6
	qbits := uint(15 + qpDiv6)
	add := int32(1<<qbits) / 3 // rounding offset (intra: 1/3, inter: 1/6)
	for i := 0; i < 16; i++ {
		if block[i] != 0 {
			mf := int32(quantMF[qpMod6][posToV[i]])
			v := int32(block[i])
			sign := int32(1)
			if v < 0 {
				sign = -1
				v = -v
			}
			block[i] = int16(sign * ((v*mf + add) >> qbits))
		}
	}
}

// Zig-zag scan order for 4×4 blocks.
var ZigZag4x4 = [16]int{
	0, 1, 4, 8,
	5, 2, 3, 6,
	9, 12, 13, 10,
	7, 11, 14, 15,
}
