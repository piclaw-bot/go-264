package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/rcarmo/go-264/nal"
	"github.com/rcarmo/go-264/slice"
)

func main() {
	input := flag.String("i", "", "input Annex B H.264 bitstream")
	limit := flag.Int("limit", 64, "maximum macroblocks to trace per slice")
	flag.Parse()
	if *input == "" {
		fmt.Fprintln(os.Stderr, "usage: trace264 -i input.h264 [-limit N]")
		os.Exit(2)
	}
	data, err := os.ReadFile(*input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read: %v\n", err)
		os.Exit(1)
	}
	if err := trace(data, *limit); err != nil {
		fmt.Fprintf(os.Stderr, "trace: %v\n", err)
		os.Exit(1)
	}
}

func trace(data []byte, limit int) error {
	units := nal.SplitNALUnits(data)
	spsMap := map[uint32]*nal.SPS{}
	ppsMap := map[uint32]*nal.PPS{}
	for nalIdx, unit := range units {
		switch unit.Type {
		case nal.TypeSPS:
			sps, err := nal.ParseSPS(unit.Payload)
			if err != nil {
				return fmt.Errorf("nal %d SPS: %w", nalIdx, err)
			}
			spsMap[sps.SPSID] = sps
			fmt.Printf("nal=%d type=SPS id=%d size=%dx%d mbs=%dx%d\n", nalIdx, sps.SPSID, sps.Width, sps.Height, sps.PicWidthInMbs, sps.PicHeightInMapUnits)
		case nal.TypePPS:
			pps, err := nal.ParsePPS(unit.Payload)
			if err != nil {
				return fmt.Errorf("nal %d PPS: %w", nalIdx, err)
			}
			ppsMap[pps.PPSID] = pps
			fmt.Printf("nal=%d type=PPS id=%d sps=%d entropy=%d initQP=%d refsL0=%d\n", nalIdx, pps.PPSID, pps.SPSID, pps.EntropyCodingMode, pps.PicInitQP, pps.NumRefIdxL0Active)
		case nal.TypeSliceIDR, nal.TypeSliceNonIDR:
			if err := traceSlice(nalIdx, unit, spsMap, ppsMap, limit); err != nil {
				return err
			}
		}
	}
	return nil
}

