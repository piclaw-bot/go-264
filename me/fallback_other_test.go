//go:build !amd64 && !arm64

package me

import "testing"

func TestSAD16x16ASMOtherInvalidStrideDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("SAD16x16_ASM panicked on invalid input: %v", r)
		}
	}()
	var a, b [256]uint8
	if got := SAD16x16_ASM(&a[0], &b[0], 15, 16); got != 0 {
		t.Fatalf("strideA < 16 got %d want 0", got)
	}
	if got := SAD16x16_ASM(&a[0], &b[0], 16, 15); got != 0 {
		t.Fatalf("strideB < 16 got %d want 0", got)
	}
	if got := SAD16x16_ASM(nil, &b[0], 16, 16); got != 0 {
		t.Fatalf("nil a got %d want 0", got)
	}
	if got := SAD16x16_ASM(&a[0], nil, 16, 16); got != 0 {
		t.Fatalf("nil b got %d want 0", got)
	}
}
