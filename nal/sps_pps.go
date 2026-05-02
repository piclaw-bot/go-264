package nal

// SPS (Sequence Parameter Set) — defines stream-level parameters.
// ITU-T H.264 §7.3.2.1

type SPS struct {
	ProfileIDC         uint8
	ConstraintFlags    uint8 // constraint_set0..5_flag packed
	LevelIDC           uint8
	SPSID              uint32
	ChromaFormatIDC    uint32 // 0=mono, 1=4:2:0, 2=4:2:2, 3=4:4:4
	BitDepthLuma       uint32
	BitDepthChroma     uint32
	Log2MaxFrameNum    uint32
	PicOrderCntType    uint32
	Log2MaxPocLsb      uint32
	MaxNumRefFrames    uint32
	PicWidthInMbs      uint32 // width = PicWidthInMbs * 16
	PicHeightInMapUnits uint32
	FrameMbsOnlyFlag   bool
	Direct8x8Inference bool
	FrameCropping      bool
	CropLeft, CropRight, CropTop, CropBottom uint32

	// Derived
	Width  int
	Height int
}

// ParseSPS parses a Sequence Parameter Set from NAL payload.
func ParseSPS(payload []byte) (*SPS, error) {
	r := NewReader(payload)
	s := &SPS{}

	s.ProfileIDC = r.ReadU8()
	s.ConstraintFlags = r.ReadU8()
	s.LevelIDC = r.ReadU8()
	s.SPSID = r.ReadUE()

	// High profile extensions
	if s.ProfileIDC == 100 || s.ProfileIDC == 110 || s.ProfileIDC == 122 ||
		s.ProfileIDC == 244 || s.ProfileIDC == 44 || s.ProfileIDC == 83 ||
		s.ProfileIDC == 86 || s.ProfileIDC == 118 || s.ProfileIDC == 128 {
		s.ChromaFormatIDC = r.ReadUE()
		if s.ChromaFormatIDC == 3 {
			r.ReadBit() // separate_colour_plane_flag
		}
		s.BitDepthLuma = r.ReadUE() + 8
		s.BitDepthChroma = r.ReadUE() + 8
		r.ReadBit() // qpprime_y_zero_transform_bypass_flag
		if r.ReadBool() { // seq_scaling_matrix_present_flag
			n := 8
			if s.ChromaFormatIDC == 3 {
				n = 12
			}
			for i := 0; i < n; i++ {
				if r.ReadBool() { // seq_scaling_list_present_flag
					skipScalingList(r, i < 6)
				}
			}
		}
	} else {
		s.ChromaFormatIDC = 1
		s.BitDepthLuma = 8
		s.BitDepthChroma = 8
	}

	s.Log2MaxFrameNum = r.ReadUE() + 4
	s.PicOrderCntType = r.ReadUE()

	if s.PicOrderCntType == 0 {
		s.Log2MaxPocLsb = r.ReadUE() + 4
	} else if s.PicOrderCntType == 1 {
		r.ReadBit()  // delta_pic_order_always_zero_flag
		r.ReadSE()   // offset_for_non_ref_pic
		r.ReadSE()   // offset_for_top_to_bottom_field
		n := r.ReadUE()
		for i := uint32(0); i < n; i++ {
			r.ReadSE() // offset_for_ref_frame
		}
	}

	s.MaxNumRefFrames = r.ReadUE()
	r.ReadBit() // gaps_in_frame_num_value_allowed_flag

	s.PicWidthInMbs = r.ReadUE() + 1
	s.PicHeightInMapUnits = r.ReadUE() + 1
	s.FrameMbsOnlyFlag = r.ReadBool()

	if !s.FrameMbsOnlyFlag {
		r.ReadBit() // mb_adaptive_frame_field_flag
	}
	s.Direct8x8Inference = r.ReadBool()

	s.FrameCropping = r.ReadBool()
	if s.FrameCropping {
		s.CropLeft = r.ReadUE()
		s.CropRight = r.ReadUE()
		s.CropTop = r.ReadUE()
		s.CropBottom = r.ReadUE()
	}

	// Derived dimensions
	s.Width = int(s.PicWidthInMbs) * 16
	s.Height = int(s.PicHeightInMapUnits) * 16
	if s.FrameMbsOnlyFlag {
		// already correct
	} else {
		s.Height *= 2
	}
	if s.FrameCropping {
		cropUnitX := uint32(1)
		cropUnitY := uint32(1)
		if s.ChromaFormatIDC == 1 {
			cropUnitX = 2
			cropUnitY = 2
		} else if s.ChromaFormatIDC == 2 {
			cropUnitX = 2
		}
		s.Width -= int((s.CropLeft + s.CropRight) * cropUnitX)
		s.Height -= int((s.CropTop + s.CropBottom) * cropUnitY)
	}

	return s, nil
}

