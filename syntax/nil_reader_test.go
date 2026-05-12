package syntax

import "testing"

func TestMacroblockDecodersHandleNilReader(t *testing.T) {
	if mb := DecodeMBIntra(nil, IntraDecodeOpts{}); mb == nil || mb.MBType != 0 {
		t.Fatalf("DecodeMBIntra(nil) = %#v", mb)
	}
	if mb := DecodeMBIntraWithType(nil, MBTypeI16x16_0, IntraDecodeOpts{}); mb == nil || mb.MBType != MBTypeI16x16_0 {
		t.Fatalf("DecodeMBIntraWithType(nil) = %#v", mb)
	}
	if mb := DecodeMBInter(nil, InterDecodeOpts{}); mb.MBType != 0 {
		t.Fatalf("DecodeMBInter(nil).MBType = %d want 0", mb.MBType)
	}
	if mb := DecodeMBBidi(nil, 26, 1, 1); mb == nil || mb.MBType != 0 {
		t.Fatalf("DecodeMBBidi(nil) = %#v", mb)
	}
	if got := readTE(nil, 1); got != 0 {
		t.Fatalf("readTE(nil,1) got %d want 0", got)
	}
	if got := decodeMVD(nil); got != (MotionVector{}) {
		t.Fatalf("decodeMVD(nil) got %+v want zero", got)
	}
}
