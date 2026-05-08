package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
)

type goMB struct {
	Type  string
	MVX   int
	MVY   int
	MVDX  int
	MVDY  int
	PredX int
	PredY int
}

type ffMV struct {
	MVX int
	MVY int
	W   int
	H   int
}

type ffMB struct {
	Area      int
	AreaSum   int
	MVX       int
	MVY       int
	W         int
	H         int
	Ambiguous bool
	All       []ffMV
}

var (
	reNAL   = regexp.MustCompile(`nal=(\d+).*frame=(\d+)`)
	reGOMB  = regexp.MustCompile(`mb=\d+ x=(\d+) y=(\d+) .*type=([^ ]+)(.*)`)
	reMV0   = regexp.MustCompile(`mv0=\((-?\d+),(-?\d+)\)`)
	reMVD0  = regexp.MustCompile(`mvd0=\((-?\d+),(-?\d+)\)`)
	rePred0 = regexp.MustCompile(`pred0=\((-?\d+),(-?\d+)\)`)
	reFFMV  = regexp.MustCompile(`mv dst=\((-?\d+),(-?\d+)\) size=(\d+)x(\d+) .*motion=\((-?\d+),(-?\d+)\)/(\d+)`)
)

func main() {
	goTrace := flag.String("go-trace", "", "trace264 output file")
	ffTrace := flag.String("ff-trace", "", "ffmvtrace output file")
	frame := flag.Int("frame", 1, "frame index to compare")
	nalIdx := flag.Int("nal", -1, "nal index in go trace to compare (overrides -frame for go trace)")
	limit := flag.Int("limit", 20, "max mismatches to print")
	maxMismatch := flag.Int("max-mismatch", -1, "fail with exit code 1 when mismatches exceed this threshold")
	onlyUnambiguous := flag.Bool("only-unambiguous", true, "compare only MBs with unambiguous FF representative MV")
	require16x16 := flag.Bool("require-16x16", false, "compare only MBs whose FF representative vector comes from a 16x16 block")
	matchAny := flag.Bool("match-any", false, "treat an MB as matched if any FF MV entry in that MB matches go mv")
	p16x16Only := flag.Bool("p16x16-only", false, "compare only go MBs with type P:0")
	requireFullCover := flag.Bool("require-full-cover", false, "compare only FF MB groups whose MV block areas sum to exactly one 16x16 MB")
	flag.Parse()

	if *goTrace == "" || *ffTrace == "" {
		fmt.Fprintln(os.Stderr, "usage: trace264diff -go-trace go.txt -ff-trace ff.txt [-frame N] [-limit K]")
		os.Exit(2)
	}

	goMBs, err := parseGo(*goTrace, *frame, *nalIdx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "parse go trace:", err)
		os.Exit(1)
	}
	ffMBs, err := parseFF(*ffTrace, *frame)
	if err != nil {
		fmt.Fprintln(os.Stderr, "parse ff trace:", err)
		os.Exit(1)
	}

	keys := make([][2]int, 0, len(goMBs))
	for k := range goMBs {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i][1] != keys[j][1] {
			return keys[i][1] < keys[j][1]
		}
		return keys[i][0] < keys[j][0]
	})

	mismatches := 0
	printed := 0
	compared := 0
	skippedAmbiguous := 0
	skippedSize := 0
	skippedCoverage := 0
	for _, k := range keys {
		g := goMBs[k]
		if g.Type == "P_SKIP" {
			continue
		}
		if *p16x16Only && g.Type != "P:0" {
			continue
		}
		f, ok := ffMBs[k]
		if !ok {
			continue
		}
		if *onlyUnambiguous && f.Ambiguous {
			skippedAmbiguous++
			continue
		}
		if *require16x16 && !(f.W == 16 && f.H == 16) {
			skippedSize++
			continue
		}
		if *requireFullCover && f.AreaSum != 256 {
			skippedCoverage++
			continue
		}
		compared++
		matched := g.MVX == f.MVX && g.MVY == f.MVY
		if *matchAny && !matched {
			for _, cand := range f.All {
				if g.MVX == cand.MVX && g.MVY == cand.MVY {
					matched = true
					break
				}
			}
		}
		if !matched {
			if printed < *limit {
				fmt.Printf("mb=(%d,%d) go type=%s mvd=(%d,%d) pred=(%d,%d) mv=(%d,%d) ff mv=(%d,%d) ff_size=%dx%d ambiguous=%v ff_candidates=%d\n", k[0], k[1], g.Type, g.MVDX, g.MVDY, g.PredX, g.PredY, g.MVX, g.MVY, f.MVX, f.MVY, f.W, f.H, f.Ambiguous, len(f.All))
				printed++
			}
			mismatches++
		}
	}
	fmt.Printf("total_mismatches=%d compared=%d skipped_ambiguous=%d skipped_size=%d skipped_coverage=%d\n", mismatches, compared, skippedAmbiguous, skippedSize, skippedCoverage)
	if *maxMismatch >= 0 && mismatches > *maxMismatch {
		os.Exit(1)
	}
}

