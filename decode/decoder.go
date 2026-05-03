package decode

// H.264 Baseline decoder — decodes Annex B bitstreams to YUV frames.

import (
	"fmt"

	"github.com/rcarmo/go-264/frame"
	"github.com/rcarmo/go-264/nal"
	"github.com/rcarmo/go-264/pred"
	"github.com/rcarmo/go-264/slice"
	"github.com/rcarmo/go-264/transform"
)


// H.264 4x4 block position within macroblock (inverse raster scan §6.4.3)
var blk4x4X = [16]int{0, 4, 0, 4, 8, 12, 8, 12, 0, 4, 0, 4, 8, 12, 8, 12}
var blk4x4Y = [16]int{0, 0, 4, 4, 0, 0, 4, 4, 8, 8, 12, 12, 8, 8, 12, 12}

// Decoder is an H.264 Baseline profile decoder.
type Decoder struct {
	SPS    map[uint32]*nal.SPS
	PPS    map[uint32]*nal.PPS
	DPB    *frame.DPB
	Frames []*frame.Frame // decoded output frames
}

// NewDecoder creates a new H.264 decoder.
func NewDecoder() *Decoder {
	return &Decoder{
		SPS: make(map[uint32]*nal.SPS),
		PPS: make(map[uint32]*nal.PPS),
		DPB: frame.NewDPB(16),
	}
}

// Decode processes an Annex B bitstream and returns decoded frames.
func (d *Decoder) Decode(data []byte) ([]*frame.Frame, error) {
	units := nal.SplitNALUnits(data)
	var frames []*frame.Frame

	for _, unit := range units {
		switch unit.Type {
		case nal.TypeSPS:
			sps, err := nal.ParseSPS(unit.Payload)
			if err != nil {
				return nil, fmt.Errorf("SPS: %w", err)
			}
			d.SPS[sps.SPSID] = sps

		case nal.TypePPS:
			pps, err := nal.ParsePPS(unit.Payload)
			if err != nil {
				return nil, fmt.Errorf("PPS: %w", err)
			}
			d.PPS[pps.PPSID] = pps

		case nal.TypeSliceIDR, nal.TypeSliceNonIDR:
			if unit.Type == nal.TypeSliceIDR {
				d.DPB.Flush()
			}
			f, err := d.decodeSlice(unit)
			if err != nil {
				return nil, fmt.Errorf("slice: %w", err)
			}
			if f != nil {
				frames = append(frames, f)
				d.DPB.Add(f)
			}

		case nal.TypeSEI, nal.TypeAUD:
			// Skip
		}
	}

	d.Frames = append(d.Frames, frames...)
	return frames, nil
}

