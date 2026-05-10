package entropy

// CABAC residual coefficient decoding.
// Implements decode_cabac_residual_internal from FFmpeg h264_cabac.c.
// ITU-T H.264 §9.3.3.1 (significant-coeff-flag and coeff_abs_level binarization).
//
// NOTE: coded_block_flag (CBF) is intentionally NOT decoded here yet.
// Full CBF decode requires non_zero_count_cache / left_cbp / top_cbp neighbour
// tracking that is not yet implemented.  Until that tracking is in place the
// function reads the sig-flag map directly, using the first sig-flag position as
// a proxy for CBF.  The early-return guard (coeff_count==0 → return 0) provides
// the bit-budget protection needed without the explicit CBF bin.

// Context base offsets for significant coeff flags per category (non-field mode).
// Source: FFmpeg libavcodec/h264_cabac.c significant_coeff_flag_offset[0][cat]
var cabacSigCoeffFlagOffset = [14]int{
	105, 120, 134, 149, 152, 402, 484, 499, 513, 660, 528, 543, 557, 718,
}

// Context base offsets for last-significant coeff flags per category.
// Source: FFmpeg last_coeff_flag_offset[0][cat]
var cabacLastCoeffFlagOffset = [14]int{
	166, 181, 195, 210, 213, 417, 572, 587, 601, 690, 616, 631, 645, 748,
}

// Context base offsets for coeff absolute level minus 1 per category.
// Source: FFmpeg coeff_abs_level_m1_offset[cat]
var cabacCoeffAbsLevelOffset = [14]int{
	227, 237, 247, 257, 266, 426, 952, 962, 972, 708, 982, 992, 1002, 766,
}

// coeff_abs_level1_ctx: ctx offset for abs == 1 bin (relative to level base).
// Source: FFmpeg coeff_abs_level1_ctx[8]
var cabacLevel1Ctx = [8]uint8{1, 2, 3, 4, 0, 0, 0, 0}

// coeff_abs_levelgt1_ctx: ctx offset for abs > 1 (non-DC-422).
// Source: FFmpeg coeff_abs_levelgt1_ctx[0][8]
var cabacLevelGT1Ctx = [8]uint8{5, 5, 5, 5, 6, 7, 8, 9}

// coeff_abs_level_transition: node context update tables.
// [0] after abs==1, [1] after abs>1.
// Source: FFmpeg coeff_abs_level_transition[2][8]
var cabacLevelTransition = [2][8]uint8{
	{1, 2, 3, 3, 4, 5, 6, 7},
	{4, 4, 4, 4, 5, 6, 7, 7},
}

// significant_coeff_flag_offset_8x8 for cat5 (luma 8x8, non-field).
// Source: FFmpeg significant_coeff_flag_offset_8x8[0][63]
var cabacSigCoeff8x8 = [63]uint8{
	0, 1, 2, 3, 4, 5, 5, 4, 4, 3, 3, 4, 4, 4, 5, 5,
	4, 4, 4, 4, 3, 3, 6, 7, 7, 7, 8, 9, 10, 9, 8, 7,
	7, 6, 11, 12, 13, 11, 6, 7, 8, 9, 14, 10, 9, 8, 6, 11,
	12, 13, 11, 6, 9, 14, 10, 9, 11, 12, 13, 11, 14, 10, 12,
}

// h264LastCoeffFlagOffset8x8: last coeff flag offsets for 8x8 blocks.
// Source: FFmpeg ff_h264_last_coeff_flag_offset_8x8[63]
var cabacLastCoeff8x8 = [63]uint8{
	0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2,
	3, 3, 3, 3, 3, 3, 3, 3, 4, 4, 4, 4, 4, 4, 4, 4,
	5, 5, 5, 5, 6, 6, 6, 6, 7, 7, 7, 7, 8, 8, 8,
}

