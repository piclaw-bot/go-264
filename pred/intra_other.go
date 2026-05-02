//go:build !amd64 && !arm64

package pred

var HasSSE2 = false

func IntraPred16x16DC_ASM(pred *uint8, dc uint8)   { panic("no SSE2") }
func IntraPred16x16V_ASM(pred *uint8, top *uint8)   { panic("no SSE2") }
func IntraPred16x16H_ASM(pred *uint8, left *uint8)  { panic("no SSE2") }

func hasNEONPred() bool { return false }
func intraPred16x16DC_NEON(pred *uint8, dc uint8) {}
func intraPred16x16V_NEON(pred *uint8, top *uint8) {}
func intraPred16x16H_NEON(pred *uint8, left *uint8) {}
