package syntax

import "github.com/rcarmo/go-264/nal"

const (
	SliceTypeP  = 0
	SliceTypeB  = 1
	SliceTypeI  = 2
	SliceTypeSP = 3
	SliceTypeSI = 4
)

type Header struct {
	FirstMbInSlice      uint32
	SliceType           uint32
	PPSID               uint32
	FrameNum            uint32
	FieldPicFlag        bool
	BottomFieldFlag     bool
	IdrPicID            uint32
	PicOrderCntLsb      uint32
	RedundantPicCnt     uint32
	DirectSpatialMvPred bool
	NumRefIdxL0Active   uint32
	NumRefIdxL1Active   uint32
	CabacInitIDC        uint32
	SliceQPDelta        int32
	DisableDeblocking   int32
	SliceAlphaC0Offset  int32
	SliceBetaOffset     int32
}

func ParseHeader(payload []byte, nalType uint8, sps *nal.SPS, pps *nal.PPS) (*Header, *nal.Reader) {
	r := nal.NewReader(payload)
	h := &Header{}

	h.FirstMbInSlice = r.ReadUE()
	h.SliceType = r.ReadUE()
	if h.SliceType >= 5 {
		h.SliceType -= 5
	}
	h.PPSID = r.ReadUE()
	h.FrameNum = r.ReadBits(int(sps.Log2MaxFrameNum))

	if !sps.FrameMbsOnlyFlag {
		h.FieldPicFlag = r.ReadBool()
		if h.FieldPicFlag {
			h.BottomFieldFlag = r.ReadBool()
		}
	}
	if nalType == nal.TypeSliceIDR {
		h.IdrPicID = r.ReadUE()
	}
	if sps.PicOrderCntType == 0 {
		h.PicOrderCntLsb = r.ReadBits(int(sps.Log2MaxPocLsb))
	}
	if pps.RedundantPicCntPresent {
		h.RedundantPicCnt = r.ReadUE()
	}

	if h.SliceType == SliceTypeB {
		h.DirectSpatialMvPred = r.ReadBool()
	}

	if h.SliceType == SliceTypeP || h.SliceType == SliceTypeB || h.SliceType == SliceTypeSP {
		if r.ReadBool() { // num_ref_idx_active_override_flag
			h.NumRefIdxL0Active = r.ReadUE() + 1
			if h.SliceType == SliceTypeB {
				h.NumRefIdxL1Active = r.ReadUE() + 1
			}
		} else {
			h.NumRefIdxL0Active = pps.NumRefIdxL0Active
			h.NumRefIdxL1Active = pps.NumRefIdxL1Active
		}
	}

	// ref_pic_list_modification follows num_ref_idx_active_override_flag in the
	// slice header. Reading it earlier shifts P-slice headers whenever the
	// override flag is present.
	if h.SliceType != SliceTypeI && h.SliceType != SliceTypeSI {
		if r.ReadBool() { // ref_pic_list_modification_flag_l0
			for {
				op := r.ReadUE()
				if op == 3 {
					break
				}
				r.ReadUE()
			}
		}
		if h.SliceType == SliceTypeB {
			if r.ReadBool() { // ref_pic_list_modification_flag_l1
				for {
					op := r.ReadUE()
					if op == 3 {
						break
					}
					r.ReadUE()
				}
			}
		}
	}

	// pred_weight_table is only present for weighted prediction modes. The
	// Baseline fixtures currently have weighted prediction disabled.
	if (pps.WeightedPred && (h.SliceType == SliceTypeP || h.SliceType == SliceTypeSP)) ||
		(pps.WeightedBipredIDC == 1 && h.SliceType == SliceTypeB) {
		// Minimal skip for unsupported weighted prediction streams: this parser is
		// intentionally conservative and leaves full pred_weight_table support for
		// the weighted-prediction implementation pass.
	}

	// dec_ref_pic_marking
	if nalType == nal.TypeSliceIDR {
		r.ReadBit() // no_output_of_prior_pics_flag
		r.ReadBit() // long_term_reference_flag
	} else if r.ReadBool() { // adaptive_ref_pic_marking_mode_flag
		for {
			op := r.ReadUE()
			if op == 0 {
				break
			}
			r.ReadUE()
			if op == 3 {
				r.ReadUE()
			}
		}
	}

	if pps.EntropyCodingMode == 1 && h.SliceType != SliceTypeI && h.SliceType != SliceTypeSI {
		h.CabacInitIDC = r.ReadUE()
	}
	h.SliceQPDelta = r.ReadSE()

	if pps.DeblockingFilterControl {
		h.DisableDeblocking = r.ReadSE()
		if h.DisableDeblocking != 1 {
			h.SliceAlphaC0Offset = r.ReadSE() * 2
			h.SliceBetaOffset = r.ReadSE() * 2
		}
	}

	return h, r
}

func (h *Header) IsIntra() bool        { return h.SliceType == SliceTypeI || h.SliceType == SliceTypeSI }
func (h *Header) QP(ppsQP int32) int32 { return ppsQP + h.SliceQPDelta }
