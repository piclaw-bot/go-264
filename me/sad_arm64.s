//go:build arm64

#include "textflag.h"

// Scalar implementation on arm64 (NEON SAD needs WORD encoding).
// The Go compiler generates good code for the scalar loop.
// func SAD16x16_ASM_NEON(a, b *uint8, strideA, strideB int) uint32
TEXT ·SAD16x16_ASM_NEON(SB), NOSPLIT, $0-36
    MOVD a+0(FP), R0
    MOVD b+8(FP), R1
    MOVD strideA+16(FP), R2
    MOVD strideB+24(FP), R3
    MOVD $0, R5          // accumulator

    MOVD $16, R4          // row counter
loop:
    MOVD $16, R6          // col counter
    MOVD R0, R7           // row ptr a
    MOVD R1, R8           // row ptr b
col_loop:
    MOVBU (R7), R9
    MOVBU (R8), R10
    SUB R10, R9, R11
    // abs
    CMP $0, R11
    CNEG LT, R11, R11
    ADD R11, R5
    ADD $1, R7
    ADD $1, R8
    SUBS $1, R6
    BNE col_loop

    ADD R2, R0
    ADD R3, R1
    SUBS $1, R4
    BNE loop

    MOVW R5, ret+32(FP)
    RET