func (d *Decoder) decodeSlice(unit nal.Unit) (resultFrame *frame.Frame, resultErr error) {
	defer func() {
		if r := recover(); r != nil {
			resultErr = fmt.Errorf("decode panic: %v", r)
			resultFrame = nil
		}
	}()
	// Find PPS/SPS (peek at pps_id in slice header)
	// For simplicity, use first available PPS/SPS
	var pps *nal.PPS
	var sps *nal.SPS
	for _, p := range d.PPS {
		pps = p
		break
	}
	if pps == nil {
		return nil, fmt.Errorf("no PPS available")
	}
	for _, s := range d.SPS {
		sps = s
		break
	}
	if sps == nil {
		return nil, fmt.Errorf("no SPS available")
	}

	hdr, r := slice.ParseHeader(unit.Payload, unit.Type, sps, pps)
	// P/B frames need reference frames
	isIntra := hdr.IsIntra()

	qp := hdr.QP(pps.PicInitQP)
	f := frame.NewFrame(sps.Width, sps.Height)
	f.IsIDR = unit.Type == nal.TypeSliceIDR
	f.IsRef = unit.RefIDC > 0
	f.FrameNum = int(hdr.FrameNum)
	f.POC = int(hdr.PicOrderCntLsb)

	mbWidth := int(sps.PicWidthInMbs)
	mbHeight := int(sps.PicHeightInMapUnits)

	maxMBs := mbWidth * mbHeight
	if maxMBs > 10000 { maxMBs = 10000 } // safety limit
	for mbIdx := int(hdr.FirstMbInSlice); mbIdx < maxMBs; mbIdx++ {
		mbX := mbIdx % mbWidth
		mbY := mbIdx / mbWidth

		if isIntra {
			mb := slice.DecodeMBIntra(r, qp, pps.EntropyCodingMode, pps.Transform8x8Mode)
			d.reconstructMB(f, mb, mbX, mbY, int(qp), sps)
		} else if hdr.SliceType == slice.SliceTypeP {
			mbInter := slice.DecodeMBInter(r, qp, hdr.NumRefIdxL0Active)
			if mbInter.MBType >= 5 {
				mb := &slice.MBIntra{MBType: mbInter.MBType - 5}
				d.reconstructMB(f, mb, mbX, mbY, int(qp), sps)
			} else {
				d.reconstructMBInter(f, mbInter, mbX, mbY, int(qp))
			}
		} else {
			// B-slice
			mbBidi := slice.DecodeMBBidi(r, qp, hdr.NumRefIdxL0Active, hdr.NumRefIdxL1Active)
			if mbBidi.MBType >= slice.BMBTypeIntra {
				mb := &slice.MBIntra{MBType: mbBidi.MBType - slice.BMBTypeIntra}
				d.reconstructMB(f, mb, mbX, mbY, int(qp), sps)
			} else {
				d.reconstructMBBidi(f, mbBidi, mbX, mbY, int(qp))
			}
		}
	}

	return f, nil
}

func (d *Decoder) reconstructMB(f *frame.Frame, mb *slice.MBIntra, mbX, mbY int, qp int, sps *nal.SPS) {
	if mb.MBType >= 1 && mb.MBType <= 24 {
		// I_16x16: predict whole 16x16 block
		d.reconstruct16x16(f, mb, mbX, mbY, qp)
	} else if mb.MBType == 0 {
		// I_NxN: predict each 4x4 block
		d.reconstruct4x4(f, mb, mbX, mbY, qp)
	}
	// I_PCM: raw samples (rare, skip)
}

func (d *Decoder) reconstruct16x16(f *frame.Frame, mb *slice.MBIntra, mbX, mbY, qp int) {
	mode := int(mb.Intra16x16PredMode)
	top := make([]uint8, 16)
	left := make([]uint8, 16)
	topLeft := uint8(128)
	if mbY > 0 {
		for x := 0; x < 16; x++ { top[x] = f.PixelY(mbX*16+x, mbY*16-1) }
	} else {
		for i := range top { top[i] = 128 }
	}
	if mbX > 0 {
		for y := 0; y < 16; y++ { left[y] = f.PixelY(mbX*16-1, mbY*16+y) }
	} else {
		for i := range left { left[i] = 128 }
	}
	if mbX > 0 && mbY > 0 { topLeft = f.PixelY(mbX*16-1, mbY*16-1) }

	predicted := make([]uint8, 256)
	pred.PredIntra16x16(predicted, mode, top, left, topLeft)

	// Hadamard DC transform
	var dcBlock [16]int16
	for i := 0; i < 16; i++ { dcBlock[i] = mb.Coeffs[i][0] }
	transform.Hadamard4x4DC(dcBlock[:], qp)

	cbpLuma := mb.CodedBlockPattern & 0xF
	for blkIdx := 0; blkIdx < 16; blkIdx++ {
		bx := blk4x4X[blkIdx]
		by := blk4x4Y[blkIdx]
		var block [16]int16
		block[0] = dcBlock[blkIdx]
		if cbpLuma != 0 {
			for j := 1; j < 16; j++ { block[j] = mb.Coeffs[blkIdx][j] }
			// Dequant AC only (DC already handled by Hadamard)
			qpDiv6 := uint(qp / 6)
			qpMod6 := qp % 6
			for j := 1; j < 16; j++ {
				if block[j] != 0 {
					v := int32(transform.DequantVTable()[qpMod6][transform.PosToVTable()[j]])
					block[j] = int16(int32(block[j]) * v << qpDiv6)
				}
			}
		}
		transform.IDCT4x4(block[:])
		for py := 0; py < 4; py++ {
			for px := 0; px < 4; px++ {
				v := int(predicted[(by+py)*16+(bx+px)]) + int(block[py*4+px])
				if v < 0 { v = 0 }
				if v > 255 { v = 255 }
				f.SetPixelY(mbX*16+bx+px, mbY*16+by+py, uint8(v))
			}
		}
	}
}

