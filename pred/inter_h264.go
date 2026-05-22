package pred

// h264LumaHalfPel applies the 6-tap FIR filter [1,-5,20,20,-5,1]/32 at half-pixel positions.
// H.264 §8.4.2.2.1

func clip8i(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func clampRef(ref []uint8, stride, x, y, h int) uint8 {
	if x < 0 {
		x = 0
	} else if x >= stride {
		x = stride - 1
	}
	if y < 0 {
		y = 0
	} else if y >= h {
		y = h - 1
	}
	return ref[y*stride+x]
}

// h264Tap6 applies 6-tap filter to 6 samples: (a-5b+20c+20d-5e+f+16)>>5
func h264Tap6(a, b, c, d, e, f int) int {
	return a - 5*b + 20*c + 20*d - 5*e + f
}

// InterPredLumaH264 performs H.264-compliant luma inter prediction for an NxM block.
// It uses the 6-tap FIR filter for half-pel and averaging for quarter-pel.
func InterPredLumaH264(out []uint8, outStride int, ref []uint8, refStride int, baseX, baseY, w, h int, mv MotionVector) {
	if len(out) < h*outStride || refStride <= 0 || len(ref) == 0 {
		return
	}
	refH := len(ref) / refStride
	mvx, mvy := int(mv.X), int(mv.Y)
	ix, iy := mvx>>2, mvy>>2
	fx, fy := mvx&3, mvy&3

	getRef := func(x, y int) int {
		return int(clampRef(ref, refStride, x, y, refH))
	}

	if fx == 0 && fy == 0 {
		// Integer pel
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				out[y*outStride+x] = uint8(getRef(baseX+ix+x, baseY+iy+y))
			}
		}
		return
	}

	if fy == 0 {
		// Horizontal half/quarter pel only
		for y := 0; y < h; y++ {
			ry := baseY + iy + y
			for x := 0; x < w; x++ {
				rx := baseX + ix + x
				if fx == 2 {
					// Half pel horizontal: 6-tap
					v := h264Tap6(getRef(rx-2, ry), getRef(rx-1, ry), getRef(rx, ry), getRef(rx+1, ry), getRef(rx+2, ry), getRef(rx+3, ry))
					out[y*outStride+x] = clip8i((v + 16) >> 5)
				} else {
					// Quarter pel: average of integer and half
					half := h264Tap6(getRef(rx-2, ry), getRef(rx-1, ry), getRef(rx, ry), getRef(rx+1, ry), getRef(rx+2, ry), getRef(rx+3, ry))
					halfPel := clip8i((half + 16) >> 5)
					var intPel uint8
					if fx == 1 {
						intPel = uint8(getRef(rx, ry))
					} else {
						intPel = uint8(getRef(rx+1, ry))
					}
					out[y*outStride+x] = uint8((int(intPel) + int(halfPel) + 1) >> 1)
				}
			}
		}
		return
	}

	if fx == 0 {
		// Vertical half/quarter pel only
		for y := 0; y < h; y++ {
			ry := baseY + iy + y
			for x := 0; x < w; x++ {
				rx := baseX + ix + x
				if fy == 2 {
					v := h264Tap6(getRef(rx, ry-2), getRef(rx, ry-1), getRef(rx, ry), getRef(rx, ry+1), getRef(rx, ry+2), getRef(rx, ry+3))
					out[y*outStride+x] = clip8i((v + 16) >> 5)
				} else {
					half := h264Tap6(getRef(rx, ry-2), getRef(rx, ry-1), getRef(rx, ry), getRef(rx, ry+1), getRef(rx, ry+2), getRef(rx, ry+3))
					halfPel := clip8i((half + 16) >> 5)
					var intPel uint8
					if fy == 1 {
						intPel = uint8(getRef(rx, ry))
					} else {
						intPel = uint8(getRef(rx, ry+1))
					}
					out[y*outStride+x] = uint8((int(intPel) + int(halfPel) + 1) >> 1)
				}
			}
		}
		return
	}

	// Both fx and fy non-zero: diagonal
	// First compute horizontal half-pel at fy positions, then vertical 6-tap on that
	if fx == 2 && fy == 2 {
		// Full half-pel diagonal: H then V on the H result
		// Intermediate: horizontal 6-tap for rows [ry-2..ry+3+h]
		tmpH := make([]int, (h+5)*w)
		for y := -2; y < h+3; y++ {
			ry := baseY + iy + y
			for x := 0; x < w; x++ {
				rx := baseX + ix + x
				tmpH[(y+2)*w+x] = h264Tap6(getRef(rx-2, ry), getRef(rx-1, ry), getRef(rx, ry), getRef(rx+1, ry), getRef(rx+2, ry), getRef(rx+3, ry))
			}
		}
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				v := h264Tap6(tmpH[(y)*w+x], tmpH[(y+1)*w+x], tmpH[(y+2)*w+x], tmpH[(y+3)*w+x], tmpH[(y+4)*w+x], tmpH[(y+5)*w+x])
				out[y*outStride+x] = clip8i((v + 512) >> 10)
			}
		}
		return
	}

	// Quarter-pel diagonal: average of two adjacent half-pel values
	// Compute the two relevant half-pel positions and average
	for y := 0; y < h; y++ {
		ry := baseY + iy + y
		for x := 0; x < w; x++ {
			rx := baseX + ix + x
			// Horizontal half at (rx, ry) or (rx, ry+1)
			var hHalf, vHalf int
			hRow := ry
			if fy == 3 {
				hRow = ry + 1
			}
			hHalf = h264Tap6(getRef(rx-2, hRow), getRef(rx-1, hRow), getRef(rx, hRow), getRef(rx+1, hRow), getRef(rx+2, hRow), getRef(rx+3, hRow))
			vCol := rx
			if fx == 3 {
				vCol = rx + 1
			}
			vHalf = h264Tap6(getRef(vCol, ry-2), getRef(vCol, ry-1), getRef(vCol, ry), getRef(vCol, ry+1), getRef(vCol, ry+2), getRef(vCol, ry+3))
			h8 := clip8i((hHalf + 16) >> 5)
			v8 := clip8i((vHalf + 16) >> 5)
			out[y*outStride+x] = uint8((int(h8) + int(v8) + 1) >> 1)
		}
	}
}
