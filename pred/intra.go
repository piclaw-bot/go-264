package pred

// H.264 intra prediction for 4×4 and 16×16 luma blocks.
// ITU-T H.264 §8.3.1

// Intra4x4 prediction modes
const (
	Intra4x4Vertical   = 0
	Intra4x4Horizontal = 1
	Intra4x4DC         = 2
	Intra4x4DiagDownLeft  = 3
	Intra4x4DiagDownRight = 4
	Intra4x4VerticalRight = 5
	Intra4x4HorizontalDown = 6
	Intra4x4VerticalLeft  = 7
	Intra4x4HorizontalUp  = 8
)

// Intra16x16 prediction modes
const (
	Intra16x16Vertical   = 0
	Intra16x16Horizontal = 1
	Intra16x16DC         = 2
	Intra16x16Plane      = 3
)

// clip8 clamps an int to [0,255].
func clip8(v int) uint8 {
	if v < 0 { return 0 }
	if v > 255 { return 255 }
	return uint8(v)
}

// PredIntra4x4 generates the predicted 4×4 block from neighboring pixels.
// top[0..3]=A..D, topRight[0..3]=E..H, left[0..3]=I..L, topLeft=M.
func PredIntra4x4(pred []uint8, mode int, top, topRight, left []uint8, topLeft uint8) {
	// Build extended reference array: p[-1,-1], p[0,-1]..p[7,-1], p[-1,0]..p[-1,3]
	// p[x,-1] for x=-1: topLeft, x=0..3: top[x], x=4..7: topRight[x-4]
	// p[-1,y] for y=0..3: left[y]
	var p [13]int // [-1,-1], [0,-1]..[7,-1], [-1,0]..[−1,3]
	// p[0] = p[-1,-1] = topLeft
	p[0] = int(topLeft)
	// p[1..4] = p[0,-1]..p[3,-1] = top
	for i := 0; i < 4; i++ { p[1+i] = int(top[i]) }
	// p[5..8] = p[4,-1]..p[7,-1] = topRight
	for i := 0; i < 4; i++ { p[5+i] = int(topRight[i]) }
	// p[9..12] = p[-1,0]..p[-1,3] = left
	for i := 0; i < 4; i++ { p[9+i] = int(left[i]) }
	
	// Helper to get p[x,-1] (x from -1 to 7)
	pt := func(x int) int { return p[x+1] }
	// Helper to get p[-1,y] (y from -1 to 3)
	pl := func(y int) int {
		if y == -1 { return p[0] }
		return p[9+y]
	}

	switch mode {
	case Intra4x4Vertical:
		for y := 0; y < 4; y++ {
			for x := 0; x < 4; x++ {
				pred[y*4+x] = uint8(pt(x))
			}
		}

	case Intra4x4Horizontal:
		for y := 0; y < 4; y++ {
			for x := 0; x < 4; x++ {
				pred[y*4+x] = uint8(pl(y))
			}
		}

	case Intra4x4DC:
		sum := 0
		for i := 0; i < 4; i++ {
			sum += pt(i) + pl(i)
		}
		dc := uint8((sum + 4) >> 3)
		for i := range pred[:16] {
			pred[i] = dc
		}

	case Intra4x4DiagDownLeft:
		for y := 0; y < 4; y++ {
			for x := 0; x < 4; x++ {
				if x == 3 && y == 3 {
					pred[y*4+x] = clip8((pt(6) + 3*pt(7) + 2) >> 2)
				} else {
					pred[y*4+x] = clip8((pt(x+y) + 2*pt(x+y+1) + pt(x+y+2) + 2) >> 2)
				}
			}
		}

	case Intra4x4DiagDownRight:
		for y := 0; y < 4; y++ {
			for x := 0; x < 4; x++ {
				if x > y {
					pred[y*4+x] = clip8((pt(x-y-2) + 2*pt(x-y-1) + pt(x-y) + 2) >> 2)
				} else if y > x {
					pred[y*4+x] = clip8((pl(y-x-2) + 2*pl(y-x-1) + pl(y-x) + 2) >> 2)
				} else { // x == y
					pred[y*4+x] = clip8((pt(0) + 2*pl(-1) + pl(0) + 2) >> 2)
				}
			}
		}

	case Intra4x4VerticalRight:
		for y := 0; y < 4; y++ {
			for x := 0; x < 4; x++ {
				zVR := 2*x - y
				switch {
				case zVR == 0 || zVR == 2 || zVR == 4 || zVR == 6:
					idx := x - (y >> 1)
					pred[y*4+x] = clip8((pt(idx-1) + pt(idx) + 1) >> 1)
				case zVR == 1 || zVR == 3 || zVR == 5:
					idx := x - (y >> 1)
					pred[y*4+x] = clip8((pt(idx-2) + 2*pt(idx-1) + pt(idx) + 2) >> 2)
				case zVR == -1:
					pred[y*4+x] = clip8((pl(0) + 2*pl(-1) + pt(0) + 2) >> 2)
				default: // zVR < -1
					pred[y*4+x] = clip8((pl(y-1) + 2*pl(y-2) + pl(y-3) + 2) >> 2)
				}
			}
		}

	case Intra4x4HorizontalDown:
		for y := 0; y < 4; y++ {
			for x := 0; x < 4; x++ {
				zHD := 2*y - x
				switch {
				case zHD == 0 || zHD == 2 || zHD == 4 || zHD == 6:
					idx := y - (x >> 1)
					pred[y*4+x] = clip8((pl(idx-1) + pl(idx) + 1) >> 1)
				case zHD == 1 || zHD == 3 || zHD == 5:
					idx := y - (x >> 1)
					pred[y*4+x] = clip8((pl(idx-2) + 2*pl(idx-1) + pl(idx) + 2) >> 2)
				case zHD == -1:
					pred[y*4+x] = clip8((pt(0) + 2*pl(-1) + pl(0) + 2) >> 2)
				default: // zHD < -1
					pred[y*4+x] = clip8((pt(x-1) + 2*pt(x-2) + pt(x-3) + 2) >> 2)
				}
			}
		}

	case Intra4x4VerticalLeft:
		for y := 0; y < 4; y++ {
			for x := 0; x < 4; x++ {
				idx := x + (y >> 1)
				if y&1 == 0 {
					pred[y*4+x] = clip8((pt(idx) + pt(idx+1) + 1) >> 1)
				} else {
					pred[y*4+x] = clip8((pt(idx) + 2*pt(idx+1) + pt(idx+2) + 2) >> 2)
				}
			}
		}

	case Intra4x4HorizontalUp:
		for y := 0; y < 4; y++ {
			for x := 0; x < 4; x++ {
				zHU := x + 2*y
				switch {
				case zHU == 0 || zHU == 2 || zHU == 4:
					idx := y + (x >> 1)
					if idx+1 <= 3 {
						pred[y*4+x] = clip8((pl(idx) + pl(idx+1) + 1) >> 1)
					} else {
						pred[y*4+x] = uint8(pl(3))
					}
				case zHU == 1 || zHU == 3:
					idx := y + (x >> 1)
					if idx+2 <= 3 {
						pred[y*4+x] = clip8((pl(idx) + 2*pl(idx+1) + pl(idx+2) + 2) >> 2)
					} else if idx+1 <= 3 {
						pred[y*4+x] = clip8((pl(idx) + 2*pl(idx+1) + pl(3) + 2) >> 2)
					} else {
						pred[y*4+x] = uint8(pl(3))
					}
				case zHU == 5:
					pred[y*4+x] = clip8((pl(2) + 3*pl(3) + 2) >> 2)
				default: // zHU >= 6
					pred[y*4+x] = uint8(pl(3))
				}
			}
		}
	}
}

