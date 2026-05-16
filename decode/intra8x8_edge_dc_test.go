package decode

import (
	"testing"

	"github.com/rcarmo/go-264/pred"
)

func TestI8x8DCEdgeReconModes(t *testing.T) {
	if got := i8x8DCEdgeReconMode(pred.Intra4x4DC, false, true); got != 9 {
		t.Fatalf("top-missing DC recon mode = %d, want LEFT_DC 9", got)
	}
	if got := i8x8DCEdgeReconMode(pred.Intra4x4DC, true, false); got != 10 {
		t.Fatalf("left-missing DC recon mode = %d, want TOP_DC 10", got)
	}
	if got := i8x8DCEdgeReconMode(pred.Intra4x4DC, true, true); got != pred.Intra4x4DC {
		t.Fatalf("available DC recon mode = %d, want DC", got)
	}
}
