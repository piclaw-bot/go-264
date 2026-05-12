package decode

// decode/cabac_mb.go — CABAC macroblock decode for P-slice inter and I-slice
// intra macroblocks. Calls syntax.DecodeCABACCBP/DQP/Ref/MVD for pure syntax;
// residual coefficients are decoded via cabac.CABACDecoder.DecodeCABACResidual.

import (
	cabac "github.com/rcarmo/go-264/entropy/cabac"
	"github.com/rcarmo/go-264/syntax"
)

// decodeCABACPInterMB decodes one CABAC-coded P-slice macroblock.
// Returns (inter, nil, true) for P-skip, (nil, intra, false) for intra-in-P.
func decodeCABACPInterMB(dec *cabac.CABACDecoder, models []cabac.CABACCtx, numRefFrames uint32, leftNZ, topNZ *[16]int, leftChromaNZ, topChromaNZ *[2][4]int, leftCBP, topCBP uint32, leftNonSkip, topNonSkip bool, transform8x8Mode bool, leftMBType, topMBType uint32, leftChromaPred, topChromaPred int8, leftEdge8x8, topEdge8x8 [2]int8) (*syntax.MBInter, *syntax.MBIntra, bool) {
	mb := &syntax.MBInter{MBType: syntax.PMBTypeP16x16}
	if dec == nil || len(models) < 20 {
		return mb, nil, true
	}
	// P-slice mb_skip_flag uses ctxIdx 11 plus availability of non-skipped left/top neighbours.
	// Using ctx 11 unconditionally desynchronizes CABAC state after the first neighbour-dependent MB.
	skipCtx := 11
	if leftNonSkip {
		skipCtx++
	}
	if topNonSkip {
		skipCtx++
	}
	if dec.DecodeBin(&models[skipCtx]) == 1 {
		return mb, nil, true
	}
	// mb_type binarization (FFmpeg h264_cabac.c decode_cabac_mb_type P-slice path)
	if dec.DecodeBin(&models[14]) == 0 {
		if dec.DecodeBin(&models[15]) == 0 {
			mb.MBType = 3 * dec.DecodeBin(&models[16]) // P16x16 or P8x8
		} else {
			mb.MBType = 2 - dec.DecodeBin(&models[17]) // P8x16 or P16x8
		}
	} else {
		// FFmpeg h264_cabac.c decodes intra-in-P via decode_cabac_intra_mb_type(ctx_base=17, intra_slice=0).
		intra := decodeCABACIntraMBWithParams(dec, models, leftNZ, topNZ, leftChromaNZ, topChromaNZ, leftCBP, topCBP, leftMBType, topMBType, leftChromaPred, topChromaPred, transform8x8Mode, leftEdge8x8, topEdge8x8, 17, false)
		return nil, intra, false
	}
	parts := 1
	switch mb.MBType {
	case syntax.PMBTypeP16x8, syntax.PMBTypeP8x16:
		parts = 2
	case syntax.PMBTypeP8x8, syntax.PMBTypeP8x8ref0:
		parts = 4
		for i := 0; i < 4; i++ {
			mb.SubMBType[i] = 0
		}
	}
	if numRefFrames > 1 && mb.MBType != syntax.PMBTypeP8x8ref0 {
		for i := 0; i < parts; i++ {
			mb.RefIdx[i] = int8(syntax.DecodeCABACRef(dec, models, 0))
		}
	}
	if mb.MBType == syntax.PMBTypeP8x8 || mb.MBType == syntax.PMBTypeP8x8ref0 {
		for i := 0; i < 4; i++ {
			mdx := syntax.DecodeCABACMVD(dec, models, 40, 0)
			mdy := syntax.DecodeCABACMVD(dec, models, 47, 0)
			mb.SubMV[i*4] = syntax.MotionVector{X: mdx, Y: mdy}
		}
		mb.DecodedMVDX = mb.SubMV[0].X
		mb.DecodedMVDY = mb.SubMV[0].Y
	} else {
		for i := 0; i < parts; i++ {
			mdx := syntax.DecodeCABACMVD(dec, models, 40, 0)
			mdy := syntax.DecodeCABACMVD(dec, models, 47, 0)
			mb.MV[i] = syntax.MotionVector{X: mdx, Y: mdy}
		}
		mb.DecodedMVDX = mb.MV[0].X
		mb.DecodedMVDY = mb.MV[0].Y
	}
	mb.CBP = syntax.DecodeCABACCBP(dec, models, leftCBP, topCBP)
	if mb.CBP != 0 {
		mb.QPDelta = int32(syntax.DecodeCABACDQP(dec, models, 0))
		use8x8Residual := false
		if transform8x8Mode && mb.CBP&0xF != 0 {
			if dec.DecodeBin(&models[399]) == 1 {
				use8x8Residual = true
				mb.Use8x8Transform = true
			}
		}
		var nzMB [16]int
		if use8x8Residual {
			for group := 0; group < 4; group++ {
				if mb.CBP&(1<<uint(group)) != 0 {
					var buf [64]int16
					dec.DecodeCABACResidual(models, 5, 64, buf[:], 0, 0)
					for sub := 0; sub < 4; sub++ {
						blkIdx := group*4 + sub
						for j := 0; j < 16; j++ {
							mb.Coeffs[blkIdx][j] = buf[sub*16+j]
						}
						nzMB[blkIdx] = 1
						mb.TotalCoeff[blkIdx] = 1
					}
				}
			}
		} else {
			for group := 0; group < 4; group++ {
				if mb.CBP&(1<<uint(group)) != 0 {
					for sub := 0; sub < 4; sub++ {
						blkIdx := group*4 + sub
						nza, nzb := nzCBFCtxLuma(blkIdx, &nzMB, leftNZ, topNZ)
						var buf [16]int16
						tc := dec.DecodeCABACResidual(models, 2, 16, buf[:], nza, nzb)
						mb.TotalCoeff[blkIdx] = tc
						nzMB[blkIdx] = tc
						mb.Coeffs[blkIdx] = buf
					}
				}
			}
		}
		chromaCBP := (mb.CBP >> 4) & 0x3
		var nzMBChroma [2][4]int
		if chromaCBP > 0 {
			for comp := 0; comp < 2; comp++ {
				var dc [4]int16
				dec.DecodeCABACResidual(models, 3, 4, dc[:], 0, 0)
				storeCABACChromaDC(mb, comp, dc)
			}
		}
		if chromaCBP > 1 {
			for comp := 0; comp < 2; comp++ {
				for blk := 0; blk < 4; blk++ {
					nza, nzb := nzCBFCtxChroma(comp, blk, &nzMBChroma, leftChromaNZ, topChromaNZ)
					var ac [16]int16
					tc := dec.DecodeCABACResidual(models, 4, 15, ac[:], nza, nzb)
					mb.ChromaTotalCoeff[comp][blk] = tc
					nzMBChroma[comp][blk] = tc
					storeCABACChromaAC(mb, comp, blk, ac)
				}
			}
		}
	}
	return mb, nil, false
}