func (d *Decoder) reconstruct4x4(f *frame.Frame, mb *slice.MBIntra, mbX, mbY, qp int) {
	// For each 4x4 block in raster scan order
	for blkIdx := 0; blkIdx < 16; blkIdx++ {
		bx := blk4x4X[blkIdx]
		by := blk4x4Y[blkIdx]

		// Get neighbors
		top := make([]uint8, 4)
		topRight := make([]uint8, 4)
		left := make([]uint8, 4)
		topLeft := uint8(128)

		x0 := mbX*16 + bx
		y0 := mbY*16 + by

		for i := 0; i < 4; i++ {
			if y0 > 0 {
				top[i] = f.PixelY(x0+i, y0-1)
			} else {
				top[i] = 128
			}
			if x0 > 0 {
				left[i] = f.PixelY(x0-1, y0+i)
			} else {
				left[i] = 128
			}
		}
		for i := 0; i < 4; i++ {
			if y0 > 0 && x0+4+i < f.Width {
				topRight[i] = f.PixelY(x0+4+i, y0-1)
			} else {
				topRight[i] = top[3]
			}
		}
		if x0 > 0 && y0 > 0 {
			topLeft = f.PixelY(x0-1, y0-1)
		}

		// Compute predicted mode from neighbors (§8.3.1.1)
		predMode := 2 // DC default
		if bx > 0 || mbX > 0 {
			// Left neighbor mode (simplified: use DC if cross-MB)
			if bx > 0 {
				predMode = 2 // would need tracking per-block modes
			}
		}
		// For now: if prev flag (-1), use DC; if rem, use directly
		mode := 2
		rawMode := mb.IntraPredMode[blkIdx]
		if rawMode == -1 {
			mode = predMode // use predicted (DC for now)
		} else if rawMode >= 0 {
			// rem_intra_pred_mode: if rem < predicted, mode=rem, else mode=rem+1
			if int(rawMode) < predMode {
				mode = int(rawMode)
			} else {
				mode = int(rawMode) + 1
			}
		}
		if mode > 8 { mode = 2 } // clamp to valid range

		predicted := make([]uint8, 16)
		pred.PredIntra4x4(predicted, mode, top, topRight, left, topLeft)

		// Add residual
		block := mb.Coeffs[blkIdx]
		hasResidual := (mb.CodedBlockPattern & (1 << uint(blkIdx/4))) != 0
		if hasResidual {
			transform.Dequant4x4(block[:], qp)
			transform.IDCT4x4(block[:])
		}

		for py := 0; py < 4; py++ {
			for px := 0; px < 4; px++ {
				v := int(predicted[py*4+px]) + int(block[py*4+px])
				if v < 0 { v = 0 }
				if v > 255 { v = 255 }
				f.SetPixelY(x0+px, y0+py, uint8(v))
			}
		}
	}
}

