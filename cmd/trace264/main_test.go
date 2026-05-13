package main

import (
	"testing"

	"github.com/rcarmo/go-264/nal"
	"github.com/rcarmo/go-264/syntax"
)

type testBitWriter struct {
	bits []uint8
}

func (w *testBitWriter) bit(v uint8) {
	w.bits = append(w.bits, v&1)
}

func (w *testBitWriter) ue(v uint32) {
	codeNum := v + 1
	bits := 0
	for tmp := codeNum; tmp > 0; tmp >>= 1 {
		bits++
	}
	for i := 0; i < bits-1; i++ {
		w.bit(0)
	}
	for i := bits - 1; i >= 0; i-- {
		w.bit(uint8((codeNum >> uint(i)) & 1))
	}
}

func (w *testBitWriter) se(v int32) {
	codeNum := uint32(0)
	if v > 0 {
		codeNum = uint32(v*2 - 1)
	} else {
		codeNum = uint32(-v * 2)
	}
	w.ue(codeNum)
}

func (w *testBitWriter) bytes() []byte {
	out := make([]byte, (len(w.bits)+7)/8)
	for i, b := range w.bits {
		if b != 0 {
			out[i/8] |= 1 << uint(7-(i%8))
		}
	}
	return out
}

func syntheticSliceUnit(payload []byte) nal.Unit {
	return nal.Unit{Type: nal.TypeSliceNonIDR, RefIDC: 1, Payload: payload}
}

func TestTraceMVCacheHelpersHandleInvalidInputs(t *testing.T) {
	writeBackInter4x4(nil, nil, 0, 0, 0, nil)
	writeBackInter4x4(make([]syntax.MotionVector, 1), nil, 4, 0, 0, &syntax.MBInter{MBType: syntax.PMBTypeP16x16})
	writeBackIntra4x4(make([]int8, 1), 4, -1, -1)
	fillMV4(make([]syntax.MotionVector, 1), nil, 4, 0, 0, 2, 2, syntax.MotionVector{}, 0)
	if _, ref := getMV4(nil, []int8{0}, 4, 0, 0); ref != -2 {
		t.Fatalf("getMV4 with short mv cache ref=%d want -2", ref)
	}
}

func TestTraceSliceBUsesBidiDecoder(t *testing.T) {
	var w testBitWriter
	w.ue(0)                             // first_mb_in_slice
	w.ue(syntax.SliceTypeB)             // slice_type
	w.ue(0)                             // pic_parameter_set_id
	w.bits = append(w.bits, 0, 0, 0, 0) // frame_num
	w.bit(1)                            // direct_spatial_mv_pred_flag
	w.bit(0)                            // num_ref_idx_active_override_flag
	w.bit(0)                            // ref_pic_list_modification_flag_l0
	w.bit(0)                            // ref_pic_list_modification_flag_l1
	w.bit(0)                            // adaptive_ref_pic_marking_mode_flag
	w.se(0)                             // slice_qp_delta
	w.ue(0)                             // mb_skip_run: next MB is coded
	w.ue(syntax.BMBTypeDirect16x16)     // direct B macroblock
	unit := syntheticSliceUnit(w.bytes())
	sps := map[uint32]*nal.SPS{0: {SPSID: 0, Log2MaxFrameNum: 4, PicOrderCntType: 2, FrameMbsOnlyFlag: true, ChromaFormatIDC: 1, PicWidthInMbs: 1, PicHeightInMapUnits: 1}}
	pps := map[uint32]*nal.PPS{0: {PPSID: 0, SPSID: 0, PicInitQP: 26, NumRefIdxL0Active: 1, NumRefIdxL1Active: 1}}
	if err := traceSlice(0, unit, sps, pps, 1, false); err != nil {
		t.Fatalf("traceSlice B returned %v", err)
	}
}