func skipScalingList(r *Reader, is4x4 bool) {
	size := 16
	if !is4x4 {
		size = 64
	}
	lastScale := int32(8)
	nextScale := int32(8)
	for j := 0; j < size; j++ {
		if nextScale != 0 {
			delta := r.ReadSE()
			nextScale = (lastScale + delta + 256) % 256
		}
		if nextScale != 0 {
			lastScale = nextScale
		}
	}
}

// PPS (Picture Parameter Set) — defines picture-level parameters.
// ITU-T H.264 §7.3.2.2

type PPS struct {
	PPSID                    uint32
	SPSID                    uint32
	EntropyCodingMode        uint32 // 0=CAVLC, 1=CABAC
	BottomFieldPicOrderInFrame bool
	NumSliceGroups           uint32
	NumRefIdxL0Active        uint32
	NumRefIdxL1Active        uint32
	WeightedPred             bool
	WeightedBipredIDC        uint32
	PicInitQP                int32
	PicInitQS                int32
	ChromaQPIndexOffset      int32
	DeblockingFilterControl  bool
	ConstrainedIntraPred     bool
	RedundantPicCntPresent   bool

	// High profile
	Transform8x8Mode         bool
	SecondChromaQPIndexOffset int32
}

// ParsePPS parses a Picture Parameter Set from NAL payload.
func ParsePPS(payload []byte) (*PPS, error) {
	r := NewReader(payload)
	p := &PPS{}

	p.PPSID = r.ReadUE()
	p.SPSID = r.ReadUE()
	p.EntropyCodingMode = r.ReadUE()
	p.BottomFieldPicOrderInFrame = r.ReadBool()
	p.NumSliceGroups = r.ReadUE() + 1

	if p.NumSliceGroups > 1 {
		// slice_group_map_type parsing — skip for now (rare)
		return p, nil
	}

	p.NumRefIdxL0Active = r.ReadUE() + 1
	p.NumRefIdxL1Active = r.ReadUE() + 1
	p.WeightedPred = r.ReadBool()
	p.WeightedBipredIDC = r.ReadBits(2)
	p.PicInitQP = int32(r.ReadSE()) + 26
	p.PicInitQS = int32(r.ReadSE()) + 26
	p.ChromaQPIndexOffset = r.ReadSE()
	p.DeblockingFilterControl = r.ReadBool()
	p.ConstrainedIntraPred = r.ReadBool()
	p.RedundantPicCntPresent = r.ReadBool()

	// High profile extensions (if more data)
	if !r.EOF() {
		p.Transform8x8Mode = r.ReadBool()
		if r.ReadBool() { // pic_scaling_matrix_present_flag
			n := 6
			if p.Transform8x8Mode {
				n = 8
			}
			for i := 0; i < n; i++ {
				if r.ReadBool() {
					skipScalingList(r, i < 6)
				}
			}
		}
		p.SecondChromaQPIndexOffset = r.ReadSE()
	} else {
		p.SecondChromaQPIndexOffset = p.ChromaQPIndexOffset
	}

	return p, nil
}