// DecodeCABACResidual decodes one residual block using CABAC context models.
//
//   - cat: residual category (0=luma DC, 1=luma AC/I16, 2=luma 4x4, 3=chroma DC, 4=chroma AC, 5=luma 8x8)
//   - maxCoeff: number of coefficients to decode (16 for 4x4, 4 for chroma DC, 15 for AC, 64 for 8x8)
//   - out: slice of length >= maxCoeff; coefficients are written in scan-position order
//   - nza, nzb: left/top neighbour nonzero flags for coded_block_flag context (0 or 1 each)
//
// Returns number of nonzero coefficients (totalCoeff).
// Decodes coded_block_flag first; if 0, returns 0 without reading more bins.
func (d *CABACDecoder) DecodeCABACResidual(models []CABACCtx, cat, maxCoeff int, out []int16, nza, nzb int) int {
	if d == nil || len(models) < 1024 || len(out) < maxCoeff {
		return 0
	}
	if cat < 0 || cat >= 14 {
		return 0
	}

	sigBase := cabacSigCoeffFlagOffset[cat]
	lastBase := cabacLastCoeffFlagOffset[cat]
	levelBase := cabacCoeffAbsLevelOffset[cat]

	is8x8 := cat == 5

	// ---- Step 0: coded_block_flag ----
	// CBF context: ctx = (nza>0) + 2*(nzb>0), base from cabacCBFBase[cat].
	// For cat 5 (8x8 DCT), CBF is not separately decoded per block.
	// Source: FFmpeg decode_cabac_residual_dc/nondc → get_cabac_cbf_ctx.
	if !is8x8 {
		cbfBase := 0
		switch cat {
		case 0:
			cbfBase = 85 // luma DC
		case 1:
			cbfBase = 89 // luma AC I16x16
		case 2:
			cbfBase = 93 // luma 4x4
		case 3:
			cbfBase = 97 // chroma DC
		case 4:
			cbfBase = 101 // chroma AC
		default:
			cbfBase = 93
		}
		cbfCtx := cbfBase + nza + 2*nzb
		if d.DecodeBin(&models[cbfCtx]) == 0 {
			return 0 // coded_block_flag = 0
		}
	}
	var index [64]int
	coeffCount := 0

	if is8x8 {
		for last := 0; last < 63; last++ {
			sigCtxIdx := sigBase + int(cabacSigCoeff8x8[last])
			if d.DecodeBin(&models[sigCtxIdx]) == 1 {
				index[coeffCount] = last
				coeffCount++
				lastCtxIdx := lastBase + int(cabacLastCoeff8x8[last])
				if d.DecodeBin(&models[lastCtxIdx]) == 1 {
					goto decode_levels
				}
			}
		}
		// Position 63 is added unconditionally (only safe after 8x8 CBF=1).
		index[coeffCount] = 63
		coeffCount++
	} else {
		// Scan positions 0..maxCoeff-2 via sig_ctx / last_ctx.
		for last := 0; last < maxCoeff-1; last++ {
			sigCtxIdx := sigBase + last
			if d.DecodeBin(&models[sigCtxIdx]) == 1 {
				index[coeffCount] = last
				coeffCount++
				lastCtxIdx := lastBase + last
				if d.DecodeBin(&models[lastCtxIdx]) == 1 {
					goto decode_levels
				}
			}
		}
		// Check whether position maxCoeff-1 is also significant.
		// Using sig_ctx check (not unconditional) to guard against context drift:
		// a wrongly decoded CBF=1 would consume extra level bins unconditionally;
		// the sig_ctx check lets a zero last position consume only one bin.
		sigCtxIdx := sigBase + (maxCoeff - 1)
		if d.DecodeBin(&models[sigCtxIdx]) == 1 {
			index[coeffCount] = maxCoeff - 1
			coeffCount++
		}
	}

decode_levels:
	if coeffCount == 0 {
		return 0
	}

	// ---- Step 2: coefficient levels in reverse scan order ----
	nodeCtx := 0
	for i := coeffCount - 1; i >= 0; i-- {
		pos := index[i]

		level1CtxIdx := levelBase + int(cabacLevel1Ctx[nodeCtx])
		if d.DecodeBin(&models[level1CtxIdx]) == 0 {
			// abs level == 1
			nodeCtx = int(cabacLevelTransition[0][nodeCtx])
			if d.DecodeBypass() == 1 {
				out[pos] = -1
			} else {
				out[pos] = 1
			}
		} else {
			// abs level >= 2
			coeffAbs := 2
			gtCtxIdx := levelBase + int(cabacLevelGT1Ctx[nodeCtx])
			nodeCtx = int(cabacLevelTransition[1][nodeCtx])
			for coeffAbs < 15 && d.DecodeBin(&models[gtCtxIdx]) == 1 {
				coeffAbs++
			}
			if coeffAbs >= 15 {
				j := 0
				for d.DecodeBypass() == 1 && j < 23 {
					j++
				}
				coeffAbs = 1
				for k := j - 1; k >= 0; k-- {
					coeffAbs = (coeffAbs << 1) | int(d.DecodeBypass())
				}
				coeffAbs += 14
			}
			if d.DecodeBypass() == 1 {
				out[pos] = int16(-coeffAbs)
			} else {
				out[pos] = int16(coeffAbs)
			}
		}
	}
	return coeffCount
}