func parseGo(path string, targetFrame int, targetNAL int) (map[[2]int]goMB, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out := map[[2]int]goMB{}
	s := bufio.NewScanner(f)
	frame := -1
	nal := -1
	for s.Scan() {
		line := s.Text()
		if len(line) > 4 && line[:4] == "nal=" {
			m := reNAL.FindStringSubmatch(line)
			if len(m) == 3 {
				fmt.Sscanf(m[1], "%d", &nal)
				fmt.Sscanf(m[2], "%d", &frame)
			}
			continue
		}
		if targetNAL >= 0 {
			if nal != targetNAL {
				continue
			}
		} else if frame != targetFrame {
			continue
		}
		m := reGOMB.FindStringSubmatch(line)
		if len(m) != 5 {
			continue
		}
		var x, y int
		fmt.Sscanf(m[1], "%d", &x)
		fmt.Sscanf(m[2], "%d", &y)
		typ := m[3]
		rest := m[4]
		mvx, mvy := 0, 0
		mvdx, mvdy := 0, 0
		predx, predy := 0, 0
		mm := reMV0.FindStringSubmatch(rest)
		if len(mm) == 3 {
			fmt.Sscanf(mm[1], "%d", &mvx)
			fmt.Sscanf(mm[2], "%d", &mvy)
		}
		if dm := reMVD0.FindStringSubmatch(rest); len(dm) == 3 {
			fmt.Sscanf(dm[1], "%d", &mvdx)
			fmt.Sscanf(dm[2], "%d", &mvdy)
		}
		if pm := rePred0.FindStringSubmatch(rest); len(pm) == 3 {
			fmt.Sscanf(pm[1], "%d", &predx)
			fmt.Sscanf(pm[2], "%d", &predy)
		}
		out[[2]int{x, y}] = goMB{Type: typ, MVX: mvx, MVY: mvy, MVDX: mvdx, MVDY: mvdy, PredX: predx, PredY: predy}
	}
	return out, s.Err()
}

func parseFF(path string, targetFrame int) (map[[2]int]ffMB, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out := map[[2]int]ffMB{}
	s := bufio.NewScanner(f)
	frame := -1
	for s.Scan() {
		line := s.Text()
		if len(line) > 6 && line[:6] == "frame=" {
			fmt.Sscanf(line, "frame=%d", &frame)
			continue
		}
		if frame != targetFrame {
			continue
		}
		m := reFFMV.FindStringSubmatch(line)
		if len(m) != 8 {
			continue
		}
		var dx, dy, w, h, mx, my, ms int
		fmt.Sscanf(m[1], "%d", &dx)
		fmt.Sscanf(m[2], "%d", &dy)
		fmt.Sscanf(m[3], "%d", &w)
		fmt.Sscanf(m[4], "%d", &h)
		fmt.Sscanf(m[5], "%d", &mx)
		fmt.Sscanf(m[6], "%d", &my)
		fmt.Sscanf(m[7], "%d", &ms)
		if ms == 0 {
			continue
		}
		mbX := (dx - 8) / 16
		mbY := (dy - 8) / 16
		if mbX < 0 || mbY < 0 {
			continue
		}
		k := [2]int{mbX, mbY}
		area := w * h
		cand := ffMB{Area: area, AreaSum: area, MVX: mx, MVY: my, W: w, H: h, All: []ffMV{{MVX: mx, MVY: my, W: w, H: h}}}
		if prev, ok := out[k]; !ok {
			out[k] = cand
			continue
		} else {
			prev.All = append(prev.All, ffMV{MVX: mx, MVY: my, W: w, H: h})
			prev.AreaSum += area
			if cand.Area > prev.Area {
				cand.All = prev.All
				cand.AreaSum = prev.AreaSum
				cand.Ambiguous = prev.Ambiguous
				out[k] = cand
				continue
			}
			if cand.Area == prev.Area && (cand.MVX != prev.MVX || cand.MVY != prev.MVY || cand.W != prev.W || cand.H != prev.H) {
				prev.Ambiguous = true
			}
			out[k] = prev
		}
	}
	return out, s.Err()
}