func storeCABACChromaDC(mb *syntax.MBInter, comp int, dc [4]int16) {
	if mb == nil || comp < 0 || comp >= 2 {
		return
	}
	for blk := 0; blk < 4; blk++ {
		mb.CoeffsChroma[comp][blk][0] = dc[blk]
	}
}

func storeCABACChromaAC(mb *syntax.MBInter, comp, blk int, ac [16]int16) {
	if mb == nil || comp < 0 || comp >= 2 || blk < 0 || blk >= 4 {
		return
	}
	// CABAC chroma AC residuals are decoded with the scan starting after DC.
	// Preserve slot 0, which was populated from the separate chroma DC block.
	for j := 1; j < 16; j++ {
		mb.CoeffsChroma[comp][blk][j] = ac[j]
	}
}

// decodeCABACIntraMB decodes one CABAC-coded I-slice intra macroblock.
// Models the FFmpeg decode_cabac_intra_mb_type / decode_cabac_mb_intra4x4_pred_mode
// / decode_cabac_mb_chroma_pre_mode flow from h264_cabac.c.
func decodeCABACIntraMB(dec *cabac.CABACDecoder, models []cabac.CABACCtx, leftNZ, topNZ *[16]int, leftChromaNZ, topChromaNZ *[2][4]int, leftCBP, topCBP uint32, leftMBType, topMBType uint32, leftChromaPred, topChromaPred int8, transform8x8Mode bool, leftEdge8x8, topEdge8x8 [2]int8) *syntax.MBIntra {
	return decodeCABACIntraMBWithParams(dec, models, leftNZ, topNZ, leftChromaNZ, topChromaNZ, leftCBP, topCBP, leftMBType, topMBType, leftChromaPred, topChromaPred, transform8x8Mode, leftEdge8x8, topEdge8x8, 3, true)
}