func traceSlice(nalIdx int, unit nal.Unit, spsMap map[uint32]*nal.SPS, ppsMap map[uint32]*nal.PPS, limit int) error {
	peek := nal.NewReader(unit.Payload)
	_ = peek.ReadUE()
	_ = peek.ReadUE()
	ppsID := peek.ReadUE()
	pps := ppsMap[ppsID]
	if pps == nil {
		return fmt.Errorf("nal %d slice: PPS %d not available", nalIdx, ppsID)
	}
	sps := spsMap[pps.SPSID]
	if sps == nil {
		return fmt.Errorf("nal %d slice: SPS %d not available", nalIdx, pps.SPSID)
	}
	hdr, r := slice.ParseHeader(unit.Payload, unit.Type, sps, pps)
	mbWidth := int(sps.PicWidthInMbs)
	mbHeight := int(sps.PicHeightInMapUnits)
	maxMBs := mbWidth * mbHeight
	if limit > 0 && maxMBs > int(hdr.FirstMbInSlice)+limit {
		maxMBs = int(hdr.FirstMbInSlice) + limit
	}
	fmt.Printf("nal=%d type=%d slice=%d firstMB=%d frame=%d qp=%d payloadBits=%d\n", nalIdx, unit.Type, hdr.SliceType, hdr.FirstMbInSlice, hdr.FrameNum, hdr.QP(pps.PicInitQP), len(unit.Payload)*8)
	currentQP := int(hdr.QP(pps.PicInitQP))
	nzCtx := make([][16]int, mbWidth*mbHeight)
	chromaNZCtx := make([][2][4]int, mbWidth*mbHeight)
	mvCtx := make([]slice.MotionVector, mbWidth*mbHeight)
	refCtx := make([]int8, mbWidth*mbHeight)
	for i := range refCtx {
		refCtx[i] = -1
	}
	skipRun := 0
	decodeAfterSkipRun := false
	for mbIdx := int(hdr.FirstMbInSlice); mbIdx < maxMBs; mbIdx++ {
		mbX := mbIdx % mbWidth
		mbY := mbIdx / mbWidth
		var leftNZ, topNZ *[16]int
		var leftChromaNZ, topChromaNZ *[2][4]int
		if mbX > 0 {
			leftNZ = &nzCtx[mbIdx-1]
			leftChromaNZ = &chromaNZCtx[mbIdx-1]
		}
		if mbY > 0 {
			topNZ = &nzCtx[mbIdx-mbWidth]
			topChromaNZ = &chromaNZCtx[mbIdx-mbWidth]
		}
		start := r.Position()
		if hdr.IsIntra() {
			mb := slice.DecodeMBIntraCtxFull(r, int32(currentQP), pps.EntropyCodingMode, pps.Transform8x8Mode, leftNZ, topNZ, leftChromaNZ, topChromaNZ)
			currentQP = (currentQP + int(mb.QPDelta)%52 + 52) % 52
			nzCtx[mbIdx] = mb.TotalCoeff
			chromaNZCtx[mbIdx] = mb.ChromaTotalCoeff
			fmt.Printf("  mb=%04d x=%02d y=%02d bits=%d..%d type=I:%d cbp=%02x chromaMode=%d qpd=%d qp=%d tc=%v\n", mbIdx, mbX, mbY, start, r.Position(), mb.MBType, mb.CodedBlockPattern, mb.ChromaPredMode, mb.QPDelta, currentQP, mb.TotalCoeff)
			if mb.MBType > slice.MBTypeIPCM || mb.ChromaPredMode > 3 {
				fmt.Printf("  !! invalid intra syntax at mb=%d: mb_type=%d chroma_mode=%d nextBit=%d\n", mbIdx, mb.MBType, mb.ChromaPredMode, r.Position())
				return nil
			}
			continue
		}
		predMV := predictMBMV(mvCtx, refCtx, 0, mbIdx, mbX, mbY, mbWidth)
		if hdr.SliceType == slice.SliceTypeP && pps.EntropyCodingMode == 0 {
			if skipRun == 0 && !decodeAfterSkipRun {
				skipRun = int(r.ReadUE())
			}
			if skipRun > 0 {
				skipMV := predictSkipMV(mvCtx, predMV, mbIdx, mbX, mbY, mbWidth)
				fmt.Printf("  mb=%04d x=%02d y=%02d bits=%d..%d type=P_SKIP remainingSkip=%d qp=%d mv0=(%d,%d) ref0=0\n", mbIdx, mbX, mbY, start, r.Position(), skipRun-1, currentQP, skipMV.X, skipMV.Y)
				mvCtx[mbIdx] = skipMV
				refCtx[mbIdx] = 0
				skipRun--
				decodeAfterSkipRun = skipRun == 0
				continue
			}
			decodeAfterSkipRun = false
		}
		mb := slice.DecodeMBInterCtxFull(r, int32(currentQP), hdr.NumRefIdxL0Active, leftNZ, topNZ, leftChromaNZ, topChromaNZ)
		if mb.MBType >= slice.PMBTypeIntra {
			intra := slice.DecodeMBIntraCtxWithTypeFull(r, mb.MBType-slice.PMBTypeIntra, int32(currentQP), pps.EntropyCodingMode, pps.Transform8x8Mode, leftNZ, topNZ, leftChromaNZ, topChromaNZ)
			currentQP = (currentQP + int(intra.QPDelta)%52 + 52) % 52
			nzCtx[mbIdx] = intra.TotalCoeff
			chromaNZCtx[mbIdx] = intra.ChromaTotalCoeff
			refCtx[mbIdx] = -1
			fmt.Printf("  mb=%04d x=%02d y=%02d bits=%d..%d type=P:I:%d cbp=%02x chromaMode=%d qpd=%d qp=%d tc=%v\n", mbIdx, mbX, mbY, start, r.Position(), intra.MBType, intra.CodedBlockPattern, intra.ChromaPredMode, intra.QPDelta, currentQP, intra.TotalCoeff)
			if intra.MBType > slice.MBTypeIPCM || intra.ChromaPredMode > 3 {
				fmt.Printf("  !! invalid P-intra syntax at mb=%d: mb_type=%d chroma_mode=%d nextBit=%d\n", mbIdx, intra.MBType, intra.ChromaPredMode, r.Position())
				return nil
			}
			continue
		}
		rawMV0 := mb.MV[0]
		pred0 := predictMBMV(mvCtx, refCtx, mb.RefIdx[0], mbIdx, mbX, mbY, mbWidth)
		applyMVPredictors(mb, mvCtx, refCtx, mbIdx, mbX, mbY, mbWidth)
		currentQP = (currentQP + int(mb.QPDelta)%52 + 52) % 52
		nzCtx[mbIdx] = mb.TotalCoeff
		chromaNZCtx[mbIdx] = mb.ChromaTotalCoeff
		mvCtx[mbIdx] = mb.MV[0]
		refCtx[mbIdx] = mb.RefIdx[0]
		fmt.Printf("  mb=%04d x=%02d y=%02d bits=%d..%d type=P:%d cbp=%02x qpd=%d qp=%d mvd0=(%d,%d) pred0=(%d,%d) mv0=(%d,%d) ref0=%d tc=%v\n", mbIdx, mbX, mbY, start, r.Position(), mb.MBType, mb.CBP, mb.QPDelta, currentQP, rawMV0.X, rawMV0.Y, pred0.X, pred0.Y, mb.MV[0].X, mb.MV[0].Y, mb.RefIdx[0], mb.TotalCoeff)
	}
	return nil
}

