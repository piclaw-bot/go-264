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
	Type string
	MVX  int
	MVY  int
}

type ffMB struct {
	Area int
	MVX  int
	MVY  int
}

var (
	reNAL  = regexp.MustCompile(`frame=(\d+)`)
	reGOMB = regexp.MustCompile(`mb=\d+ x=(\d+) y=(\d+) .*type=([^ ]+)(.*)`)
	reMV0  = regexp.MustCompile(`mv0=\((-?\d+),(-?\d+)\)`)
	reFFMV = regexp.MustCompile(`mv dst=\((-?\d+),(-?\d+)\) size=(\d+)x(\d+) .*motion=\((-?\d+),(-?\d+)\)/(\d+)`)
)

func main() {
	goTrace := flag.String("go-trace", "", "trace264 output file")
	ffTrace := flag.String("ff-trace", "", "ffmvtrace output file")
	frame := flag.Int("frame", 1, "frame index to compare")
	limit := flag.Int("limit", 20, "max mismatches to print")
	flag.Parse()

	if *goTrace == "" || *ffTrace == "" {
		fmt.Fprintln(os.Stderr, "usage: trace264diff -go-trace go.txt -ff-trace ff.txt [-frame N] [-limit K]")
		os.Exit(2)
	}

	goMBs, err := parseGo(*goTrace, *frame)
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
	for _, k := range keys {
		g := goMBs[k]
		if g.Type == "P_SKIP" {
			continue
		}
		f, ok := ffMBs[k]
		if !ok {
			continue
		}
		if g.MVX != f.MVX || g.MVY != f.MVY {
			fmt.Printf("mb=(%d,%d) go type=%s mv=(%d,%d) ff mv=(%d,%d)\n", k[0], k[1], g.Type, g.MVX, g.MVY, f.MVX, f.MVY)
			mismatches++
			if mismatches >= *limit {
				break
			}
		}
	}
	fmt.Printf("total_mismatches=%d\n", mismatches)
}

func parseGo(path string, targetFrame int) (map[[2]int]goMB, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out := map[[2]int]goMB{}
	s := bufio.NewScanner(f)
	frame := -1
	for s.Scan() {
		line := s.Text()
		if len(line) > 4 && line[:4] == "nal=" {
			m := reNAL.FindStringSubmatch(line)
			if len(m) == 2 {
				fmt.Sscanf(m[1], "%d", &frame)
			}
			continue
		}
		if frame != targetFrame {
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
		mm := reMV0.FindStringSubmatch(rest)
		if len(mm) == 3 {
			fmt.Sscanf(mm[1], "%d", &mvx)
			fmt.Sscanf(mm[2], "%d", &mvy)
		}
		out[[2]int{x, y}] = goMB{Type: typ, MVX: mvx, MVY: mvy}
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
		cand := ffMB{Area: area, MVX: mx, MVY: my}
		if prev, ok := out[k]; !ok || cand.Area > prev.Area {
			out[k] = cand
		}
	}
	return out, s.Err()
}