func (d *Decoder) reconstructMBInter(f *frame.Frame, mb *slice.MBInter, mbX, mbY, qp int) {
	// Get reference frame
	ref := d.DPB.GetRef(0)
	if ref == nil || len(d.DPB.Frames) == 0 {
		// No reference available — fill with gray
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				f.SetPixelY(mbX*16+x, mbY*16+y, 128)
			}
		}
		return
	}
	if ref == nil {
		ref = d.DPB.Frames[len(d.DPB.Frames)-1]
	}

	// Motion compensation based on partition type
	switch mb.MBType {
	case slice.PMBTypeP16x16:
		// Single 16x16 partition
		mv := mb.MV[0]
		pred.InterPred16x16(
			make([]uint8, 256),
			ref.Y,
			ref.StrideY,
			pred.MotionVector{X: mv.X, Y: mv.Y},
		)
		// Write predicted + residual to output
		predicted := make([]uint8, 256)
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				srcX := int(mv.X>>2) + mbX*16 + x
				srcY := int(mv.Y>>2) + mbY*16 + y
				if srcX < 0 { srcX = 0 }
				if srcY < 0 { srcY = 0 }
				if srcX >= ref.Width { srcX = ref.Width - 1 }
				if srcY >= ref.Height { srcY = ref.Height - 1 }
				predicted[y*16+x] = ref.PixelY(srcX, srcY)
			}
		}
		// Dequant + IDCT residual blocks, then add to prediction
		cbpLuma := mb.CBP & 0xF
		for blkIdx := 0; blkIdx < 16; blkIdx++ {
			group := blkIdx / 4
			if cbpLuma&(1<<uint(group)) != 0 {
				block := mb.Coeffs[blkIdx]
				transform.Dequant4x4(block[:], qp)
				transform.IDCT4x4(block[:])
			}
		}
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				_ = blk4x4X
				// Simplified: use raster order for residual
				bi := (y/4)*4 + (x/4)
				py, px := y%4, x%4
				v := int(predicted[y*16+x]) + int(mb.Coeffs[bi][py*4+px])
				if v < 0 { v = 0 }
				if v > 255 { v = 255 }
				f.SetPixelY(mbX*16+x, mbY*16+y, uint8(v))
			}
		}

	default:
		// For other partition types, copy from reference with zero MV as fallback
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				srcX := mbX*16 + x
				srcY := mbY*16 + y
				if srcX < ref.Width && srcY < ref.Height {
					f.SetPixelY(srcX, srcY, ref.PixelY(srcX, srcY))
				}
			}
		}
	}
}

func (d *Decoder) reconstructMBBidi(f *frame.Frame, mb *slice.MBBidi, mbX, mbY, qp int) {
	// B-frame reconstruction: blend L0 and L1 predictions
	// For Direct mode (mb_type=0): use co-located MV from future reference
	// For L0/L1/Bi: use respective reference frames

	// Get reference frames
	var refL0, refL1 *frame.Frame
	if len(d.DPB.Frames) > 0 {
		refL0 = d.DPB.Frames[len(d.DPB.Frames)-1]
	}
	if len(d.DPB.Frames) > 1 {
		refL1 = d.DPB.Frames[len(d.DPB.Frames)-2]
	}
	if refL0 == nil {
		refL0 = f // self-reference fallback
	}
	if refL1 == nil {
		refL1 = refL0
	}

	// Simple implementation: copy from L0 reference (Direct/L0) or blend (Bi)
	predL0 := make([]uint8, 256)
	predL1 := make([]uint8, 256)

	mvL0 := mb.MVL0[0]
	mvL1 := mb.MVL1[0]

	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			sx0 := clampInt(mbX*16+x+int(mvL0.X>>2), 0, refL0.Width-1)
			sy0 := clampInt(mbY*16+y+int(mvL0.Y>>2), 0, refL0.Height-1)
			predL0[y*16+x] = refL0.PixelY(sx0, sy0)

			sx1 := clampInt(mbX*16+x+int(mvL1.X>>2), 0, refL1.Width-1)
			sy1 := clampInt(mbY*16+y+int(mvL1.Y>>2), 0, refL1.Height-1)
			predL1[y*16+x] = refL1.PixelY(sx1, sy1)
		}
	}

	// Blend and write
	blended := make([]uint8, 256)
	useBi := mb.MBType == slice.BMBTypeBi16x16 || mb.MBType == slice.BMBTypeDirect16x16
	if useBi {
		slice.BiPredBlend(blended, predL0, predL1, 256)
	} else if slice.BMBTypeL116x16 == mb.MBType {
		copy(blended, predL1)
	} else {
		copy(blended, predL0)
	}

	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			f.SetPixelY(mbX*16+x, mbY*16+y, blended[y*16+x])
		}
	}
}

// DecodedFrame is an alias for frame.Frame for CLI convenience.
type DecodedFrame = frame.Frame

func clampInt(v, lo, hi int) int {
	if v < lo { return lo }
	if v > hi { return hi }
	return v
}