func TestTraceBContextHelpers(t *testing.T) {
	var nz [][16]int
	var chroma [][2][4]int
	mbInter := &syntax.MBBidi{TotalCoeff: [16]int{0: 2}, ChromaTotalCoeff: [2][4]int{{0: 1}}}
	traceBUpdateContexts(mbInter, nz, chroma, nil, 4, 0, 0)
	if len(nz) != 0 || len(chroma) != 0 {
		t.Fatal("short context slices should not be written")
	}
	nz = make([][16]int, 1)
	chroma = make([][2][4]int, 1)
	traceBUpdateContexts(mbInter, nz, chroma, make([]int8, 16), 4, 0, 0)
	if nz[0][0] != 2 || chroma[0][0][0] != 1 {
		t.Fatalf("B inter contexts not stored: nz=%v chroma=%v", nz[0], chroma[0])
	}
	mbIntra := &syntax.MBBidi{Intra: &syntax.MBIntra{TotalCoeff: [16]int{1: 3}, ChromaTotalCoeff: [2][4]int{{1: 4}}}}
	traceBUpdateContexts(mbIntra, nz, chroma, make([]int8, 16), 4, 0, 0)
	if nz[0][1] != 3 || chroma[0][0][1] != 4 {
		t.Fatalf("B intra contexts not stored: nz=%v chroma=%v", nz[0], chroma[0])
	}
}

func TestBTraceQPDeltaUsesIntraPayload(t *testing.T) {
	if got := bTraceQPDelta(nil); got != 0 {
		t.Fatalf("nil B MB qpd got %d want 0", got)
	}
	if got := bTraceQPDelta(&syntax.MBBidi{QPDelta: 3}); got != 3 {
		t.Fatalf("inter B qpd got %d want 3", got)
	}
	if got := bTraceQPDelta(&syntax.MBBidi{QPDelta: 3, Intra: &syntax.MBIntra{QPDelta: -2}}); got != -2 {
		t.Fatalf("B-intra qpd got %d want -2", got)
	}
}

func TestTraceSliceBConsumesSkipRun(t *testing.T) {
	var w testBitWriter
	w.ue(0)                             // first_mb_in_slice
	w.ue(syntax.SliceTypeB)             // slice_type
	w.ue(0)                             // pic_parameter_set_id
	w.bits = append(w.bits, 0, 0, 0, 0) // frame_num
	w.bit(1)                            // direct_spatial_mv_pred_flag
	w.bit(0)                            // num_ref_idx_active_override_flag
	w.bit(0)                            // ref_pic_list_modification_flag_l0
	w.bit(0)                            // ref_pic_list_modification_flag_l1
	w.bit(0)                            // adaptive_ref_pic_marking_mode_flag
	w.se(0)                             // slice_qp_delta
	w.ue(1)                             // mb_skip_run: one skipped direct B MB
	unit := syntheticSliceUnit(w.bytes())
	sps := map[uint32]*nal.SPS{0: {SPSID: 0, Log2MaxFrameNum: 4, PicOrderCntType: 2, FrameMbsOnlyFlag: true, ChromaFormatIDC: 1, PicWidthInMbs: 1, PicHeightInMapUnits: 1}}
	pps := map[uint32]*nal.PPS{0: {PPSID: 0, SPSID: 0, PicInitQP: 26, NumRefIdxL0Active: 1, NumRefIdxL1Active: 1}}
	if err := traceSlice(0, unit, sps, pps, 1, false); err != nil {
		t.Fatalf("traceSlice B skip returned %v", err)
	}
}

func TestUpdateQPMatchesDecoderModulo(t *testing.T) {
	cases := []struct {
		current, delta int
		want           int
	}{
		{26, 0, 26},
		{26, 1, 27},
		{26, -1, 25},
		{51, 1, 0},
		{0, -1, 51},
		{50, 5, 3},
		{10, -70, 44},
	}
	for _, tc := range cases {
		if got := updateQP(tc.current, tc.delta); got != tc.want {
			t.Fatalf("updateQP(%d,%d) got %d want %d", tc.current, tc.delta, got, tc.want)
		}
	}
}
