//go:build arm64

package pred

//go:noescape
func IntraPred16x16DC_NEON(pred *uint8, dc uint8)

//go:noescape
func IntraPred16x16V_NEON(pred *uint8, top *uint8)

//go:noescape
func IntraPred16x16H_NEON(pred *uint8, left *uint8)

//go:noescape
func InterPred16x16Copy_NEON(dst *uint8, src *uint8, dstStride, srcStride int)

func init() { HasSSE2 = false } // not on arm64
var HasNEON = true

func hasNEONPred() bool { return true }
func intraPred16x16DC_NEON(pred *uint8, dc uint8) { IntraPred16x16DC_NEON(pred, dc) }
func intraPred16x16V_NEON(pred *uint8, top *uint8) { IntraPred16x16V_NEON(pred, top) }
func intraPred16x16H_NEON(pred *uint8, left *uint8) { IntraPred16x16H_NEON(pred, left) }
func IntraPred16x16DC_ASM(pred *uint8, dc uint8)   { IntraPred16x16DC_NEON(pred, dc) }
func IntraPred16x16V_ASM(pred *uint8, top *uint8)   { IntraPred16x16V_NEON(pred, top) }
func IntraPred16x16H_ASM(pred *uint8, left *uint8)  { IntraPred16x16H_NEON(pred, left) }
func InterPred16x16Copy_ASM(dst *uint8, src *uint8, dstStride, srcStride int) {
	InterPred16x16Copy_NEON(dst, src, dstStride, srcStride)
}
var HasSSE2 = false
