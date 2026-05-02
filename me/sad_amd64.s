//go:build amd64

#include "textflag.h"

// func SAD16x16_ASM(a, b *uint8, strideA, strideB int) uint32
// Uses PSADBW (SSE2) for 16-byte absolute difference + horizontal sum.
// PSADBW computes SAD of 16 unsigned bytes in one instruction.
TEXT ·SAD16x16_ASM(SB), NOSPLIT, $0-36
    MOVQ a+0(FP), SI        // a pointer
    MOVQ b+8(FP), DI        // b pointer
    MOVQ strideA+16(FP), R8 // stride A
    MOVQ strideB+24(FP), R9 // stride B

    PXOR X0, X0             // accumulator = 0

    // Process 16 rows, each 16 bytes wide
    // MOVOU loads 16 bytes unaligned
    // PSADBW computes SAD of X1 vs X2, result in low 16 bits of each 64-bit lane

    // Row 0
    MOVOU (SI), X1
    MOVOU (DI), X2
    PSADBW X2, X1           // X1 = |a[i]-b[i]| summed per 8-byte lane
    PADDQ X1, X0
    ADDQ R8, SI; ADDQ R9, DI

    // Row 1
    MOVOU (SI), X1; MOVOU (DI), X2; PSADBW X2, X1; PADDQ X1, X0
    ADDQ R8, SI; ADDQ R9, DI

    // Row 2
    MOVOU (SI), X1; MOVOU (DI), X2; PSADBW X2, X1; PADDQ X1, X0
    ADDQ R8, SI; ADDQ R9, DI

    // Row 3
    MOVOU (SI), X1; MOVOU (DI), X2; PSADBW X2, X1; PADDQ X1, X0
    ADDQ R8, SI; ADDQ R9, DI

    // Row 4
    MOVOU (SI), X1; MOVOU (DI), X2; PSADBW X2, X1; PADDQ X1, X0
    ADDQ R8, SI; ADDQ R9, DI

    // Row 5
    MOVOU (SI), X1; MOVOU (DI), X2; PSADBW X2, X1; PADDQ X1, X0
    ADDQ R8, SI; ADDQ R9, DI

    // Row 6
    MOVOU (SI), X1; MOVOU (DI), X2; PSADBW X2, X1; PADDQ X1, X0
    ADDQ R8, SI; ADDQ R9, DI

    // Row 7
    MOVOU (SI), X1; MOVOU (DI), X2; PSADBW X2, X1; PADDQ X1, X0
    ADDQ R8, SI; ADDQ R9, DI

    // Row 8
    MOVOU (SI), X1; MOVOU (DI), X2; PSADBW X2, X1; PADDQ X1, X0
    ADDQ R8, SI; ADDQ R9, DI

    // Row 9
    MOVOU (SI), X1; MOVOU (DI), X2; PSADBW X2, X1; PADDQ X1, X0
    ADDQ R8, SI; ADDQ R9, DI

    // Row 10
    MOVOU (SI), X1; MOVOU (DI), X2; PSADBW X2, X1; PADDQ X1, X0
    ADDQ R8, SI; ADDQ R9, DI

    // Row 11
    MOVOU (SI), X1; MOVOU (DI), X2; PSADBW X2, X1; PADDQ X1, X0
    ADDQ R8, SI; ADDQ R9, DI

    // Row 12
    MOVOU (SI), X1; MOVOU (DI), X2; PSADBW X2, X1; PADDQ X1, X0
    ADDQ R8, SI; ADDQ R9, DI

    // Row 13
    MOVOU (SI), X1; MOVOU (DI), X2; PSADBW X2, X1; PADDQ X1, X0
    ADDQ R8, SI; ADDQ R9, DI

    // Row 14
    MOVOU (SI), X1; MOVOU (DI), X2; PSADBW X2, X1; PADDQ X1, X0
    ADDQ R8, SI; ADDQ R9, DI

    // Row 15
    MOVOU (SI), X1; MOVOU (DI), X2; PSADBW X2, X1; PADDQ X1, X0

    // Horizontal reduction: X0 has two 64-bit partial sums
    MOVOA X0, X1
    PSRLO $8, X1            // shift high 64-bit lane to low
    PADDQ X1, X0            // sum both lanes
    MOVQ  X0, AX            // extract to GP register
    MOVL  AX, ret+32(FP)
    RET
