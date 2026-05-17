#!/usr/bin/env python3
"""Compare FFmpeg FFDIRECT rows with Go GODIRECT rows for a chosen frame/POC pair.

Usage:
  compare_direct_trace.py ffdirect.rows go-direct.log --ff-frame 2 --go-poc 6

The Go side is produced by running decode264 with GO264_DIRECT_TRACE=1.
"""
from __future__ import annotations
import argparse, re

FF_RE = re.compile(r'FFDIRECT mb=(\d+).*?frame=(\d+).*?spatial=(\d+).*?ref0=(-?\d+) ref1=(-?\d+) mv0=\{(-?\d+),(-?\d+)\} mv1=\{(-?\d+),(-?\d+)\}.*?sub0=(\d+) sub1=(\d+) sub2=(\d+) sub3=(\d+)')
GO_RE = re.compile(r'GODIRECT mb=(\d+).*?poc=(\d+).*?mb_type=(\d+) ref0=(-?\d+) ref1=(-?\d+) mv0=\{(-?\d+),(-?\d+)\} mv1=\{(-?\d+),(-?\d+)\}.*?sub0=(\d+) sub1=(\d+) sub2=(\d+) sub3=(\d+)')

def load(path, regex, frame_group):
    out = {}
    for line in open(path, errors='replace'):
        m = regex.search(line)
        if not m:
            continue
        vals = tuple(int(x) for x in m.groups())
        mb = vals[0]
        frame = vals[frame_group]
        out.setdefault((frame, mb), vals)
    return out

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('ffdirect')
    ap.add_argument('godirect')
    ap.add_argument('--ff-frame', type=int, required=True)
    ap.add_argument('--go-poc', type=int, required=True)
    ap.add_argument('--limit', type=int, default=50)
    args = ap.parse_args()
    ff = load(args.ffdirect, FF_RE, 1)
    go = load(args.godirect, GO_RE, 1)
    rows = 0
    diffs = 0
    for _, mb in sorted(k for k in ff if k[0] == args.ff_frame):
        f = ff[(args.ff_frame, mb)]
        g = go.get((args.go_poc, mb))
        rows += 1
        if g is None:
            print(f'mb={mb:04d} missing_go')
            diffs += 1
            continue
        f_ref_mv = (f[4], f[5], f[6], f[7], f[8])
        g_ref_mv = (g[3], g[4], g[5], g[6], g[7])
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
