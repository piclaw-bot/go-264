//go:build !amd64

package me

var hasSSE2 = false

func SAD16x16_ASM(a, b *uint8, strideA, strideB int) uint32 {
	panic("SSE2 not available")
}
