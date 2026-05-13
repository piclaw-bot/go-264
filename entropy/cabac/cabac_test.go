package cabac

import (
	"testing"

	"github.com/rcarmo/go-264/nal"
)

func TestCABACDecoder_Init(t *testing.T) {
	// Initialize with known byte stream
	data := []byte{0xAB, 0xCD, 0xEF, 0x01, 0x23}
	r := nal.NewReader(data)
	dec := NewCABACDecoder(r)

	if dec.codIRange != 510 {
		t.Errorf("range=%d want 510", dec.codIRange)
	}
	t.Logf("CABAC init: range=%d low=%d", dec.codIRange, dec.codILow)
}

func TestCABACDecoder_DecodeBin(t *testing.T) {
	data := []byte{0xFF, 0x00, 0xAA, 0x55, 0xFF, 0x00, 0xFF, 0x00}
	r := nal.NewReader(data)
	dec := NewCABACDecoder(r)
	ctx := CABACCtx{PState: 32, ValMPS: 0}

	// Decode several bins — should not panic
	for i := 0; i < 20; i++ {
		bin := dec.DecodeBin(&ctx)
		if i < 3 {
			t.Logf("bin[%d] = %d (pState=%d valMPS=%d)", i, bin, ctx.PState, ctx.ValMPS)
		}
	}
	// Context should have adapted
	if ctx.PState == 32 {
		t.Error("context didn't adapt after 20 bins")
	}
}

func TestCABACDecoder_DecodeBypass(t *testing.T) {
	data := []byte{0xAA, 0x55, 0xAA, 0x55}
	r := nal.NewReader(data)
	dec := NewCABACDecoder(r)

	var bits []uint32
	for i := 0; i < 16; i++ {
		bits = append(bits, dec.DecodeBypass())
	}
	t.Logf("Bypass bits: %v", bits)
}

func TestCABACDecoder_DecodeTerminate(t *testing.T) {
	data := []byte{0x00, 0x00, 0x80, 0x00}
	r := nal.NewReader(data)
	dec := NewCABACDecoder(r)

	// Should not terminate on normal data
	for i := 0; i < 5; i++ {
		term := dec.DecodeTerminate()
		if term == 1 {
			t.Logf("Terminated at step %d", i)
			return
		}
	}
	t.Log("No termination in 5 steps (expected for non-terminal data)")
}

func TestCABACDecoderPublicMethodsHandleInvalidInputs(t *testing.T) {
	var nilDec *CABACDecoder
	if nilDec.DecodeBin(nil) != 0 || nilDec.DecodeBypass() != 0 || nilDec.DecodeTerminate() != 0 || nilDec.DecodeUEG(0) != 0 {
		t.Fatal("nil decoder methods should return zero")
	}
	dec := &CABACDecoder{}
	if dec.DecodeBin(nil) != 0 || dec.DecodeBypass() != 0 || dec.DecodeTerminate() != 0 || dec.DecodeUEG(0) != 0 {
		t.Fatal("decoder without reader should return zero")
	}
	dec = &CABACDecoder{r: nal.NewReader([]byte{0xff, 0xff})}
	if dec.DecodeBin(&CABACCtx{PState: 32}) != 0 || dec.DecodeBypass() != 0 || dec.DecodeTerminate() != 0 || dec.DecodeUEG(-1) != 0 {
		t.Fatal("decoder with zero range should return zero")
	}
	dec = NewCABACDecoder(nal.NewReader([]byte{0xff, 0xff}))
	if dec.DecodeBin(nil) != 0 {
		t.Fatal("nil context should decode as zero")
	}
	if dec.DecodeBin(&CABACCtx{PState: 64}) != 0 || dec.DecodeBin(&CABACCtx{PState: 32, ValMPS: 2}) != 0 {
		t.Fatal("invalid context state should decode as zero")
	}
}

func TestValidResidualCoeffCount(t *testing.T) {
	cases := []struct {
		cat, maxCoeff int
		want          bool
	}{
		{0, 16, true}, {2, 16, true}, {3, 4, true}, {4, 15, true}, {5, 64, true},
		{3, 16, false}, {4, 16, false}, {5, 16, false}, {2, 64, false},
	}
	for _, tc := range cases {
		if got := validResidualCoeffCount(tc.cat, tc.maxCoeff); got != tc.want {
			t.Fatalf("validResidualCoeffCount(%d,%d) got %v want %v", tc.cat, tc.maxCoeff, got, tc.want)
		}
	}
}

func TestDecodeCABACResidualRejectsInvalidBounds(t *testing.T) {
	dec := NewCABACDecoder(nal.NewReader([]byte{0xff, 0xff, 0xff, 0xff}))
	models := InitContextModels(26, 0, false)
	var out [64]int16
	if got := dec.DecodeCABACResidual(models, 2, 0, out[:], 0, 0); got != 0 {
		t.Fatalf("maxCoeff=0 got %d want 0", got)
	}
	if got := dec.DecodeCABACResidual(models, 2, 65, make([]int16, 65), 0, 0); got != 0 {
		t.Fatalf("maxCoeff=65 got %d want 0", got)
	}
	if got := dec.DecodeCABACResidual(models[:10], 2, 16, out[:], 99, -4); got != 0 {
		t.Fatalf("short models got %d want 0", got)
	}
	if got := dec.DecodeCABACResidual(models, 3, 16, out[:], 0, 0); got != 0 {
		t.Fatalf("invalid cat/maxCoeff got %d want 0", got)
	}
}

func TestCABACContextInit(t *testing.T) {
	models := InitContextModels(26, 0, true)
	if len(models) != 1024 {
		t.Fatalf("expected 1024 models, got %d", len(models))
	}
	// All should be initialized
	for i, m := range models {
		if m.PState > 63 {
			t.Fatalf("model[%d].pState=%d out of range", i, m.PState)
		}
	}
}

func TestCABACTransitionTables(t *testing.T) {
	// Verify table sizes
	if len(transIdxLPS) != 64 {
		t.Fatal("transIdxLPS size")
	}
	if len(transIdxMPS) != 64 {
		t.Fatal("transIdxMPS size")
	}
	if len(rangeTabLPS) != 64 {
		t.Fatal("rangeTabLPS size")
	}

	// LPS transition should decrease pState (toward equi-probable)
	if transIdxLPS[63] != 63 {
		t.Error("LPS[63] should stay at 63")
	}
	if transIdxLPS[0] != 0 {
		t.Error("LPS[0] should stay at 0")
	}

	// MPS transition should increase pState
	if transIdxMPS[0] != 1 {
		t.Error("MPS[0] should go to 1")
	}
	if transIdxMPS[62] != 62 {
		t.Error("MPS[62] should stay at 62")
	}
}

func FuzzCABACDecode(f *testing.F) {
	f.Add([]byte{0xFF, 0x00, 0xAA, 0x55, 0xFF, 0x00})
	f.Add([]byte{0x00, 0x00, 0x00, 0x00})
	f.Add([]byte{0xFF, 0xFF, 0xFF, 0xFF})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 2 {
			return
		}
		r := nal.NewReader(data)
		dec := NewCABACDecoder(r)
		ctx := CABACCtx{PState: 32, ValMPS: 0}
		for i := 0; i < 50; i++ {
			dec.DecodeBin(&ctx)
		}
	})
}