// PredIntra16x16 generates a 16×16 predicted block.
func PredIntra16x16(pred []uint8, mode int, top, left []uint8, topLeft uint8) {
	switch mode {
	case Intra16x16Vertical:
		if HasSSE2 {
			IntraPred16x16V_ASM(&pred[0], &top[0])
		} else if hasNEONPred() {
			intraPred16x16V_NEON(&pred[0], &top[0])
		} else {
			for y := 0; y < 16; y++ {
				copy(pred[y*16:(y+1)*16], top[:16])
			}
		}

	case Intra16x16Horizontal:
		if HasSSE2 {
			IntraPred16x16H_ASM(&pred[0], &left[0])
		} else if hasNEONPred() {
			intraPred16x16H_NEON(&pred[0], &left[0])
		} else {
			for y := 0; y < 16; y++ {
				for x := 0; x < 16; x++ {
					pred[y*16+x] = left[y]
				}
			}
		}

	case Intra16x16DC:
		sum := uint32(0)
		for i := 0; i < 16; i++ {
			sum += uint32(top[i]) + uint32(left[i])
		}
		dc := uint8((sum + 16) >> 5)
		if HasSSE2 {
			IntraPred16x16DC_ASM(&pred[0], dc)
		} else if hasNEONPred() {
			intraPred16x16DC_NEON(&pred[0], dc)
		} else {
			for i := range pred[:256] {
				pred[i] = dc
			}
		}

	case Intra16x16Plane:
		H := int32(0)
		for x := 0; x < 8; x++ {
			var l uint8
			if x == 7 { l = topLeft } else { l = top[6-x] }
			H += int32(x+1) * (int32(top[8+x]) - int32(l))
		}
		V := int32(0)
		for y := 0; y < 8; y++ {
			var u uint8
			if y == 7 { u = topLeft } else { u = left[6-y] }
			V += int32(y+1) * (int32(left[8+y]) - int32(u))
		}
		a := 16 * (int32(top[15]) + int32(left[15]))
		b := (5*H + 32) >> 6
		c := (5*V + 32) >> 6
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				v := (a + b*(int32(x)-7) + c*(int32(y)-7) + 16) >> 5
				if v < 0 { v = 0 }
				if v > 255 { v = 255 }
				pred[y*16+x] = uint8(v)
			}
		}
	}
}
