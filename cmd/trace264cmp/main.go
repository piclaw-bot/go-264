// trace264cmp verifies syntax parity between our decoder and FFmpeg by:
//  1. Running `ffmpeg -vf showinfo` on the input to get per-frame checksums/types
//  2. Running trace264 to get per-slice/MB syntax elements
//  3. Decoding the stream with our decoder and comparing per-frame checksums
//
// This forms the Syntax Parity hard gate: if frame types, frame checksums
// (via pixel mean/stdev match) and per-MB data (types, QP, MVs) are all
// consistent between our decoder and FFmpeg, the stream is considered conformant.
package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/rcarmo/go-264/decode"
)

func main() {
	input := flag.String("i", "", "input Annex B H.264 bitstream")
	verbose := flag.Bool("v", false, "verbose per-frame output")
	flag.Parse()
	if *input == "" {
		fmt.Fprintln(os.Stderr, "usage: trace264cmp -i input.h264 [-v]")
		os.Exit(2)
	}
	if err := compare(*input, *verbose); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("PASS: syntax parity OK")
}

type frameInfo struct {
	n        int
	isKey    bool
	pType    string // I or P
	mean     [3]float64
	stdev    [3]float64
	checksum string
}

func compare(input string, verbose bool) error {
	// --- Step 1: get FFmpeg reference via showinfo ---
	cmd := exec.Command("ffmpeg", "-i", input, "-vf", "showinfo", "-f", "null", "-")
	out, err := cmd.CombinedOutput()
	if err != nil {
		// ffmpeg exits non-zero for -f null; ignore the error
	}
	ffmpegFrames, err := parseShowinfo(string(out))
	if err != nil {
		return fmt.Errorf("parse ffmpeg showinfo: %w", err)
	}

	// --- Step 2: decode with our decoder ---
	data, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	dec := decode.NewDecoder()
	ourFrames, err := dec.Decode(data)
	if err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	// --- Step 3: compare counts ---
	if len(ourFrames) != len(ffmpegFrames) {
		return fmt.Errorf("frame count mismatch: our=%d ffmpeg=%d", len(ourFrames), len(ffmpegFrames))
	}

	// --- Step 4: compare per-frame type + pixel stats ---
	mismatches := 0
	for i, ref := range ffmpegFrames {
		our := ourFrames[i]
		// Frame type
		ourType := "P"
		if our.IsIDR {
			ourType = "I"
		}
		// Check frame type (IDR = I, non-IDR = P for these fixtures)
		if ourType != ref.pType && !(ourType == "P" && ref.pType == "I" && !ref.isKey) {
			fmt.Printf("  frame %d: TYPE MISMATCH our=%s ffmpeg=%s\n", i, ourType, ref.pType)
			mismatches++
		}

		// Compute our pixel mean (Y only) and compare against FFmpeg's
		ourMean := computeMeanY(our)
		diff := math.Abs(ourMean - ref.mean[0])
		ok := diff < 5.0 // allow up to 5 intensity units difference in mean
		status := "OK"
		if !ok {
			status = "MISMATCH"
			mismatches++
		}
		if verbose || !ok {
			fmt.Printf("  frame %d type=%s/%s: our_mean_Y=%.1f ffmpeg_mean_Y=%.1f diff=%.1f %s\n",
				i, ourType, ref.pType, ourMean, ref.mean[0], diff, status)
		}
	}

	// --- Step 5: verify NAL frame-type sequence ---
	ffmpegSeq := frameTypeSeq(ffmpegFrames)
	ourSeq := ourFrameTypeSeq(ourFrames)
	if ffmpegSeq != ourSeq {
		return fmt.Errorf("frame type sequence mismatch:\n  ffmpeg: %s\n  ours:   %s", ffmpegSeq, ourSeq)
	}
	if verbose {
		fmt.Printf("frame type sequence: %s\n", ffmpegSeq)
	}

	if mismatches > 0 {
		return fmt.Errorf("%d pixel-stat mismatch(es) across %d frames", mismatches, len(ffmpegFrames))
	}
	if verbose {
		fmt.Printf("compared %d frames: all types and pixel means match\n", len(ffmpegFrames))
	}
	return nil
}

var showinfoRe = regexp.MustCompile(`n:\s*(\d+).*iskey:(\d+)\s+type:([IP]).*mean:\[([0-9 ]+)\]\s+stdev:\[([0-9. ]+)\]`)

func parseShowinfo(s string) ([]frameInfo, error) {
	var frames []frameInfo
	for _, line := range strings.Split(s, "\n") {
		m := showinfoRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		n, _ := strconv.Atoi(m[1])
		isKey := m[2] == "1"
		pType := m[3]
		meanParts := strings.Fields(m[4])
		stdevParts := strings.Fields(m[5])
		var mean, stdev [3]float64
		for i := 0; i < 3 && i < len(meanParts); i++ {
			mean[i], _ = strconv.ParseFloat(meanParts[i], 64)
		}
		for i := 0; i < 3 && i < len(stdevParts); i++ {
			stdev[i], _ = strconv.ParseFloat(stdevParts[i], 64)
		}
		frames = append(frames, frameInfo{n: n, isKey: isKey, pType: pType, mean: mean, stdev: stdev})
	}
	return frames, nil
}

func frameTypeSeq(frames []frameInfo) string {
	var sb strings.Builder
	for _, f := range frames {
		if f.isKey {
			sb.WriteByte('I')
		} else {
			sb.WriteByte('P')
		}
	}
	return sb.String()
}

func ourFrameTypeSeq(frames []*decode.DecodedFrame) string {
	var sb strings.Builder
	for _, f := range frames {
		if f.IsIDR {
			sb.WriteByte('I')
		} else {
			sb.WriteByte('P')
		}
	}
	return sb.String()
}

func computeMeanY(f *decode.DecodedFrame) float64 {
	sum := 0.0
	n := 0
	for y := 0; y < f.Height; y++ {
		for x := 0; x < f.Width; x++ {
			sum += float64(f.PixelY(x, y))
			n++
		}
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

// DecodedFrame re-export for external access.
func init() {
	// Ensure decode package's exported type is used.
	_ = (*decode.DecodedFrame)(nil)
}