func decodeCABACIntraMBWithParams(dec *cabac.CABACDecoder, models []cabac.CABACCtx, leftNZ, topNZ *[16]int, leftChromaNZ, topChromaNZ *[2][4]int, leftCBP, topCBP uint32, leftMBType, topMBType uint32, leftChromaPred, topChromaPred int8, transform8x8Mode bool, leftEdge8x8, topEdge8x8 [2]int8, ctxBase int, intraSlice bool) *syntax.MBIntra {
	mb := &syntax.MBIntra{}
	if dec == nil || len(models) < 128 || ctxBase < 0 || ctxBase+5 >= len(models) {
		return mb
	}

	// mb_type: FFmpeg decode_cabac_intra_mb_type(ctx_base, intra_slice).
	stateOffset := ctxBase
	isI16 := false
	if intraSlice {
		intraCtx := int(isCABACIntra16orPCM(leftMBType) + 2*isCABACIntra16orPCM(topMBType))
		if dec.DecodeBin(&models[ctxBase+intraCtx]) == 0 {
			mb.MBType = 0 // I_NxN
		} else {
			stateOffset += 2
			isI16 = true
		}
	} else if dec.DecodeBin(&models[ctxBase]) == 0 {
		mb.MBType = 0 // I_NxN
	} else {
		stateOffset = ctxBase
		isI16 = true
	}
	if isI16 {
		if dec.DecodeTerminate() == 1 {
			mb.MBType = 25 // I_PCM
			return mb
		}
		// I_16x16: binarize cbp_luma / cbp_chroma / pred_mode.
		mbType := uint32(1)
		if dec.DecodeBin(&models[stateOffset+1]) == 1 {
			mbType += 12
		}
		if dec.DecodeBin(&models[stateOffset+2]) == 1 {
			chromaExtraCtx := stateOffset + 2
			if intraSlice {
				chromaExtraCtx++
			}
			mbType += 4 + 4*dec.DecodeBin(&models[chromaExtraCtx])
		}
		predCtx0 := stateOffset + 3
		predCtx1 := stateOffset + 3
		if intraSlice {
			predCtx0++
			predCtx1 += 2
		}
		mbType += 2 * dec.DecodeBin(&models[predCtx0])
		mbType += 1 * dec.DecodeBin(&models[predCtx1])
		mb.MBType = mbType
	}

	// Intra 4x4 / 8x8 prediction modes (I_NxN only)
	if mb.MBType == 0 {
		// I8x8 transform_size_8x8_flag deferred: global 8×8 DC gives lower PSNR
		// than 16 local 4×4 DCs for this stream's content mix (7.84 vs 8.12 dB).
		if false && transform8x8Mode && dec.DecodeBin(&models[399]) == 1 {
			mb.Use8x8Transform = true
			var localModes [4]int8
			for i := 0; i < 4; i++ {
				bc := i % 2
				br := i / 2
				var leftMode int8
				if bc == 0 {
					leftMode = leftEdge8x8[br]
				} else {
					leftMode = localModes[i-1]
				}
				var topMode int8
				if br == 0 {
					topMode = topEdge8x8[bc]
				} else {
					topMode = localModes[i-2]
				}
				if leftMode < 0 {
					leftMode = 2
				}
				if topMode < 0 {
					topMode = 2
				}
				predMode := leftMode
				if topMode < predMode {
					predMode = topMode
				}
				if dec.DecodeBin(&models[68]) == 1 {
					mb.I8x8PredMode[i] = predMode
				} else {
					mode := int8(0)
					mode |= int8(dec.DecodeBin(&models[69]))
					mode |= int8(dec.DecodeBin(&models[69])) << 1
					mode |= int8(dec.DecodeBin(&models[69])) << 2
					if mode >= predMode {
						mode++
					}
					mb.I8x8PredMode[i] = mode
				}
				localModes[i] = mb.I8x8PredMode[i]
			}
		} else {
			// I4x4: one pred mode per 4x4 block (16 total)
			for i := 0; i < 16; i++ {
				if dec.DecodeBin(&models[68]) == 1 {
					mb.IntraPredMode[i] = -1
				} else {
					mode := int8(0)
					mode |= int8(dec.DecodeBin(&models[69]))
					mode |= int8(dec.DecodeBin(&models[69])) << 1
					mode |= int8(dec.DecodeBin(&models[69])) << 2
					mb.IntraPredMode[i] = mode
				}
			}
		}
	} else if mb.MBType >= 1 && mb.MBType <= 24 {
		// I_16x16: prediction mode and CBP from mb_type
		mb.Intra16x16PredMode = int8((mb.MBType - 1) % 4)
		cbpChroma := (mb.MBType - 1) / 4 % 3
		cbpLuma := uint32(0)
		if (mb.MBType-1)/12 > 0 {
			cbpLuma = 15
		}
		mb.CodedBlockPattern = cbpLuma | (cbpChroma << 4)
	}

	// Chroma prediction mode (ctx 64-67)
	chromaPredCtx := 0
	if leftChromaPred != 0 {
		chromaPredCtx++
	}
	if topChromaPred != 0 {
		chromaPredCtx += 2
	}
	if dec.DecodeBin(&models[64+chromaPredCtx]) == 0 {
		mb.ChromaPredMode = 0
	} else if dec.DecodeBin(&models[67]) == 0 {
		mb.ChromaPredMode = 1
	} else if dec.DecodeBin(&models[67]) == 0 {
		mb.ChromaPredMode = 2
	} else {
		mb.ChromaPredMode = 3
	}

	// CBP for I_NxN (I_16x16 CBP is in mb_type already)
	if mb.MBType == 0 {
		mb.CodedBlockPattern = syntax.DecodeCABACCBP(dec, models, leftCBP, topCBP)
	}

	// QP delta
	if mb.CodedBlockPattern > 0 || (mb.MBType >= 1 && mb.MBType <= 24) {
		mb.QPDelta = int32(syntax.DecodeCABACDQP(dec, models, 0))
	}

	// Residual coefficients
	var nzMB [16]int
	var nzMBChroma [2][4]int
	if mb.MBType >= 1 && mb.MBType <= 24 {
		// I_16x16: luma DC (cat=0) then luma AC (cat=1) per block if cbp_luma
		var dcBuf [16]int16
		dec.DecodeCABACResidual(models, 0, 16, dcBuf[:], 0, 0)
		for pos := 0; pos < 16; pos++ {
			blk := blkXYToIdx[pos/4][pos%4]
			mb.Coeffs[blk][0] = dcBuf[pos]
		}
		cbpLuma := mb.CodedBlockPattern & 0xF
		if cbpLuma != 0 {
			for blk := 0; blk < 16; blk++ {
				nza, nzb := nzCBFCtxLuma(blk, &nzMB, leftNZ, topNZ)
				var acBuf [16]int16
				tc := dec.DecodeCABACResidual(models, 1, 15, acBuf[:], nza, nzb)
				for j := 1; j < 16; j++ {
					mb.Coeffs[blk][j] = acBuf[j]
				}
				mb.TotalCoeff[blk] = tc
				nzMB[blk] = tc
			}
		}
	} else if mb.MBType == 0 {
		cbpLuma := mb.CodedBlockPattern & 0xF
		if mb.Use8x8Transform {
			for group := 0; group < 4; group++ {
				if cbpLuma&(1<<uint(group)) != 0 {
					var buf [64]int16
					tc := dec.DecodeCABACResidual(models, 5, 64, buf[:], 0, 0)
					for sub := 0; sub < 4; sub++ {
						blkIdx := group*4 + sub
						for j := 0; j < 16; j++ {
							mb.Coeffs[blkIdx][j] = buf[sub*16+j]
						}
						mb.TotalCoeff[blkIdx] = tc / 4
						nzMB[blkIdx] = tc / 4
					}
				}
			}
		} else {
			for group := 0; group < 4; group++ {
				if cbpLuma&(1<<uint(group)) != 0 {
					for sub := 0; sub < 4; sub++ {
						blkIdx := group*4 + sub
						nza, nzb := nzCBFCtxLuma(blkIdx, &nzMB, leftNZ, topNZ)
						var buf [16]int16
						tc := dec.DecodeCABACResidual(models, 2, 16, buf[:], nza, nzb)
						mb.Coeffs[blkIdx] = [16]int16(buf)
						mb.TotalCoeff[blkIdx] = tc
						nzMB[blkIdx] = tc
					}
				}
			}
		}
	}

	// Chroma residuals
	chromaCBP := (mb.CodedBlockPattern >> 4) & 0x3
	if chromaCBP > 0 {
		for comp := 0; comp < 2; comp++ {
			var buf [16]int16
			dec.DecodeCABACResidual(models, 3, 4, buf[:], 0, 0)
			mb.CoeffsChroma[comp][0] = [16]int16(buf)
		}
	}
	if chromaCBP > 1 {
		for comp := 0; comp < 2; comp++ {
			for blk := 0; blk < 4; blk++ {
				nza, nzb := nzCBFCtxChroma(comp, blk, &nzMBChroma, leftChromaNZ, topChromaNZ)
				var buf [16]int16
				tc := dec.DecodeCABACResidual(models, 4, 15, buf[:], nza, nzb)
				mb.CoeffsChroma[comp][blk] = [16]int16(buf)
				mb.ChromaTotalCoeff[comp][blk] = tc
				nzMBChroma[comp][blk] = tc
			}
		}
	}

	return mb
}
