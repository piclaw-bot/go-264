//go:build !amd64

package transform

// Stubs for non-amd64 platforms.
var HasAVX2 = false

func IDCT4x4_AVX2(block *int16) { panic("AVX2 not available") }
func DCT4x4_AVX2(block *int16)  { panic("AVX2 not available") }
