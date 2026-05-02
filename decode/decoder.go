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

func (d *Decoder) decodeSlice(unit nal.Unit) (*frame.Frame, error) {
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

	for mbIdx := int(hdr.FirstMbInSlice); mbIdx < mbWidth*mbHeight; mbIdx++ {
		mbX := mbIdx % mbWidth
		mbY := mbIdx / mbWidth

		if isIntra {
			mb := slice.DecodeMBIntra(r, qp, pps.EntropyCodingMode)
			d.reconstructMB(f, mb, mbX, mbY, int(qp), sps)
		} else {
			// P/B slice: peek at mb_type to decide intra vs inter
			mbInter := slice.DecodeMBInter(r, qp, hdr.NumRefIdxL0Active)
			if mbInter.MBType >= 5 {
				// Intra MB within P-slice
				mb := &slice.MBIntra{MBType: mbInter.MBType - 5}
				d.reconstructMB(f, mb, mbX, mbY, int(qp), sps)
			} else {
				d.reconstructMBInter(f, mbInter, mbX, mbY, int(qp))
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

	// Get neighbors
	top := make([]uint8, 16)
	left := make([]uint8, 16)
	topLeft := uint8(128) // default if unavailable

	if mbY > 0 {
		for x := 0; x < 16; x++ {
			top[x] = f.PixelY(mbX*16+x, mbY*16-1)
		}
	} else {
		for i := range top { top[i] = 128 }
	}
	if mbX > 0 {
		for y := 0; y < 16; y++ {
			left[y] = f.PixelY(mbX*16-1, mbY*16+y)
		}
	} else {
		for i := range left { left[i] = 128 }
	}
	if mbX > 0 && mbY > 0 {
		topLeft = f.PixelY(mbX*16-1, mbY*16-1)
	}

	// Predict
	predicted := make([]uint8, 256)
	pred.PredIntra16x16(predicted, mode, top, left, topLeft)

	// Add residual (if CBP indicates coded blocks)
	cbpLuma := mb.CodedBlockPattern & 0xF
	for by := 0; by < 4; by++ {
		for bx := 0; bx < 4; bx++ {
			blkIdx := by*4 + bx
			if cbpLuma != 0 {
				block := mb.Coeffs[blkIdx]
				transform.Dequant4x4(block[:], qp)
				transform.IDCT4x4(block[:])
				// Add residual to prediction
				for py := 0; py < 4; py++ {
					for px := 0; px < 4; px++ {
						x := bx*4 + px
						y := by*4 + py
						v := int(predicted[y*16+x]) + int(block[py*4+px])
						if v < 0 { v = 0 }
						if v > 255 { v = 255 }
						f.SetPixelY(mbX*16+x, mbY*16+y, uint8(v))
					}
				}
			} else {
				// No residual: write prediction directly
				for py := 0; py < 4; py++ {
					for px := 0; px < 4; px++ {
						x := bx*4 + px
						y := by*4 + py
						f.SetPixelY(mbX*16+x, mbY*16+y, predicted[y*16+x])
					}
				}
			}
		}
	}
}

func (d *Decoder) reconstruct4x4(f *frame.Frame, mb *slice.MBIntra, mbX, mbY, qp int) {
	// For each 4x4 block in raster scan order
	for blkIdx := 0; blkIdx < 16; blkIdx++ {
		bx := (blkIdx % 4) * 4
		by := (blkIdx / 4) * 4

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

		// Determine prediction mode
		mode := 2 // default DC
		if mb.IntraPredMode[blkIdx] >= 0 {
			mode = int(mb.IntraPredMode[blkIdx])
		}

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
				if srcX >= 0 && srcX < ref.Width && srcY >= 0 && srcY < ref.Height {
					predicted[y*16+x] = ref.PixelY(srcX, srcY)
				} else {
					predicted[y*16+x] = 128
				}
			}
		}
		// Add residual and write to frame
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				blkIdx := (y/4)*4 + (x / 4)
				py, px := y%4, x%4
				v := int(predicted[y*16+x]) + int(mb.Coeffs[blkIdx][py*4+px])
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
