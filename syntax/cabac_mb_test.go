package syntax

import (
	"testing"

	cabac "github.com/rcarmo/go-264/entropy/cabac"
)

func TestDecodeCABACRefInvalidInputsReturnZero(t *testing.T) {
	if got := DecodeCABACRef(nil, nil, -10); got != 0 {
		t.Fatalf("nil decoder got %d want 0", got)
	}
	if got := DecodeCABACRef(nil, make([]cabac.CABACCtx, 59), 99); got != 0 {
		t.Fatalf("nil decoder with invalid ctx got %d want 0", got)
	}
}
