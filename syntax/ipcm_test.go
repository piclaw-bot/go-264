package syntax

import (
	"testing"

	"github.com/rcarmo/go-264/nal"
)

func TestDecodeMBIntraIPCMConsumesRawSamples(t *testing.T) {
	payload := make([]byte, 256+64+64)
	for i := range payload {
		payload[i] = byte(i)
	}
	mb := DecodeMBIntraWithType(nal.NewReader(payload), MBTypeIPCM, IntraDecodeOpts{})
	if mb.MBType != MBTypeIPCM {
		t.Fatalf("mb type got %d want IPCM", mb.MBType)
	}
	if mb.PCMY[0] != 0 || mb.PCMY[255] != 255 || mb.PCMCb[0] != 0 || mb.PCMCb[63] != 63 || mb.PCMCr[0] != 64 || mb.PCMCr[63] != 127 {
		t.Fatalf("unexpected PCM samples: Y0=%d Y255=%d Cb0=%d Cb63=%d Cr0=%d Cr63=%d", mb.PCMY[0], mb.PCMY[255], mb.PCMCb[0], mb.PCMCb[63], mb.PCMCr[0], mb.PCMCr[63])
	}
}
