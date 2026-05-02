//go:build arm64

#include "textflag.h"

// func IntraPred16x16DC_NEON(pred *uint8, dc uint8)
TEXT ·IntraPred16x16DC_NEON(SB), NOSPLIT, $0-9
    MOVD pred+0(FP), R0
    MOVBU dc+8(FP), R1
    VDUP R1, V0.B16       // broadcast byte to all 16 lanes
    MOVD $16, R2
dc_loop:
    VST1 [V0.B16], (R0)
    ADD $16, R0
    SUBS $1, R2
    BNE dc_loop
    RET

// func IntraPred16x16V_NEON(pred *uint8, top *uint8)
TEXT ·IntraPred16x16V_NEON(SB), NOSPLIT, $0-16
    MOVD pred+0(FP), R0
    MOVD top+8(FP), R1
    VLD1 (R1), [V0.B16]   // load 16-byte top row
    MOVD $16, R2
v_loop:
    VST1 [V0.B16], (R0)
    ADD $16, R0
    SUBS $1, R2
    BNE v_loop
    RET

// func IntraPred16x16H_NEON(pred *uint8, left *uint8)
TEXT ·IntraPred16x16H_NEON(SB), NOSPLIT, $0-16
    MOVD pred+0(FP), R0
    MOVD left+8(FP), R1
    MOVD $16, R2
h_loop:
    MOVBU (R1), R3
    VDUP R3, V0.B16       // broadcast left[y] to all 16 lanes
    VST1 [V0.B16], (R0)
    ADD $16, R0
    ADD $1, R1
    SUBS $1, R2
    BNE h_loop
    RET

// func InterPred16x16Copy_NEON(dst *uint8, src *uint8, dstStride, srcStride int)
TEXT ·InterPred16x16Copy_NEON(SB), NOSPLIT, $0-32
    MOVD dst+0(FP), R0
    MOVD src+8(FP), R1
    MOVD dstStride+16(FP), R2
    MOVD srcStride+24(FP), R3
    MOVD $16, R4
copy_loop:
    VLD1 (R1), [V0.B16]
    VST1 [V0.B16], (R0)
    ADD R2, R0
    ADD R3, R1
    SUBS $1, R4
    BNE copy_loop
    RET
