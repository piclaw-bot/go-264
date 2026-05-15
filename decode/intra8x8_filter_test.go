package decode

import "testing"

func TestFilteredI8x8TopMatchesFFmpegPred8x8L(t *testing.T) {
	top := [16]uint8{10, 20, 30, 40, 50, 60, 70, 80, 90}
	got := filteredI8x8Top(top, 5, true, true)
	want := [8]int{
		(5 + 2*10 + 20 + 2) >> 2,
		(10 + 2*20 + 30 + 2) >> 2,
		(20 + 2*30 + 40 + 2) >> 2,
		(30 + 2*40 + 50 + 2) >> 2,
		(40 + 2*50 + 60 + 2) >> 2,
		(50 + 2*60 + 70 + 2) >> 2,
		(60 + 2*70 + 80 + 2) >> 2,
		(90 + 2*80 + 70 + 2) >> 2,
	}
	if got != want {
		t.Fatalf("filtered top got %v want %v", got, want)
	}

	got = filteredI8x8Top(top, 5, false, false)
	want[0] = (10 + 2*10 + 20 + 2) >> 2
	want[7] = (80 + 2*80 + 70 + 2) >> 2
	if got != want {
		t.Fatalf("filtered top unavailable edges got %v want %v", got, want)
	}
}

func TestFilteredI8x8LeftMatchesFFmpegPred8x8L(t *testing.T) {
	left := [8]uint8{11, 21, 31, 41, 51, 61, 71, 81}
	got := filteredI8x8Left(left, 7, true)
	want := [8]int{
		(7 + 2*11 + 21 + 2) >> 2,
		(11 + 2*21 + 31 + 2) >> 2,
		(21 + 2*31 + 41 + 2) >> 2,
		(31 + 2*41 + 51 + 2) >> 2,
		(41 + 2*51 + 61 + 2) >> 2,
		(51 + 2*61 + 71 + 2) >> 2,
		(61 + 2*71 + 81 + 2) >> 2,
		(71 + 3*81 + 2) >> 2,
	}
	if got != want {
		t.Fatalf("filtered left got %v want %v", got, want)
	}

	got = filteredI8x8Left(left, 7, false)
	want[0] = (11 + 2*11 + 21 + 2) >> 2
	if got != want {
		t.Fatalf("filtered left unavailable top-left got %v want %v", got, want)
	}
}
