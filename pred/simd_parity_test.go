package pred

import "testing"

func TestIntraPred16x16ASMParity(t *testing.T) {
	var top, left [16]uint8
	for i := 0; i < 16; i++ {
		top[i] = uint8(3*i + 17)
		left[i] = uint8(5*i + 23)
	}

	t.Run("DC", func(t *testing.T) {
		var asm, scalar [256]uint8
		dc := uint8(91)
		IntraPred16x16DC_ASM(&asm[0], dc)
		for i := range scalar {
			scalar[i] = dc
		}
		if asm != scalar {
			t.Fatalf("DC ASM mismatch")
		}
	})

	t.Run("Vertical", func(t *testing.T) {
		var asm, scalar [256]uint8
		IntraPred16x16V_ASM(&asm[0], &top[0])
		for y := 0; y < 16; y++ {
			copy(scalar[y*16:(y+1)*16], top[:])
		}
		if asm != scalar {
			t.Fatalf("Vertical ASM mismatch")
		}
	})

	t.Run("Horizontal", func(t *testing.T) {
		var asm, scalar [256]uint8
		IntraPred16x16H_ASM(&asm[0], &left[0])
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				scalar[y*16+x] = left[y]
			}
		}
		if asm != scalar {
			t.Fatalf("Horizontal ASM mismatch")
		}
	})
}

func TestInterPred16x16CopyASMParity(t *testing.T) {
	const srcStride = 24
	const dstStride = 20
	src := make([]uint8, srcStride*20)
	for i := range src {
		src[i] = uint8((i*7 + 11) & 0xff)
	}
	var asm, scalar [dstStride * 16]uint8

	InterPred16x16Copy_ASM(&asm[0], &src[2*srcStride+3], dstStride, srcStride)
	for y := 0; y < 16; y++ {
		copy(scalar[y*dstStride:y*dstStride+16], src[(2+y)*srcStride+3:(2+y)*srcStride+3+16])
	}
	if asm != scalar {
		t.Fatalf("InterPred16x16Copy_ASM mismatch")
	}
}
