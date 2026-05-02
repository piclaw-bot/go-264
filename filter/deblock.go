package filter

// H.264 in-loop deblocking filter.
// ITU-T H.264 §8.7
//
// Applied at macroblock edges (horizontal and vertical).
// Filters luma and chroma independently.
// Strength depends on whether edge is at block/MB boundary and coding modes.

// Clip3 clamps v to [lo, hi].
func Clip3(lo, hi, v int) int {
	if v < lo { return lo }
	if v > hi { return hi }
	return v
}

// Clip1 clamps to [0, 255].
func Clip1(v int) uint8 {
	if v < 0 { return 0 }
	if v > 255 { return 255 }
	return uint8(v)
}

// Alpha and Beta threshold tables (Table 8-16, 8-17)
var alphaTable = [52]int{
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	4, 4, 5, 6, 7, 8, 9, 10, 12, 13, 15, 17, 20, 22, 25, 28,
	32, 36, 40, 45, 50, 56, 63, 71, 80, 90, 101, 113, 127, 144, 162, 182,
	203, 226, 255, 255,
}

var betaTable = [52]int{
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	2, 2, 2, 3, 3, 3, 3, 4, 4, 4, 6, 6, 7, 7, 8, 8,
	9, 9, 10, 10, 11, 11, 12, 12, 13, 13, 14, 14, 15, 15, 16, 16,
	17, 17, 18, 18,
}

// TC0 table (Table 8-18) indexed by [indexA][bS-1]
var tc0Table = [52][3]int{
	{0, 0, 0}, {0, 0, 0}, {0, 0, 0}, {0, 0, 0}, {0, 0, 0}, {0, 0, 0},
	{0, 0, 0}, {0, 0, 0}, {0, 0, 0}, {0, 0, 0}, {0, 0, 0}, {0, 0, 0},
	{0, 0, 0}, {0, 0, 0}, {0, 0, 0}, {0, 0, 0}, {0, 0, 0}, {0, 0, 1},
	{0, 0, 1}, {0, 0, 1}, {0, 0, 1}, {0, 1, 1}, {0, 1, 1}, {1, 1, 1},
	{1, 1, 1}, {1, 1, 1}, {1, 1, 1}, {1, 1, 2}, {1, 1, 2}, {1, 1, 2},
	{1, 1, 2}, {1, 2, 3}, {1, 2, 3}, {2, 2, 3}, {2, 2, 4}, {2, 3, 4},
	{2, 3, 4}, {3, 3, 5}, {3, 4, 6}, {3, 4, 6}, {4, 5, 7}, {4, 5, 8},
	{4, 6, 9}, {5, 7, 10}, {6, 8, 11}, {6, 8, 13}, {7, 10, 14}, {8, 11, 16},
	{9, 12, 18}, {10, 13, 20}, {11, 15, 23}, {13, 17, 25},
}

// FilterEdgeV applies the vertical deblocking filter to a 4-pixel edge.
// p3,p2,p1,p0 | q0,q1,q2,q3 (p on left, q on right)
// bS: boundary strength (0-4)
// indexA: alpha/tc0 table index (QP-dependent)
func FilterEdgeV(p []uint8, q []uint8, stride int, bS int, indexA int) {
	if bS == 0 || indexA < 0 || indexA > 51 {
		return
	}

	alpha := alphaTable[indexA]
	beta := betaTable[indexA]

	for i := 0; i < 4; i++ {
		p0 := int(p[i*stride])
		p1 := int(p[i*stride-1])
		p2 := int(p[i*stride-2])
		q0 := int(q[i*stride])
		q1 := int(q[i*stride+1])
		q2 := int(q[i*stride+2])

		// Check filter condition
		if abs(p0-q0) >= alpha || abs(p1-p0) >= beta || abs(q1-q0) >= beta {
			continue
		}

		if bS < 4 {
			// Normal filter
			tc0 := tc0Table[indexA][bS-1]
			tc := tc0
			if abs(p2-p0) < beta { tc++ }
			if abs(q2-q0) < beta { tc++ }

			delta := Clip3(-tc, tc, ((q0-p0)*4+(p1-q1)+4)>>3)
			p[i*stride] = Clip1(p0 + delta)
			q[i*stride] = Clip1(q0 - delta)

			if abs(p2-p0) < beta {
				p[i*stride-1] = Clip1(p1 + Clip3(-tc0, tc0, (p2+((p0+q0+1)>>1)-(p1<<1))>>1))
			}
			if abs(q2-q0) < beta {
				q[i*stride+1] = Clip1(q1 + Clip3(-tc0, tc0, (q2+((p0+q0+1)>>1)-(q1<<1))>>1))
			}
		} else {
			// Strong filter (bS == 4, for intra edges)
			if abs(p0-q0) < ((alpha>>2)+2) && abs(p2-p0) < beta {
				p[i*stride] = Clip1((p2 + 2*p1 + 2*p0 + 2*q0 + q1 + 4) >> 3)
				p[i*stride-1] = Clip1((p2 + p1 + p0 + q0 + 2) >> 2)
				p[i*stride-2] = Clip1((2*int(p[i*stride-3]) + 3*p2 + p1 + p0 + q0 + 4) >> 3)
			} else {
				p[i*stride] = Clip1((2*p1 + p0 + q1 + 2) >> 2)
			}
			if abs(p0-q0) < ((alpha>>2)+2) && abs(q2-q0) < beta {
				q[i*stride] = Clip1((p1 + 2*p0 + 2*q0 + 2*q1 + q2 + 4) >> 3)
				q[i*stride+1] = Clip1((p0 + q0 + q1 + q2 + 2) >> 2)
				q[i*stride+2] = Clip1((2*int(q[i*stride+3]) + 3*q2 + q1 + q0 + p0 + 4) >> 3)
			} else {
				q[i*stride] = Clip1((2*q1 + q0 + p1 + 2) >> 2)
			}
		}
	}
}

func abs(x int) int {
	if x < 0 { return -x }
	return x
}
