package decode

import "testing"

func TestLuma8x8FFmpegStorageFromDirectMatchesTraceLayout(t *testing.T) {
	direct := [64]int16{
		1, -3, 0, 2, 0, 0, 0, 0,
		-5, 0, 1, -1, 0, 0, 0, 0,
		1, -1, 0, 0, -1, 0, 0, 0,
		0, 1, 0, -1, 1, 0, 0, 0,
		-1, 1, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		-1, 0, 0, 0, 0, 0, 0, 0,
	}
	want := [64]int16{
		1, -5, 1, 0, -1, 0, 0, -1,
		-3, 0, -1, 1, 1, 0, 0, 0,
		0, 1, 0, 0, 0, 0, 0, 0,
		2, -1, 0, -1, 0, 0, 0, 0,
		0, 0, -1, 1, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
	}
	if got := luma8x8FFmpegStorageFromDirect(direct); got != want {
		t.Fatalf("FFmpeg storage trace conversion got %v want %v", got, want)
	}
}
