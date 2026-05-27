package decode

import (
	"fmt"
	"os"
	"strconv"

	"github.com/rcarmo/go-264/frame"
)

func traceSavedMotion(f *frame.Frame, mbWidth int) {
	if os.Getenv("GO264_MOTION_SAVE_TRACE") == "" || f == nil || f.MotionStride4 <= 0 || mbWidth <= 0 || len(f.MotionL0) == 0 || len(f.RefIdxL0) != len(f.MotionL0) {
		return
	}
	limit := len(f.MBType)
	if v := os.Getenv("GO264_MOTION_SAVE_MB_LIMIT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 && n < limit {
			limit = n
		}
	}
	detail := os.Getenv("GO264_MOTION_SAVE_DETAIL") != ""
	for mb := 0; mb < limit; mb++ {
		mbX, mbY := mb%mbWidth, mb/mbWidth
		if detail {
			for y := 0; y < 4; y++ {
				for x := 0; x < 4; x++ {
					idx := (mbY*4+y)*f.MotionStride4 + mbX*4 + x
					if idx < 0 || idx >= len(f.MotionL0) || idx >= len(f.RefIdxL0) {
						continue
					}
					mv := f.MotionL0[idx]
					fmt.Fprintf(os.Stderr, "GOMOTSAVE4 frame=%d poc=%d mb=%04d x=%02d y=%02d cell=%d,%d idx=%d mbtype=%d ref0=%d mv0={%d,%d}\n", f.FrameNum, f.POC, mb, mbX, mbY, x, y, idx, f.MBType[mb], f.RefIdxL0[idx], mv[0], mv[1])
				}
			}
		}
		for part := 0; part < 4; part++ {
			x4 := mbX*4 + (part&1)*3
			y4 := mbY*4 + (part>>1)*3
			idx := y4*f.MotionStride4 + x4
			if idx < 0 || idx >= len(f.MotionL0) || idx >= len(f.RefIdxL0) {
				continue
			}
			mv := f.MotionL0[idx]
			fmt.Fprintf(os.Stderr, "GOMOTSAVE frame=%d poc=%d mb=%04d x=%02d y=%02d part=%d idx=%d mbtype=%d ref0=%d mv0={%d,%d}\n", f.FrameNum, f.POC, mb, mbX, mbY, part, idx, f.MBType[mb], f.RefIdxL0[idx], mv[0], mv[1])
		}
	}
}
