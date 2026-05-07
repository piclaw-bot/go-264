//go:build !amd64 && !arm64

package transform

import "unsafe"

func IDCT8x8_ASM(block *int16) { IDCT8x8(unsafe.Slice(block, 64)) }
func DCT8x8_ASM(block *int16)  { DCT8x8(unsafe.Slice(block, 64)) }