func predictSkipMV(ctx []slice.MotionVector, pred slice.MotionVector, mbIdx, mbX, mbY, mbWidth int) slice.MotionVector {
	if mbX == 0 || mbY == 0 {
		return slice.MotionVector{}
	}
	left := ctx[mbIdx-1]
	top := ctx[mbIdx-mbWidth]
	if (left.X == 0 && left.Y == 0) || (top.X == 0 && top.Y == 0) {
		return slice.MotionVector{}
	}
	return pred
}

func predictMBMV(ctx []slice.MotionVector, refCtx []int8, targetRef int8, mbIdx, mbX, mbY, mbWidth int) slice.MotionVector {
	a, b, c, availA, availB, availC := neighbourMVs(ctx, refCtx, targetRef, mbIdx, mbX, mbY, mbWidth)
	return slice.PredictMV(a, b, c, availA, availB, availC)
}

func neighbourMVs(ctx []slice.MotionVector, refCtx []int8, targetRef int8, mbIdx, mbX, mbY, mbWidth int) (a, b, c slice.MotionVector, availA, availB, availC bool) {
	availA = mbX > 0 && refCtx[mbIdx-1] == targetRef
	availB = mbY > 0 && refCtx[mbIdx-mbWidth] == targetRef
	availC = mbY > 0 && mbX+1 < mbWidth && refCtx[mbIdx-mbWidth+1] == targetRef
	if availA {
		a = ctx[mbIdx-1]
	}
	if availB {
		b = ctx[mbIdx-mbWidth]
	}
	if availC {
		c = ctx[mbIdx-mbWidth+1]
	} else if mbY > 0 && mbX > 0 && refCtx[mbIdx-mbWidth-1] == targetRef {
		availC = true
		c = ctx[mbIdx-mbWidth-1]
	}
	return
}

func addMV(mv *slice.MotionVector, pred slice.MotionVector) {
	mv.X += pred.X
	mv.Y += pred.Y
}

func applyMVPredictors(mb *slice.MBInter, ctx []slice.MotionVector, refCtx []int8, mbIdx, mbX, mbY, mbWidth int) {
	switch mb.MBType {
	case slice.PMBTypeP16x16:
		addMV(&mb.MV[0], predictMBMV(ctx, refCtx, mb.RefIdx[0], mbIdx, mbX, mbY, mbWidth))
	case slice.PMBTypeP16x8:
		a0, b0, c0, availA0, availB0, availC0 := neighbourMVs(ctx, refCtx, mb.RefIdx[0], mbIdx, mbX, mbY, mbWidth)
		pred0 := slice.PredictMV(a0, b0, c0, availA0, availB0, availC0)
		if availB0 {
			pred0 = b0
		}
		a1, b1, c1, availA1, availB1, availC1 := neighbourMVs(ctx, refCtx, mb.RefIdx[1], mbIdx, mbX, mbY, mbWidth)
		pred1 := slice.PredictMV(a1, b1, c1, availA1, availB1, availC1)
		if availA1 {
			pred1 = a1
		}
		addMV(&mb.MV[0], pred0)
		addMV(&mb.MV[1], pred1)
	case slice.PMBTypeP8x16:
		a0, b0, c0, availA0, availB0, availC0 := neighbourMVs(ctx, refCtx, mb.RefIdx[0], mbIdx, mbX, mbY, mbWidth)
		pred0 := slice.PredictMV(a0, b0, c0, availA0, availB0, availC0)
		if availA0 {
			pred0 = a0
		}
		a1, b1, c1, availA1, availB1, availC1 := neighbourMVs(ctx, refCtx, mb.RefIdx[1], mbIdx, mbX, mbY, mbWidth)
		pred1 := slice.PredictMV(a1, b1, c1, availA1, availB1, availC1)
		if availC1 {
			pred1 = c1
		}
		addMV(&mb.MV[0], pred0)
		addMV(&mb.MV[1], pred1)
	case slice.PMBTypeP8x8, slice.PMBTypeP8x8ref0:
		for part := 0; part < 4; part++ {
			a, b, c, availA, availB, availC := neighbourMVs(ctx, refCtx, mb.RefIdx[part], mbIdx, mbX, mbY, mbWidth)
			subPred := slice.PredictMV(a, b, c, availA, availB, availC)
			for i := 0; i < 4; i++ {
				addMV(&mb.SubMV[part*4+i], subPred)
			}
		}
		mb.MV[0] = mb.SubMV[0]
	}
}
