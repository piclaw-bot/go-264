package decode

import (
	"testing"

	"github.com/rcarmo/go-264/frame"
)

func TestRefListsSkipNonReferenceFrames(t *testing.T) {
	d := NewDecoder()
	d.DPB = frame.NewDPB(16)
	d.DPB.Add(&frame.Frame{POC: 0, IsRef: true})
	d.DPB.Add(&frame.Frame{POC: 2, IsRef: false})
	d.DPB.Add(&frame.Frame{POC: 4, IsRef: true})
	d.DPB.Add(&frame.Frame{POC: 6, IsRef: false})

	if got := d.refL0(0); got == nil || got.POC != 4 {
		t.Fatalf("refL0(0) = POC %v, want latest reference POC 4", pocOf(got))
	}
	if got := d.refL0(1); got == nil || got.POC != 0 {
		t.Fatalf("refL0(1) = POC %v, want previous reference POC 0", pocOf(got))
	}
	if got := d.refL1(0); got == nil || got.POC != 0 {
		t.Fatalf("refL1(0) = POC %v, want second-latest reference POC 0", pocOf(got))
	}
}

func TestRefListsKeepLegacySyntheticFrames(t *testing.T) {
	d := NewDecoder()
	d.DPB = frame.NewDPB(16)
	d.DPB.Add(&frame.Frame{POC: 0})
	d.DPB.Add(&frame.Frame{POC: 2})
	d.DPB.Add(&frame.Frame{POC: 4})

	if got := d.refL0(0); got == nil || got.POC != 4 {
		t.Fatalf("legacy refL0(0) = POC %v, want most recent synthetic POC 4", pocOf(got))
	}
	if got := d.refL1(0); got == nil || got.POC != 2 {
		t.Fatalf("legacy refL1(0) = POC %v, want second-most recent synthetic POC 2", pocOf(got))
	}
}

func pocOf(f *frame.Frame) any {
	if f == nil {
		return nil
	}
	return f.POC
}
