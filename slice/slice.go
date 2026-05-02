package slice

import "github.com/rcarmo/go-264/nal"

const (
	SliceTypeP  = 0
	SliceTypeB  = 1
	SliceTypeI  = 2
	SliceTypeSP = 3
	SliceTypeSI = 4
)

type Header struct {
	FirstMbInSlice     uint32
	SliceType          uint32
	PPSID              uint32
	FrameNum           uint32
	FieldPicFlag       bool
	BottomFieldFlag    bool
	IdrPicID           uint32
	PicOrderCntLsb     uint32
	RedundantPicCnt    uint32
	DirectSpatialMvPred bool
	NumRefIdxL0Active  uint32
	NumRefIdxL1Active  uint32
	CabacInitIDC       uint32
	SliceQPDelta       int32
	DisableDeblocking  int32
	SliceAlphaC0Offset int32
	SliceBetaOffset    int32
}

func ParseHeader(payload []byte, nalType uint8, sps *nal.SPS, pps *nal.PPS) (*Header, *nal.Reader) {
	r := nal.NewReader(payload)
	h := &Header{}

	h.FirstMbInSlice = r.ReadUE()
	h.SliceType = r.ReadUE()
	if h.SliceType >= 5 { h.SliceType -= 5 }
	h.PPSID = r.ReadUE()
	h.FrameNum = r.ReadBits(int(sps.Log2MaxFrameNum))

	if !sps.FrameMbsOnlyFlag {
		h.FieldPicFlag = r.ReadBool()
		if h.FieldPicFlag { h.BottomFieldFlag = r.ReadBool() }
	}
	if nalType == nal.TypeSliceIDR { h.IdrPicID = r.ReadUE() }
	if sps.PicOrderCntType == 0 { h.PicOrderCntLsb = r.ReadBits(int(sps.Log2MaxPocLsb)) }
	if pps.RedundantPicCntPresent { h.RedundantPicCnt = r.ReadUE() }

	if h.SliceType == SliceTypeB { h.DirectSpatialMvPred = r.ReadBool() }

	if h.SliceType == SliceTypeP || h.SliceType == SliceTypeB || h.SliceType == SliceTypeSP {
		if r.ReadBool() {
			h.NumRefIdxL0Active = r.ReadUE() + 1
			if h.SliceType == SliceTypeB { h.NumRefIdxL1Active = r.ReadUE() + 1 }
		} else {
			h.NumRefIdxL0Active = pps.NumRefIdxL0Active
			h.NumRefIdxL1Active = pps.NumRefIdxL1Active
		}
	}

	// Skip ref pic list mod, pred weight, dec ref pic marking for I-frame focus
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

func (h *Header) IsIntra() bool { return h.SliceType == SliceTypeI || h.SliceType == SliceTypeSI }
func (h *Header) QP(ppsQP int32) int32 { return ppsQP + h.SliceQPDelta }
