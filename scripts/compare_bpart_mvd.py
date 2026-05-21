#!/usr/bin/env python3
"""Compare FFmpeg FF_BPART_MVD rows with Go GOBIDI MVD/MVP diagnostics."""
from __future__ import annotations
import argparse
import re
from collections import defaultdict

FF_RE = re.compile(
    r'FF_BPART_MVD mb=(?P<mb>\d+)(?: frame=(?P<frame>\d+))? part=(?P<part>\d+) list=(?P<list>\d+) '
    r'mvd_abs=\{(?P<absx>-?\d+),(?P<absy>-?\d+)\} mvd=\{(?P<mvdx>-?\d+),(?P<mvdy>-?\d+)\} '
    r'mvp=\{(?P<mvpx>-?\d+),(?P<mvpy>-?\d+)\} final=\{(?P<finalx>-?\d+),(?P<finaly>-?\d+)\}'
)
GO_RE = re.compile(
    r'GOBIDI mb=(?P<mb>\d+).*?poc=(?P<poc>-?\d+)\b.*?'
    r'mv0=\{(?P<final0x>-?\d+),(?P<final0y>-?\d+)\} mv1=\{(?P<final1x>-?\d+),(?P<final1y>-?\d+)\} '
    r'mv0p1=\{(?P<final0p1x>-?\d+),(?P<final0p1y>-?\d+)\} mv1p1=\{(?P<final1p1x>-?\d+),(?P<final1p1y>-?\d+)\}.*?'
    r'amvd0=\{(?P<amvd0x>-?\d+),(?P<amvd0y>-?\d+)\} mvd0=\{(?P<mvd0x>-?\d+),(?P<mvd0y>-?\d+)\} mvp0=\{(?P<mvp0x>-?\d+),(?P<mvp0y>-?\d+)\} '
    r'amvd0p1=\{(?P<amvd0p1x>-?\d+),(?P<amvd0p1y>-?\d+)\} mvd0p1=\{(?P<mvd0p1x>-?\d+),(?P<mvd0p1y>-?\d+)\} mvp0p1=\{(?P<mvp0p1x>-?\d+),(?P<mvp0p1y>-?\d+)\} '
    r'amvd1=\{(?P<amvd1x>-?\d+),(?P<amvd1y>-?\d+)\} mvd1=\{(?P<mvd1x>-?\d+),(?P<mvd1y>-?\d+)\} mvp1=\{(?P<mvp1x>-?\d+),(?P<mvp1y>-?\d+)\} '
    r'amvd1p1=\{(?P<amvd1p1x>-?\d+),(?P<amvd1p1y>-?\d+)\} mvd1p1=\{(?P<mvd1p1x>-?\d+),(?P<mvd1p1y>-?\d+)\} mvp1p1=\{(?P<mvp1p1x>-?\d+),(?P<mvp1p1y>-?\d+)\}'
)

def iv(m: re.Match[str], name: str) -> int:
    return int(m.group(name))

def load_ff(path: str, frame_filter: int | None, occurrence: int) -> dict[tuple[int, int, int], dict[str, tuple[int, int]]]:
    out = {}
    seen: defaultdict[tuple[int, int, int], int] = defaultdict(int)
    for line in open(path, errors='replace'):
        m = FF_RE.search(line)
        if not m:
            continue
        if frame_filter is not None:
            # Frame-qualified FF rows are required for reliable comparison because
            # H.264 frame_num repeats across B pictures. Older artifacts without
            # frame= are ambiguous, so skip them instead of silently mixing frames.
            if m.group('frame') is None or iv(m, 'frame') != frame_filter:
                continue
        key = (iv(m, 'mb'), iv(m, 'part'), iv(m, 'list'))
        occ = seen[key]; seen[key] += 1
        if occ != occurrence:
            continue
        out[key] = {
            'mvd': (iv(m, 'mvdx'), iv(m, 'mvdy')),
            'mvp': (iv(m, 'mvpx'), iv(m, 'mvpy')),
            'final': (iv(m, 'finalx'), iv(m, 'finaly')),
        }
    return out

def load_go(path: str, poc: int, occurrence: int) -> dict[tuple[int, int, int], dict[str, tuple[int, int]]]:
    out = {}
    seen: defaultdict[int, int] = defaultdict(int)
    for line in open(path, errors='replace'):
        m = GO_RE.search(line)
        if not m or iv(m, 'poc') != poc:
            continue
        mb = iv(m, 'mb')
        occ = seen[mb]; seen[mb] += 1
        if occ != occurrence:
            continue
        for part, suffix in ((0, ''), (1, 'p1')):
            for list_idx in (0, 1):
                pfx = f'{list_idx}{suffix}'
                out[(mb, part, list_idx)] = {
                    'amvd': (iv(m, f'amvd{pfx}x'), iv(m, f'amvd{pfx}y')),
                    'mvd': (iv(m, f'mvd{pfx}x'), iv(m, f'mvd{pfx}y')),
                    'mvp': (iv(m, f'mvp{pfx}x'), iv(m, f'mvp{pfx}y')),
                    'final': (iv(m, f'final{pfx}x'), iv(m, f'final{pfx}y')),
                }
    return out

def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument('ffbpart_mvd')
    ap.add_argument('gobidi')
    ap.add_argument('--ff-frame', type=int, help='only compare FF rows for this frame when rows include frame=')
    ap.add_argument('--go-poc', type=int, required=True)
    ap.add_argument('--ff-occurrence', type=int, default=0)
    ap.add_argument('--go-occurrence', type=int, default=0)
    ap.add_argument('--mb', type=int)
    ap.add_argument('--from-mb', type=int, dest='from_mb')
    ap.add_argument('--to-mb', type=int, dest='to_mb')
    ap.add_argument('--limit', type=int, default=50)
    ap.add_argument('--fail-on-diff', action='store_true')
    args = ap.parse_args()
    ff = load_ff(args.ffbpart_mvd, args.ff_frame, args.ff_occurrence)
    go = load_go(args.gobidi, args.go_poc, args.go_occurrence)
    if not ff:
        print(f'no_ff_mvd_rows frame={args.ff_frame} occurrence={args.ff_occurrence}')
        if args.fail_on_diff:
            raise SystemExit(1)
        return
    compared = diffs = 0
    for key in sorted(ff):
        mb, part, list_idx = key
        if args.mb is not None and mb != args.mb:
            continue
        if args.from_mb is not None and mb < args.from_mb:
            continue
        if args.to_mb is not None and mb > args.to_mb:
            continue
        g = go.get(key)
        if g is None:
            print(f'mb={mb:04d} part={part} list={list_idx} missing_go')
            diffs += 1
        else:
            compared += 1
            fields = [name for name in ('mvd', 'mvp', 'final') if ff[key][name] != g[name]]
            if fields:
                print(f'mb={mb:04d} part={part} list={list_idx} fields={",".join(fields)} ff_mvd={ff[key]["mvd"]} go_mvd={g["mvd"]} go_amvd={g["amvd"]} ff_mvp={ff[key]["mvp"]} go_mvp={g["mvp"]} ff_final={ff[key]["final"]} go_final={g["final"]}')
                diffs += 1
        if diffs >= args.limit:
            break
    print(f'compared={compared} diffs={diffs}')
    if args.fail_on_diff and diffs:
        raise SystemExit(1)

if __name__ == '__main__':
    main()
