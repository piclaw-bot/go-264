# Decoder pattern matrix

Current working checklist:

- [ ] Finish the decoder
- [ ] Validate frame correctness
- [ ] Validate SIMD and GPU paths
- [ ] Create a decoding pattern matrix
- [ ] Ensure test coverage

## Pattern coverage

| Pattern | Entropy | Profile | Current status | Validation target | Notes |
|---|---:|---:|---|---|---|
| I_16x16 solid grey | CAVLC | Baseline | **Good** | max diff <= 2 | `gray16.h264`, `dark64.h264` |
| I_NxN 4x4 intra | CAVLC | Baseline | **Partial** | PSNR >= 30 dB | Residual/prediction still diverges on `testsrc_bl.h264` |
| P_Skip | CAVLC | Baseline | **Partial** | no bitstream shift | `mb_skip_run` parsed; MV prediction still conservative zero-MV |
| P16x16 | CAVLC | Baseline | **Partial** | PSNR >= 30 dB | Residuals decoded, SIMD MC wired |
| P16x8/P8x16 | CAVLC | Baseline | **Partial** | PSNR >= 30 dB | Partition MC implemented, needs reference trace |
| P8x8/P8x8ref0 | CAVLC | Baseline | **Conservative** | PSNR >= 30 dB | Uses first MV per 8x8; sub-partition split pending |
| B slices | CAVLC/CABAC | Main/High | **Skeleton** | no panic + PSNR target | Direct/L0/L1/Bi blend path exists but incomplete residual/MV semantics |
| CABAC I/P/B | CABAC | Main/High | **Incomplete** | PSNR target | CABAC arithmetic core exists; macroblock syntax integration pending |
| 8x8 transform | CABAC/CAVLC | High | **Incomplete** | reference parity | IDCT SIMD wired; residual decode/reconstruction not fully 8x8-aware |
| Chroma 4:2:0 | CAVLC/CABAC | All | **Partial** | YUV PSNR | Chroma residual parsed for intra CAVLC; reconstruction pending; neutral 128 default now fixed |

## Next reference-driven fixes

1. Produce per-MB trace for `testsrc_bl.h264`: `mb_type`, `QP`, `CBP`, `nC`, coeff_token, total_zeros, run_before.
2. Compare against ffmpeg/JM trace for first divergent macroblock.
3. Wire cross-MB nC only after the trace matches within a macroblock.
4. Implement P_Skip MV prediction from neighbouring MVs instead of zero-MV fallback.
5. Add chroma reconstruction and YUV output correctness checks.
