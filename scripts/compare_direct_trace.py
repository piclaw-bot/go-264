#!/usr/bin/env python3
"""Compare FFmpeg FFDIRECT rows with Go GODIRECT rows for a chosen frame/POC pair.

Usage:
  compare_direct_trace.py ffdirect.rows go-direct.log --ff-frame 2 --go-poc 6

FFmpeg's trace uses H.264 frame_num, which repeats across display/decode-order
B pictures. The comparator therefore splits repeated frame_num groups whenever the
macroblock address wraps and selects one occurrence with --ff-occurrence.
The Go side is produced by running decode264 with GO264_DIRECT_TRACE=1.
"""
from __future__ import annotations
import argparse, re
from collections import defaultdict

FF_RE = re.compile(r'FFDIRECT mb=(\d+).*?frame=(\d+).*?spatial=(\d+).*?ref0=(-?\d+) ref1=(-?\d+) mv0=\{(-?\d+),(-?\d+)\} mv1=\{(-?\d+),(-?\d+)\}.*?sub0=(\d+) sub1=(\d+) sub2=(\d+) sub3=(\d+)')
GO_RE = re.compile(r'GODIRECT mb=(\d+).*?poc=(\d+).*?mb_type=(\d+) ref0=(-?\d+) ref1=(-?\d+) mv0=\{(-?\d+),(-?\d+)\} mv1=\{(-?\d+),(-?\d+)\}.*?sub0=(\d+) sub1=(\d+) sub2=(\d+) sub3=(\d+)')

def load_ff(path):
    out = {}
    occurrence = defaultdict(int)
    last_mb = {}
    for line in open(path, errors='replace'):
        m = FF_RE.search(line)
        if not m:
            continue
        vals = tuple(int(x) for x in m.groups())
        mb, frame = vals[0], vals[1]
        if frame in last_mb and mb <= last_mb[frame]:
            occurrence[frame] += 1
        last_mb[frame] = mb
        out[(frame, occurrence[frame], mb)] = vals
    return out

def load_go(path):
    out = {}
    for line in open(path, errors='replace'):
        m = GO_RE.search(line)
        if not m:
            continue
        vals = tuple(int(x) for x in m.groups())
        mb, poc = vals[0], vals[1]
        out[(poc, mb)] = vals
    return out

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('ffdirect')
    ap.add_argument('godirect')
    ap.add_argument('--ff-frame', type=int, required=True)
    ap.add_argument('--ff-occurrence', type=int, default=0, help='which repeated frame_num group to compare')
    ap.add_argument('--go-poc', type=int, required=True)
    ap.add_argument('--limit', type=int, default=50)
    args = ap.parse_args()
    ff = load_ff(args.ffdirect)
    go = load_go(args.godirect)
    rows = 0
    diffs = 0
    frame_keys = sorted(k for k in ff if k[0] == args.ff_frame and k[1] == args.ff_occurrence)
    if not frame_keys:
        occurrences = sorted({k[1] for k in ff if k[0] == args.ff_frame})
        print(f'no_ff_rows frame={args.ff_frame} occurrence={args.ff_occurrence} available_occurrences={occurrences}')
        return
    for _, _, mb in frame_keys:
        f = ff[(args.ff_frame, args.ff_occurrence, mb)]
        g = go.get((args.go_poc, mb))
        rows += 1
        if g is None:
            print(f'mb={mb:04d} missing_go')
            diffs += 1
            continue
        f_ref_mv = (f[3], f[4], f[5], f[6], f[7], f[8])
        g_ref_mv = (g[3], g[4], g[5], g[6], g[7], g[8])
        f_sub = f[9:13]
        g_sub = g[9:13]
        if f_ref_mv != g_ref_mv or f_sub != g_sub:
            print(f'mb={mb:04d} ff_ref_mv={f_ref_mv} go_ref_mv={g_ref_mv} ff_sub={f_sub} go_sub={g_sub}')
            diffs += 1
            if diffs >= args.limit:
                break
    print(f'compared={rows} diffs={diffs}')

if __name__ == '__main__':
    main()
