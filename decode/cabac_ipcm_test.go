package decode

import (
	"testing"

	cabac "github.com/rcarmo/go-264/entropy/cabac"
	"github.com/rcarmo/go-264/nal"
	"github.com/rcarmo/go-264/syntax"
)

func TestDecodeCABACIPCMSamplesConsumesRawBytesAndReinitializes(t *testing.T) {
	payload := make([]byte, 386)
	for i := range payload {
		payload[i] = byte(i)
	}
	dec := cabac.NewCABACDecoder(nal.NewReader(append([]byte{0, 0}, payload...)))
	mb := &syntax.MBIntra{}
	decodeCABACIPCMSamples(dec, mb)
	if mb.PCMY[0] != 0 || mb.PCMY[255] != 255 {
		t.Fatalf("luma PCM endpoints got %d/%d want 0/255", mb.PCMY[0], mb.PCMY[255])
	}
	if mb.PCMCb[0] != 0 || mb.PCMCb[63] != 63 {
		t.Fatalf("Cb PCM endpoints got %d/%d want 0/63", mb.PCMCb[0], mb.PCMCb[63])
	}
	if mb.PCMCr[0] != 64 || mb.PCMCr[63] != 127 {
		t.Fatalf("Cr PCM endpoints got %d/%d want 64/127", mb.PCMCr[0], mb.PCMCr[63])
	}
	if dec.DecodeTerminate() != 0 {
		t.Fatal("CABAC decoder was not reinitialized on bytes following I_PCM payload")
	}
}

func TestDecodeCABACIPCMSamplesHandlesInvalidInputs(t *testing.T) {
	decodeCABACIPCMSamples(nil, nil)
	decodeCABACIPCMSamples(cabac.NewCABACDecoder(nal.NewReader(nil)), nil)
}
