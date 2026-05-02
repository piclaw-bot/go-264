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
	// Horizontal pass (rows)
	for i := 0; i < 4; i++ {
		row := block[i*4 : i*4+4]
		e0 := row[0] + row[2]
		e1 := row[0] - row[2]
		e2 := (row[1] >> 1) - row[3]
		e3 := row[1] + (row[3] >> 1)
		row[0] = e0 + e3
		row[1] = e1 + e2
		row[2] = e1 - e2
		row[3] = e0 - e3
	}
	// Vertical pass (columns)
	for j := 0; j < 4; j++ {
		c0, c1, c2, c3 := block[j], block[4+j], block[8+j], block[12+j]
		e0 := c0 + c2
		e1 := c0 - c2
		e2 := (c1 >> 1) - c3
		e3 := c1 + (c3 >> 1)
		block[j] = (e0 + e3 + 32) >> 6
		block[4+j] = (e1 + e2 + 32) >> 6
		block[8+j] = (e1 - e2 + 32) >> 6
		block[12+j] = (e0 - e3 + 32) >> 6
	}
}

// DCT4x4 performs the forward 4×4 integer transform (in-place).
// Input: residual pixel values in block[0:16].
// Output: transform coefficients (before quantization).
func DCT4x4(block []int16) {
	// Horizontal pass (rows)
	for i := 0; i < 4; i++ {
		row := block[i*4 : i*4+4]
		s0 := row[0] + row[3]
		s1 := row[1] + row[2]
		s2 := row[1] - row[2]
		s3 := row[0] - row[3]
		row[0] = s0 + s1
		row[1] = (s3 << 1) + s2
		row[2] = s0 - s1
		row[3] = s3 - (s2 << 1)
	}
	// Vertical pass (columns)
	for j := 0; j < 4; j++ {
		c0, c1, c2, c3 := block[j], block[4+j], block[8+j], block[12+j]
		s0 := c0 + c3
		s1 := c1 + c2
		s2 := c1 - c2
		s3 := c0 - c3
		block[j] = s0 + s1
		block[4+j] = (s3 << 1) + s2
		block[8+j] = s0 - s1
		block[12+j] = s3 - (s2 << 1)
	}
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
