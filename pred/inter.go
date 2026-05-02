package pred

// H.264 inter prediction (motion compensation).
// ITU-T H.264 §8.4

// MotionVector represents a motion vector in quarter-pixel units.
type MotionVector struct {
	X, Y int16
}

// InterPred16x16 performs motion compensation for a 16x16 block.
// ref: reference frame luma plane
// mv: motion vector in quarter-pixel units
// stride: reference frame stride
// Output: predicted 16x16 block
func InterPred16x16(out []uint8, ref []uint8, stride int, mv MotionVector) {
	fx := int(mv.X) >> 2
	fy := int(mv.Y) >> 2
	refH := len(ref) / stride

	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			srcX := x + fx
			srcY := y + fy
			if srcX < 0 { srcX = 0 }
			if srcY < 0 { srcY = 0 }
			if srcX >= stride { srcX = stride - 1 }
			if srcY >= refH { srcY = refH - 1 }
			out[y*16+x] = ref[srcY*stride+srcX]
		}
	}
}

// SubpelFilter6Tap applies the 6-tap FIR filter for half-pixel interpolation.
// Coefficients: [1, -5, 20, 20, -5, 1] / 32
func SubpelFilter6Tap(samples [6]uint8) uint8 {
	v := int(samples[0]) - 5*int(samples[1]) + 20*int(samples[2]) +
		20*int(samples[3]) - 5*int(samples[4]) + int(samples[5])
	v = (v + 16) >> 5 // round
	if v < 0 { return 0 }
	if v > 255 { return 255 }
	return uint8(v)
}
