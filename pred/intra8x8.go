package pred

// PredIntra8x8 generates the filtered predicted 8×8 block from neighboring pixels.
// Implements all 9 H.264 §8.3.2.2 Intra_8x8 prediction modes with the mandatory
// §8.3.2.3 reference-pixel strong filter applied inline (matching FFmpeg's
// PREDICT_8x8_LOAD_TOP/LEFT/TOPLEFT/TOPRIGHT macros in h264pred_template.c).
//
//   - top:     16-byte slice (top[0..7] = above block; top[8..15] = top-right)
//   - left:    8-byte slice  (left[0..7] = pixels to the left)
//   - topLeft: corner pixel p[-1][-1]
func PredIntra8x8(pred []uint8, mode int, top, left []uint8, topLeft uint8) {
	hasTopRight := len(top) >= 16 && top[8] != top[7]

	clip8 := func(v int) uint8 {
		if v < 0 {
			return 0
		}
		if v > 255 {
			return 255
		}
		return uint8(v)
	}

	// Compute filtered topLeft (lt), top row (t[0..15]) and left column (l[0..7]).
	// Uses raw unfiltered pixels as computed in §8.3.2.3.
	var lt int
	var t [16]int
	var l [8]int

	// lt = (left[0] + 2*topLeft + top[0] + 2) >> 2
	lt = (int(left[0]) + 2*int(topLeft) + int(top[0]) + 2) >> 2

	// t[0] = (topLeft + 2*top[0] + top[1] + 2) >> 2
	t[0] = (int(topLeft) + 2*int(top[0]) + int(top[1]) + 2) >> 2
	for x := 1; x <= 6; x++ {
		t[x] = (int(top[x-1]) + 2*int(top[x]) + int(top[x+1]) + 2) >> 2
	}
	if hasTopRight {
		t[7] = (int(top[6]) + 2*int(top[7]) + int(top[8]) + 2) >> 2
		for x := 8; x <= 14; x++ {
			t[x] = (int(top[x-1]) + 2*int(top[x]) + int(top[x+1]) + 2) >> 2
		}
		t[15] = (int(top[14]) + 3*int(top[15]) + 2) >> 2
	} else {
		t[7] = (int(top[6]) + 3*int(top[7]) + 2) >> 2
		for x := 8; x <= 15; x++ {
			t[x] = int(top[7])
		}
	}

	// l[0] = (topLeft + 2*left[0] + left[1] + 2) >> 2
	l[0] = (int(topLeft) + 2*int(left[0]) + int(left[1]) + 2) >> 2
	for k := 1; k <= 6; k++ {
		l[k] = (int(left[k-1]) + 2*int(left[k]) + int(left[k+1]) + 2) >> 2
	}
	l[7] = (int(left[6]) + 3*int(left[7]) + 2) >> 2

	// Bounds-safe accessors: index -1 maps to filtered topLeft (lt).
	tAt := func(k int) int {
		if k < 0 {
			return lt
		}
		return t[k]
	}
	lAt := func(k int) int {
		if k < 0 {
			return lt
		}
		return l[k]
	}

	switch mode {
	case Intra4x4Vertical:
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				pred[y*8+x] = uint8(t[x])
			}
		}

	case Intra4x4Horizontal:
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				pred[y*8+x] = uint8(l[y])
			}
		}

	case Intra4x4DC:
		// DC: no filter — use raw reference pixels.
		sum := 0
		for i := 0; i < 8; i++ {
			sum += int(top[i]) + int(left[i])
		}
		dc := uint8((sum + 8) >> 4)
		for i := range pred[:64] {
			pred[i] = dc
		}

	case Intra4x4DiagDownLeft:
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				if x == 7 && y == 7 {
					pred[y*8+x] = clip8((t[14] + 3*t[15] + 2) >> 2)
				} else {
					pred[y*8+x] = clip8((t[x+y] + 2*t[x+y+1] + t[x+y+2] + 2) >> 2)
				}
			}
		}

	case Intra4x4DiagDownRight:
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				if x > y {
					// tAt handles t[-1] → lt when x-y-2 = -1
					pred[y*8+x] = clip8((tAt(x-y-2) + 2*tAt(x-y-1) + t[x-y] + 2) >> 2)
				} else if y > x {
					pred[y*8+x] = clip8((lAt(y-x-2) + 2*lAt(y-x-1) + l[y-x] + 2) >> 2)
				} else { // x == y
					pred[y*8+x] = clip8((t[0] + 2*lt + l[0] + 2) >> 2)
				}
			}
		}

	case Intra4x4VerticalRight:
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				zVR := 2*x - y
				switch {
				case zVR >= 0 && zVR%2 == 0:
					idx := x - (y >> 1)
					pred[y*8+x] = clip8((tAt(idx-1) + t[idx] + 1) >> 1)
				case zVR > 0 && zVR%2 == 1:
					idx := x - (y >> 1)
					pred[y*8+x] = clip8((tAt(idx-2) + 2*tAt(idx-1) + t[idx] + 2) >> 2)
				case zVR == -1:
					pred[y*8+x] = clip8((l[0] + 2*lt + t[0] + 2) >> 2)
				default: // zVR <= -2
					idx := y - 2*x
					pred[y*8+x] = clip8((lAt(idx-2) + 2*lAt(idx-1) + l[idx] + 2) >> 2)
				}
			}
		}

	case Intra4x4HorizontalDown:
		set := func(x, y, v int) { pred[y*8+x] = clip8(v) }
		set(0, 7, (l[6]+l[7]+1)>>1)
		set(1, 7, (l[5]+2*l[6]+l[7]+2)>>2)
		set(0, 6, (l[5]+l[6]+1)>>1)
		set(2, 7, (l[5]+l[6]+1)>>1)
		set(1, 6, (l[4]+2*l[5]+l[6]+2)>>2)
		set(3, 7, (l[4]+2*l[5]+l[6]+2)>>2)
		set(0, 5, (l[4]+l[5]+1)>>1)
		set(2, 6, (l[4]+l[5]+1)>>1)
		set(4, 7, (l[4]+l[5]+1)>>1)
		set(1, 5, (l[3]+2*l[4]+l[5]+2)>>2)
		set(3, 6, (l[3]+2*l[4]+l[5]+2)>>2)
		set(5, 7, (l[3]+2*l[4]+l[5]+2)>>2)
		set(0, 4, (l[3]+l[4]+1)>>1)
		set(2, 5, (l[3]+l[4]+1)>>1)
		set(4, 6, (l[3]+l[4]+1)>>1)
		set(6, 7, (l[3]+l[4]+1)>>1)
		set(1, 4, (l[2]+2*l[3]+l[4]+2)>>2)
		set(3, 5, (l[2]+2*l[3]+l[4]+2)>>2)
		set(5, 6, (l[2]+2*l[3]+l[4]+2)>>2)
		set(7, 7, (l[2]+2*l[3]+l[4]+2)>>2)
		set(0, 3, (l[2]+l[3]+1)>>1)
		set(2, 4, (l[2]+l[3]+1)>>1)
		set(4, 5, (l[2]+l[3]+1)>>1)
		set(6, 6, (l[2]+l[3]+1)>>1)
		set(1, 3, (l[1]+2*l[2]+l[3]+2)>>2)
		set(3, 4, (l[1]+2*l[2]+l[3]+2)>>2)
		set(5, 5, (l[1]+2*l[2]+l[3]+2)>>2)
		set(7, 6, (l[1]+2*l[2]+l[3]+2)>>2)
		set(0, 2, (l[1]+l[2]+1)>>1)
		set(2, 3, (l[1]+l[2]+1)>>1)
		set(4, 4, (l[1]+l[2]+1)>>1)
		set(6, 5, (l[1]+l[2]+1)>>1)
		set(1, 2, (l[0]+2*l[1]+l[2]+2)>>2)
		set(3, 3, (l[0]+2*l[1]+l[2]+2)>>2)
		set(5, 4, (l[0]+2*l[1]+l[2]+2)>>2)
		set(7, 5, (l[0]+2*l[1]+l[2]+2)>>2)
		set(0, 1, (l[0]+l[1]+1)>>1)
		set(2, 2, (l[0]+l[1]+1)>>1)
		set(4, 3, (l[0]+l[1]+1)>>1)
		set(6, 4, (l[0]+l[1]+1)>>1)
		set(1, 1, (lt+2*l[0]+l[1]+2)>>2)
		set(3, 2, (lt+2*l[0]+l[1]+2)>>2)
		set(5, 3, (lt+2*l[0]+l[1]+2)>>2)
		set(7, 4, (lt+2*l[0]+l[1]+2)>>2)
		set(0, 0, (lt+l[0]+1)>>1)
		set(2, 1, (lt+l[0]+1)>>1)
		set(4, 2, (lt+l[0]+1)>>1)
		set(6, 3, (lt+l[0]+1)>>1)
		set(1, 0, (l[0]+2*lt+t[0]+2)>>2)
		set(3, 1, (l[0]+2*lt+t[0]+2)>>2)
		set(5, 2, (l[0]+2*lt+t[0]+2)>>2)
		set(7, 3, (l[0]+2*lt+t[0]+2)>>2)
		set(2, 0, (t[1]+2*t[0]+lt+2)>>2)
		set(4, 1, (t[1]+2*t[0]+lt+2)>>2)
		set(6, 2, (t[1]+2*t[0]+lt+2)>>2)
		set(3, 0, (t[2]+2*t[1]+t[0]+2)>>2)
		set(5, 1, (t[2]+2*t[1]+t[0]+2)>>2)
		set(7, 2, (t[2]+2*t[1]+t[0]+2)>>2)
		set(4, 0, (t[3]+2*t[2]+t[1]+2)>>2)
		set(6, 1, (t[3]+2*t[2]+t[1]+2)>>2)
		set(5, 0, (t[4]+2*t[3]+t[2]+2)>>2)
		set(7, 1, (t[4]+2*t[3]+t[2]+2)>>2)
		set(6, 0, (t[5]+2*t[4]+t[3]+2)>>2)
		set(7, 0, (t[6]+2*t[5]+t[4]+2)>>2)

	case Intra4x4VerticalLeft:
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				idx := x + (y >> 1)
				if y&1 == 0 {
					pred[y*8+x] = clip8((t[idx] + t[idx+1] + 1) >> 1)
				} else {
					pred[y*8+x] = clip8((t[idx] + 2*t[idx+1] + t[idx+2] + 2) >> 2)
				}
			}
		}

	case Intra4x4HorizontalUp:
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				zHU := x + 2*y
				idx := y + (x >> 1)
				switch {
				case zHU%2 == 0 && idx < 7:
					pred[y*8+x] = clip8((l[idx] + l[idx+1] + 1) >> 1)
				case zHU%2 == 1 && idx < 6:
					pred[y*8+x] = clip8((l[idx] + 2*l[idx+1] + l[idx+2] + 2) >> 2)
				case zHU%2 == 1 && idx == 6:
					pred[y*8+x] = clip8((l[6] + 2*l[7] + l[7] + 2) >> 2)
				case zHU == 13:
					pred[y*8+x] = clip8((l[6] + 3*l[7] + 2) >> 2)
				default:
					pred[y*8+x] = uint8(l[7])
				}
			}
		}

	default:
		sum := uint16(0)
		for i := 0; i < 8; i++ {
			sum += uint16(top[i]) + uint16(left[i])
		}
		dc := uint8((sum + 8) >> 4)
		for i := range pred[:64] {
			pred[i] = dc
		}
	}
}
