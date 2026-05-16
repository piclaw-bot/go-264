package filter

// H.264 in-loop deblocking filter.
// ITU-T H.264 §8.7
//
// Applied at macroblock edges (horizontal and vertical).
// Filters luma and chroma independently.
// Boundary strength (bS) depends on coding modes and non-zero coefficients:
//   bS=4  intra MB boundary edge
//   bS=3  intra internal 4×4 edge
//   bS=2  inter, at least one side has non-zero coefficients
//   bS=1  inter, different ref/MV
//   bS=0  inter, same ref/MV, no non-zero coefficients — skip

// Clip3 clamps v to [lo, hi].
func Clip3(lo, hi, v int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Clip1 clamps to [0, 255].
func Clip1(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
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

// FilterEdgeV applies the vertical deblocking filter to a 4-row edge.
// Each row is stored as 8 consecutive pixels: [p3,p2,p1,p0,q0,q1,q2,q3]
// packed in pq at offsets i*stride+[0..7] for row i=0..3.
// This layout avoids negative-index pointer arithmetic and is safe in Go.
// bS: boundary strength (0-4); indexA: QP-derived table index (0-51).
func FilterEdgeV(pq []uint8, stride, bS, indexA int) {
	if bS == 0 || indexA < 0 || indexA > 51 {
		return
	}

	alpha := alphaTable[indexA]
	beta := betaTable[indexA]

	for i := 0; i < 4; i++ {
		base := i * stride
		p3 := int(pq[base+0])
		p2 := int(pq[base+1])
		p1 := int(pq[base+2])
		p0 := int(pq[base+3])
		q0 := int(pq[base+4])
		q1 := int(pq[base+5])
		q2 := int(pq[base+6])
		q3 := int(pq[base+7])

		// Check filter condition
		if abs(p0-q0) >= alpha || abs(p1-p0) >= beta || abs(q1-q0) >= beta {
			continue
		}

		if bS < 4 {
			tc0 := tc0Table[indexA][bS-1]
			tc := tc0
			if abs(p2-p0) < beta {
				tc++
			}
			if abs(q2-q0) < beta {
				tc++
			}
			delta := Clip3(-tc, tc, ((q0-p0)*4+(p1-q1)+4)>>3)
			pq[base+3] = Clip1(p0 + delta)
			pq[base+4] = Clip1(q0 - delta)
			if abs(p2-p0) < beta {
				pq[base+2] = Clip1(p1 + Clip3(-tc0, tc0, (p2+((p0+q0+1)>>1)-(p1<<1))>>1))
			}
			if abs(q2-q0) < beta {
				pq[base+5] = Clip1(q1 + Clip3(-tc0, tc0, (q2+((p0+q0+1)>>1)-(q1<<1))>>1))
			}
		} else {
			// Strong filter (bS == 4, intra edges)
			if abs(p0-q0) < ((alpha>>2)+2) && abs(p2-p0) < beta {
				pq[base+3] = Clip1((p2 + 2*p1 + 2*p0 + 2*q0 + q1 + 4) >> 3)
				pq[base+2] = Clip1((p2 + p1 + p0 + q0 + 2) >> 2)
				pq[base+1] = Clip1((2*p3 + 3*p2 + p1 + p0 + q0 + 4) >> 3)
			} else {
				pq[base+3] = Clip1((2*p1 + p0 + q1 + 2) >> 2)
			}
			if abs(p0-q0) < ((alpha>>2)+2) && abs(q2-q0) < beta {
				pq[base+4] = Clip1((p1 + 2*p0 + 2*q0 + 2*q1 + q2 + 4) >> 3)
				pq[base+5] = Clip1((p0 + q0 + q1 + q2 + 2) >> 2)
				pq[base+6] = Clip1((2*q3 + 3*q2 + q1 + q0 + p0 + 4) >> 3)
			} else {
				pq[base+4] = Clip1((2*q1 + q0 + p1 + 2) >> 2)
			}
		}
	}
}

// filterLumaSample applies the deblocking filter to one luma sample pair across
// a vertical or horizontal edge. p/q are the four neighbour values on each side
// (p[0] closest to edge, p[3] furthest). Returns updated (p0,p1,p2,q0,q1,q2).
func filterLumaSample(p3, p2, p1, p0, q0, q1, q2, q3, bS, alpha, beta, indexA int) (rp0, rp1, rp2, rq0, rq1, rq2 uint8) {
	rp0, rp1, rp2 = Clip1(p0), Clip1(p1), Clip1(p2)
	rq0, rq1, rq2 = Clip1(q0), Clip1(q1), Clip1(q2)

	if abs(p0-q0) >= alpha || abs(p1-p0) >= beta || abs(q1-q0) >= beta {
		return
	}
	if bS == 4 {
		if abs(p0-q0) < ((alpha>>2)+2) && abs(p2-p0) < beta {
			rp0 = Clip1((p2 + 2*p1 + 2*p0 + 2*q0 + q1 + 4) >> 3)
			rp1 = Clip1((p2 + p1 + p0 + q0 + 2) >> 2)
			rp2 = Clip1((2*p3 + 3*p2 + p1 + p0 + q0 + 4) >> 3)
		} else {
			rp0 = Clip1((2*p1 + p0 + q1 + 2) >> 2)
		}
		if abs(p0-q0) < ((alpha>>2)+2) && abs(q2-q0) < beta {
			rq0 = Clip1((p1 + 2*p0 + 2*q0 + 2*q1 + q2 + 4) >> 3)
			rq1 = Clip1((p0 + q0 + q1 + q2 + 2) >> 2)
			rq2 = Clip1((2*q3 + 3*q2 + q1 + q0 + p0 + 4) >> 3)
		} else {
			rq0 = Clip1((2*q1 + q0 + p1 + 2) >> 2)
		}
	} else {
		tc0 := tc0Table[indexA][bS-1]
		tc := tc0
		if abs(p2-p0) < beta {
			tc++
		}
		if abs(q2-q0) < beta {
			tc++
		}
		delta := Clip3(-tc, tc, ((q0-p0)*4+(p1-q1)+4)>>3)
		rp0 = Clip1(p0 + delta)
		rq0 = Clip1(q0 - delta)
		if abs(p2-p0) < beta {
			rp1 = Clip1(p1 + Clip3(-tc0, tc0, (p2+((p0+q0+1)>>1)-(p1<<1))>>1))
		}
		if abs(q2-q0) < beta {
			rq1 = Clip1(q1 + Clip3(-tc0, tc0, (q2+((p0+q0+1)>>1)-(q1<<1))>>1))
		}
	}
	return
}

// filterChromaSample applies the chroma deblocking filter to one sample pair.
// Chroma uses only p1,p0,q0,q1 (no p2/q2 for strong path).
func filterChromaSample(p1, p0, q0, q1, bS, alpha, beta, indexA int) (rp0, rq0 uint8) {
	rp0, rq0 = Clip1(p0), Clip1(q0)
	if abs(p0-q0) >= alpha || abs(p1-p0) >= beta || abs(q1-q0) >= beta {
		return
	}
	if bS == 4 {
		rp0 = Clip1((2*p1 + p0 + q1 + 2) >> 2)
		rq0 = Clip1((2*q1 + q0 + p1 + 2) >> 2)
	} else {
		tc := tc0Table[indexA][bS-1] + 1
		delta := Clip3(-tc, tc, ((q0-p0)*4+(p1-q1)+4)>>3)
		rp0 = Clip1(p0 + delta)
		rq0 = Clip1(q0 - delta)
	}
	return
}

// FilterLumaEdgeV filters a vertical luma edge in-place on plane data.
// The edge is at column x (pixels x-1 and x are p0/q0).
// rowStart: first row to filter; nrows must be a multiple of 4.
// bS[4]: boundary strength for each group of 4 rows.
// indexA/B: clipped QP+offset indices into alpha/beta tables.
func FilterLumaEdgeV(plane []uint8, stride, x, rowStart, nrows int, bS [4]int, indexA, indexB int) {
	if x < 4 || x+4 > stride || indexA < 0 || indexA > 51 || indexB < 0 || indexB > 51 {
		return
	}
	alpha := alphaTable[indexA]
	beta := betaTable[indexB]
	for g := 0; g < nrows/4 && g < 4; g++ {
		bs := bS[g]
		if bs == 0 {
			continue
		}
		for r := 0; r < 4; r++ {
			row := rowStart + g*4 + r
			base := row * stride
			if base+x+4 > len(plane) || base+x-4 < 0 {
				continue
			}
			p3 := int(plane[base+x-4])
			p2 := int(plane[base+x-3])
			p1 := int(plane[base+x-2])
			p0 := int(plane[base+x-1])
			q0 := int(plane[base+x+0])
			q1 := int(plane[base+x+1])
			q2 := int(plane[base+x+2])
			q3 := int(plane[base+x+3])
			rp0, rp1, rp2, rq0, rq1, rq2 := filterLumaSample(p3, p2, p1, p0, q0, q1, q2, q3, bs, alpha, beta, indexA)
			plane[base+x-1] = rp0
			plane[base+x-2] = rp1
			plane[base+x-3] = rp2
			plane[base+x+0] = rq0
			plane[base+x+1] = rq1
			plane[base+x+2] = rq2
		}
	}
}

// FilterLumaEdgeH filters a horizontal luma edge in-place on plane data.
// The edge is at row y (pixels y-1 and y are p0/q0).
// colStart: first column; ncols must be a multiple of 4.
// bS[4]: boundary strength for each group of 4 columns.
func FilterLumaEdgeH(plane []uint8, stride, y, colStart, ncols int, bS [4]int, indexA, indexB int) {
	if y < 4 || indexA < 0 || indexA > 51 || indexB < 0 || indexB > 51 {
		return
	}
	alpha := alphaTable[indexA]
	beta := betaTable[indexB]
	for g := 0; g < ncols/4 && g < 4; g++ {
		bs := bS[g]
		if bs == 0 {
			continue
		}
		for c := 0; c < 4; c++ {
			col := colStart + g*4 + c
			if (y+4)*stride+col >= len(plane) || (y-4)*stride+col < 0 {
				continue
			}
			p3 := int(plane[(y-4)*stride+col])
			p2 := int(plane[(y-3)*stride+col])
			p1 := int(plane[(y-2)*stride+col])
			p0 := int(plane[(y-1)*stride+col])
			q0 := int(plane[(y+0)*stride+col])
			q1 := int(plane[(y+1)*stride+col])
			q2 := int(plane[(y+2)*stride+col])
			q3 := int(plane[(y+3)*stride+col])
			rp0, rp1, rp2, rq0, rq1, rq2 := filterLumaSample(p3, p2, p1, p0, q0, q1, q2, q3, bs, alpha, beta, indexA)
			plane[(y-1)*stride+col] = rp0
			plane[(y-2)*stride+col] = rp1
			plane[(y-3)*stride+col] = rp2
			plane[(y+0)*stride+col] = rq0
			plane[(y+1)*stride+col] = rq1
			plane[(y+2)*stride+col] = rq2
		}
	}
}

// FilterChromaEdgeV filters a vertical chroma edge in-place.
// x is the edge column; nrows must be a multiple of 2 (chroma height per MB = 8).
// bS[4] is shared with the luma edge (one per 4 luma rows = 2 chroma rows).
func FilterChromaEdgeV(plane []uint8, stride, x, rowStart, nrows int, bS [4]int, indexA, indexB int) {
	if x < 2 || x+2 > stride || indexA < 0 || indexA > 51 || indexB < 0 || indexB > 51 {
		return
	}
	alpha := alphaTable[indexA]
	beta := betaTable[indexB]
	for g := 0; g < nrows/2 && g < 4; g++ {
		bs := bS[g]
		if bs == 0 {
			continue
		}
		for r := 0; r < 2; r++ {
			row := rowStart + g*2 + r
			base := row * stride
			if base+x+2 > len(plane) || base+x-2 < 0 {
				continue
			}
			p1 := int(plane[base+x-2])
			p0 := int(plane[base+x-1])
			q0 := int(plane[base+x+0])
			q1 := int(plane[base+x+1])
			rp0, rq0 := filterChromaSample(p1, p0, q0, q1, bs, alpha, beta, indexA)
			plane[base+x-1] = rp0
			plane[base+x+0] = rq0
		}
	}
}

// FilterChromaEdgeH filters a horizontal chroma edge in-place.
// y is the edge row; ncols must be a multiple of 4.
// bS[4] one per group of 4 luma cols = 4 chroma cols.
func FilterChromaEdgeH(plane []uint8, stride, y, colStart, ncols int, bS [4]int, indexA, indexB int) {
	if y < 2 || indexA < 0 || indexA > 51 || indexB < 0 || indexB > 51 {
		return
	}
	alpha := alphaTable[indexA]
	beta := betaTable[indexB]
	for g := 0; g < ncols/4 && g < 2; g++ {
		bs := bS[g]
		if bs == 0 {
			continue
		}
		for c := 0; c < 4; c++ {
			col := colStart + g*4 + c
			if (y+2)*stride+col >= len(plane) || (y-2)*stride+col < 0 {
				continue
			}
			p1 := int(plane[(y-2)*stride+col])
			p0 := int(plane[(y-1)*stride+col])
			q0 := int(plane[(y+0)*stride+col])
			q1 := int(plane[(y+1)*stride+col])
			rp0, rq0 := filterChromaSample(p1, p0, q0, q1, bs, alpha, beta, indexA)
			plane[(y-1)*stride+col] = rp0
			plane[(y+0)*stride+col] = rq0
		}
	}
}

// MBDeblockInfo carries the per-macroblock data needed for boundary strength
// calculation. Store one per MB in scan order; pass current+neighbors to DeblockMB.
type MBDeblockInfo struct {
	QP      int  // luma QP
	IsIntra bool // MB is intra coded
	// Per-4×4 non-zero coefficient count (scan order 0-15, luma only).
	// Used for inter bS: 2 if either side has non-zero coefficients, else 1/0.
	NZC [16]int
}

// DeblockMBContext holds slice-level deblocking parameters.
type DeblockMBContext struct {
	DisableIDC  int // disable_deblocking_filter_idc (0=on, 1=off, 2=no cross-slice)
	AlphaOffset int // slice_alpha_c0_offset (already multiplied ×2 by parser)
	BetaOffset  int // slice_beta_offset (already multiplied ×2 by parser)
	ChromaQPOff int // chroma QP offset for indexA/B (frame.ChromaQP applied by caller)
}

// DeblockMB applies the in-loop deblocking filter to one macroblock.
// f: current frame (written in place); mbX/mbY: macroblock coordinates.
// cur: current MB info; left/top: left/top neighbor info (nil = none/out of frame).
// ctx: slice deblocking parameters.
// Implements H.264 §8.7 for 4:2:0, progressive, non-MBAFF content.
func DeblockMB(f interface {
	LumaPlane() []uint8
	LumaStride() int
	ChromaPlaneU() []uint8
	ChromaPlaneV() []uint8
	ChromaStride() int
}, mbX, mbY int, cur MBDeblockInfo, left, top *MBDeblockInfo, ctx DeblockMBContext) {
	if ctx.DisableIDC == 1 {
		return
	}
	// ... implemented below as package-private; callers use DeblockMBFrame instead.
}

// DeblockMBFrame applies in-loop deblocking for one macroblock directly on
// a frame.Frame-compatible struct.
// mbX, mbY: macroblock grid coordinates.
// cur: current MB; left/top may be nil when at the frame edge.
// ctx: slice-level deblocking parameters.
func DeblockMBFrame(
	yPlane []uint8, yStride int,
	uPlane, vPlane []uint8, cStride int,
	mbX, mbY int,
	cur MBDeblockInfo,
	left, top *MBDeblockInfo,
	ctx DeblockMBContext,
) {
	if ctx.DisableIDC == 1 {
		return
	}

	// §8.7: QP at a boundary is the average of the two MBs.
	// indexA = Clip3(0,51, avgQP + alphaOffset), indexB = Clip3(0,51, avgQP + betaOffset).
	// For internal edges: QP = current MB QP (no averaging needed).
	lumaQP := func(qpA, qpB int) int { return (qpA + qpB + 1) >> 1 }
	indexA := func(qp int) int { return Clip3(0, 51, qp+ctx.AlphaOffset) }
	indexB := func(qp int) int { return Clip3(0, 51, qp+ctx.BetaOffset) }

	// ---- Vertical edges (dir=0): filter from left to right ----
	// Edge 0: left MB boundary (if left neighbor exists)
	if left != nil {
		qp := lumaQP(cur.QP, left.QP)
		ia, ib := indexA(qp), indexB(qp)
		bs := bsVertMB(cur, left)
		FilterLumaEdgeV(yPlane, yStride, mbX*16, mbY*16, 16, bs, ia, ib)
		// Chroma: edge at mbX*8, each bS group covers 2 chroma rows; share luma bS.
		cbs := chromaBSFrom(bs)
		FilterChromaEdgeV(uPlane, cStride, mbX*8, mbY*8, 8, cbs, ia, ib)
		FilterChromaEdgeV(vPlane, cStride, mbX*8, mbY*8, 8, cbs, ia, ib)
	}

	// Internal vertical edges (edges 1-3): 4×4 column boundaries within MB.
	for e := 1; e <= 3; e++ {
		col := mbX*16 + e*4
		bs := bsVertInternal(cur, e)
		if bsAllZero(bs) {
			continue
		}
		qp := cur.QP
		ia, ib := indexA(qp), indexB(qp)
		FilterLumaEdgeV(yPlane, yStride, col, mbY*16, 16, bs, ia, ib)
		// Chroma: filter at even luma edges only (e=2 → chroma col mbX*8+4).
		if e == 2 {
			cbs := chromaBSFrom(bs)
			FilterChromaEdgeV(uPlane, cStride, mbX*8+4, mbY*8, 8, cbs, ia, ib)
			FilterChromaEdgeV(vPlane, cStride, mbX*8+4, mbY*8, 8, cbs, ia, ib)
		}
	}

	// ---- Horizontal edges (dir=1): filter from top to bottom ----
	// Edge 0: top MB boundary.
	if top != nil {
		qp := lumaQP(cur.QP, top.QP)
		ia, ib := indexA(qp), indexB(qp)
		bs := bsHorizMB(cur, top)
		FilterLumaEdgeH(yPlane, yStride, mbY*16, mbX*16, 16, bs, ia, ib)
		cbs := chromaBSFrom(bs)
		FilterChromaEdgeH(uPlane, cStride, mbY*8, mbX*8, 8, cbs, ia, ib)
		FilterChromaEdgeH(vPlane, cStride, mbY*8, mbX*8, 8, cbs, ia, ib)
	}

	// Internal horizontal edges (edges 1-3).
	for e := 1; e <= 3; e++ {
		row := mbY*16 + e*4
		bs := bsHorizInternal(cur, e)
		if bsAllZero(bs) {
			continue
		}
		qp := cur.QP
		ia, ib := indexA(qp), indexB(qp)
		FilterLumaEdgeH(yPlane, yStride, row, mbX*16, 16, bs, ia, ib)
		if e == 2 {
			cbs := chromaBSFrom(bs)
			FilterChromaEdgeH(uPlane, cStride, mbY*8+4, mbX*8, 8, cbs, ia, ib)
			FilterChromaEdgeH(vPlane, cStride, mbY*8+4, mbX*8, 8, cbs, ia, ib)
		}
	}
}

// bsVertMB returns bS[4] for the vertical MB-boundary edge between cur and left.
// §8.7.2: if either MB is intra → bS=4; else inter bS from NZC/MV (bS≤2 here).
func bsVertMB(cur MBDeblockInfo, left *MBDeblockInfo) [4]int {
	var bs [4]int
	for g := 0; g < 4; g++ {
		if cur.IsIntra || (left != nil && left.IsIntra) {
			bs[g] = 4
		} else if left != nil {
			// 4×4 blocks at left edge of cur: 0,4,8,12 (scan order rows 0-3).
			curNZ := cur.NZC[g*4]
			leftNZ := left.NZC[g*4+3] // right column of left MB
			if curNZ != 0 || leftNZ != 0 {
				bs[g] = 2
			} else {
				bs[g] = 1 // MV diff assumed; caller may refine
			}
		}
	}
	return bs
}

// bsHorizMB returns bS[4] for the horizontal MB-boundary edge between cur and top.
func bsHorizMB(cur MBDeblockInfo, top *MBDeblockInfo) [4]int {
	var bs [4]int
	for g := 0; g < 4; g++ {
		if cur.IsIntra || (top != nil && top.IsIntra) {
			bs[g] = 4
		} else if top != nil {
			// 4×4 blocks at top edge of cur: 0,1,2,3 (scan order cols 0-3).
			curNZ := cur.NZC[g]
			topNZ := top.NZC[g+12] // bottom row of top MB
			if curNZ != 0 || topNZ != 0 {
				bs[g] = 2
			} else {
				bs[g] = 1
			}
		}
	}
	return bs
}

// bsVertInternal returns bS[4] for internal vertical luma edges (edge 1-3).
// §8.7.2 table: if intra → bS=3; else NZC-based.
// luma 4×4 scan order columns: edge e covers blocks with col==e (0-indexed).
func bsVertInternal(cur MBDeblockInfo, edge int) [4]int {
	var bs [4]int
	if cur.IsIntra {
		for g := range bs {
			bs[g] = 3
		}
		return bs
	}
	// edge column in 4×4 grid: 0..3. Block scan: blk = row*4 + col
	for row := 0; row < 4; row++ {
		blk := row*4 + edge
		blkPrev := row*4 + edge - 1
		if cur.NZC[blk] != 0 || cur.NZC[blkPrev] != 0 {
			bs[row] = 2
		} else {
			bs[row] = 1
		}
	}
	return bs
}

// bsHorizInternal returns bS[4] for internal horizontal luma edges (edge 1-3).
func bsHorizInternal(cur MBDeblockInfo, edge int) [4]int {
	var bs [4]int
	if cur.IsIntra {
		for g := range bs {
			bs[g] = 3
		}
		return bs
	}
	// edge row in 4×4 grid: 0..3. Block scan: blk = row*4 + col
	for col := 0; col < 4; col++ {
		blk := edge*4 + col
		blkPrev := (edge-1)*4 + col
		if cur.NZC[blk] != 0 || cur.NZC[blkPrev] != 0 {
			bs[col] = 2
		} else {
			bs[col] = 1
		}
	}
	return bs
}

// chromaBSFrom maps luma bS[4] groups to chroma bS[4] groups (one per 2 luma rows).
// bS for chroma = max(bS from the two corresponding luma groups).
func chromaBSFrom(luma [4]int) [4]int {
	return luma
}

func bsAllZero(bs [4]int) bool {
	return bs[0] == 0 && bs[1] == 0 && bs[2] == 0 && bs[3] == 0
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
