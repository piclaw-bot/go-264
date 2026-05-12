package pred

// H.264 inter prediction (motion compensation).
// ITU-T H.264 §8.4

// MotionVector represents a motion vector in quarter-pixel units.
type MotionVector struct {
	X, Y int16
}

// InterPred16x16 performs motion compensation for a 16x16 block from origin (0,0).
// ref: reference frame luma plane
// mv: motion vector in quarter-pixel units
// stride: reference frame stride
// Output: predicted 16x16 block
func InterPred16x16(out []uint8, ref []uint8, stride int, mv MotionVector) {
	InterPred16x16At(out, ref, stride, 0, 0, mv)
}

// InterPred16x16At performs motion compensation for a 16x16 block at
// macroblock origin (baseX, baseY) using bilinear interpolation for
// fractional MVs. Integer MVs use the fast SIMD copy path.
//
// Fast path: when the requested 16x16 source rectangle is fully inside the
// reference plane, copy rows with the platform SIMD routine:
//
//	amd64: SSE2 MOVOU row copies
//	arm64: NEON row copies
//
// Scalar fallback handles clipped edges.
func InterPred16x16At(out []uint8, ref []uint8, stride int, baseX, baseY int, mv MotionVector) {
	if len(out) < 256 || stride <= 0 || len(ref) == 0 {
		return
	}
	mvx, mvy := int(mv.X), int(mv.Y)
	ix, iy := mvx>>2, mvy>>2
	fx, fy := mvx&3, mvy&3
	refH := len(ref) / stride
	sx := baseX + ix
	sy := baseY + iy

	if fx == 0 && fy == 0 && sx >= 0 && sy >= 0 && sx+16 <= stride && sy+16 <= refH {
		if HasSSE2 || hasNEONPred() {
			InterPred16x16Copy_ASM(&out[0], &ref[sy*stride+sx], 16, stride)
			return
		}
		for y := 0; y < 16; y++ {
			copy(out[y*16:y*16+16], ref[(sy+y)*stride+sx:(sy+y)*stride+sx+16])
		}
		return
	}
	if sx >= 0 && sy >= 0 && sx+17 <= stride && sy+17 <= refH {
		if fy == 0 {
			w0, w1 := 4-fx, fx
			for y := 0; y < 16; y++ {
				row := (sy+y)*stride + sx
				outRow := y * 16
				for x := 0; x < 16; x++ {
					a := int(ref[row+x])
					b := int(ref[row+x+1])
					out[outRow+x] = uint8((a*w0 + b*w1 + 2) >> 2)
				}
			}
			return
		}
		if fx == 0 {
			w0, w1 := 4-fy, fy
			for y := 0; y < 16; y++ {
				row0 := (sy+y)*stride + sx
				row1 := row0 + stride
				outRow := y * 16
				for x := 0; x < 16; x++ {
					a := int(ref[row0+x])
					c := int(ref[row1+x])
					out[outRow+x] = uint8((a*w0 + c*w1 + 2) >> 2)
				}
			}
			return
		}
		w00 := (4 - fx) * (4 - fy)
		w10 := fx * (4 - fy)
		w01 := (4 - fx) * fy
		w11 := fx * fy
		for y := 0; y < 16; y++ {
			row0 := (sy+y)*stride + sx
			row1 := row0 + stride
			outRow := y * 16
			for x := 0; x < 16; x++ {
				a := int(ref[row0+x])
				b := int(ref[row0+x+1])
				c := int(ref[row1+x])
				d := int(ref[row1+x+1])
				out[outRow+x] = uint8((a*w00 + b*w10 + c*w01 + d*w11 + 8) >> 4)
			}
		}
		return
	}
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			x0, y0 := sx+x, sy+y
			cx0, cy0 := x0, y0
			if cx0 < 0 {
				cx0 = 0
			} else if cx0 >= stride {
				cx0 = stride - 1
			}
			if cy0 < 0 {
				cy0 = 0
			} else if cy0 >= refH {
				cy0 = refH - 1
			}
			if fx == 0 && fy == 0 {
				out[y*16+x] = ref[cy0*stride+cx0]
				continue
			}
			cx1, cy1 := x0+1, y0+1
			if cx1 < 0 {
				cx1 = 0
			} else if cx1 >= stride {
				cx1 = stride - 1
			}
			if cy1 < 0 {
				cy1 = 0
			} else if cy1 >= refH {
				cy1 = refH - 1
			}
			a := int(ref[cy0*stride+cx0])
			b := int(ref[cy0*stride+cx1])
			c := int(ref[cy1*stride+cx0])
			d := int(ref[cy1*stride+cx1])
			v := a*(4-fx)*(4-fy) + b*fx*(4-fy) + c*(4-fx)*fy + d*fx*fy
			out[y*16+x] = uint8((v + 8) >> 4)
		}
	}
}

// SubpelFilter6Tap applies the 6-tap FIR filter for half-pixel interpolation.
// Coefficients: [1, -5, 20, 20, -5, 1] / 32
func SubpelFilter6Tap(samples [6]uint8) uint8 {
	v := int(samples[0]) - 5*int(samples[1]) + 20*int(samples[2]) +
		20*int(samples[3]) - 5*int(samples[4]) + int(samples[5])
	v = (v + 16) >> 5 // round
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}
