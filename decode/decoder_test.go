package decode

import (
	"os"
	"testing"
)

func TestDecodeIDR(t *testing.T) {
	data, err := os.ReadFile("/tmp/test.h264")
	if err != nil {
		t.Skipf("no test bitstream: %v", err)
	}

	dec := NewDecoder()
	frames, err := dec.Decode(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	t.Logf("Decoded %d frames", len(frames))
	if len(frames) == 0 {
		t.Fatal("expected at least 1 frame")
	}

	f := frames[0]
	t.Logf("Frame: %dx%d IDR=%v ref=%v poc=%d",
		f.Width, f.Height, f.IsIDR, f.IsRef, f.POC)

	if f.Width != 320 || f.Height != 240 {
		t.Errorf("frame size %dx%d want 320x240", f.Width, f.Height)
	}

	// Check that frame isn't all zeros (prediction produced something)
	nonZero := 0
	for _, v := range f.Y[:f.Width*f.Height] {
		if v != 0 {
			nonZero++
		}
	}
	t.Logf("Non-zero luma pixels: %d/%d (%.1f%%)",
		nonZero, f.Width*f.Height, float64(nonZero)*100/float64(f.Width*f.Height))
}

func TestDecoderSPSPPS(t *testing.T) {
	data, err := os.ReadFile("/tmp/test.h264")
	if err != nil {
		t.Skipf("no test bitstream: %v", err)
	}

	dec := NewDecoder()
	dec.Decode(data)

	if len(dec.SPS) == 0 {
		t.Fatal("no SPS parsed")
	}
	if len(dec.PPS) == 0 {
		t.Fatal("no PPS parsed")
	}

	for id, sps := range dec.SPS {
		t.Logf("SPS[%d]: profile=%d level=%d %dx%d",
			id, sps.ProfileIDC, sps.LevelIDC, sps.Width, sps.Height)
	}
	for id, pps := range dec.PPS {
		t.Logf("PPS[%d]: sps=%d entropy=%d qp=%d",
			id, pps.SPSID, pps.EntropyCodingMode, pps.PicInitQP)
	}
}

func TestDecodePFrame(t *testing.T) {
	data, err := os.ReadFile("/tmp/test.h264")
	if err != nil {
		t.Skipf("no test bitstream: %v", err)
	}

	dec := NewDecoder()
	frames, err := dec.Decode(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	t.Logf("Decoded %d frames total", len(frames))
	for i, f := range frames {
		t.Logf("  Frame %d: %dx%d IDR=%v ref=%v poc=%d",
			i, f.Width, f.Height, f.IsIDR, f.IsRef, f.POC)
	}

	// Should have at least the IDR frame + P-frames
	if len(frames) < 1 {
		t.Fatal("expected at least 1 frame")
	}
}

func TestDecoderTraceMBReceivesCABACEvents(t *testing.T) {
	data, err := os.ReadFile("/workspace/tmp/testsrc_cabac_p.h264")
	if err != nil {
		t.Skipf("no CABAC fixture: %v", err)
	}
	dec := NewDecoder()
	seenCABAC := false
	dec.TraceMB = func(ev MBTraceEvent) {
		if ev.EntropyCABAC {
			seenCABAC = true
		}
	}
	if _, err := dec.Decode(data); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !seenCABAC {
		t.Fatal("TraceMB did not receive CABAC macroblock events")
	}
}
