#!/usr/bin/env python3
"""Compare FFmpeg colocated direct rows against Go saved frame motion rows.

FFCOLZERO/FFCOLZERO8 rows expose the colocated ref/MV FFmpeg read from the
future reference. GOMOTSAVE rows expose the 4x4 representative ref/MV Go saved
for a decoded frame. This comparator checks whether Go's saved metadata for a
chosen colocated POC matches the FFmpeg rows for the same absolute MB/8x8 part.
"""
from __future__ import annotations
import argparse
import re

FF_RE = re.compile(
    r'FFCOLZERO(?P<kind>8?) mb=(?P<mb>\d+)(?: poc=(?P<poc>-?\d+))?(?: i8=(?P<part>\d+))?(?: i4=\d+)?.*?'
    r'colref0=(?P<ref>-?\d+).*?colmv=\{(?P<x>-?\d+),(?P<y>-?\d+)\} ref0=(?P<ref0>-?\d+) ref1=(?P<ref1>-?\d+)'
)
GO_RE = re.compile(
    r'GOMOTSAVE frame=(?P<frame>-?\d+) poc=(?P<poc>-?\d+) mb=(?P<mb>\d+).*?part=(?P<part>\d+) '
    r'mbtype=(?P<mbtype>\d+) ref0=(?P<ref>-?\d+) mv0=\{(?P<x>-?\d+),(?P<y>-?\d+)\}'
)
GO4_RE = re.compile(
    r'GOMOTSAVE4 frame=(?P<frame>-?\d+) poc=(?P<poc>-?\d+) mb=(?P<mb>\d+).*?cell=(?P<xcell>\d+),(?P<ycell>\d+) '
    r'mbtype=(?P<mbtype>\d+) ref0=(?P<ref>-?\d+) mv0=\{(?P<x>-?\d+),(?P<y>-?\d+)\}'
)


def load_ff(path: str) -> list[dict[str, int]]:
    rows = []
    for line in open(path, errors='replace'):
        m = FF_RE.search(line)
        if not m:
            continue
        rows.append({
            'mb': int(m['mb']),
            'part': int(m['part'] or 0),
            'ref_mv': (int(m['ref']), int(m['x']), int(m['y'])),
            'ref0': int(m['ref0']),
            'ref1': int(m['ref1']),
        })
    return rows


def load_go(path: str, frame_filter: int | None = None, detail: bool = False) -> dict[tuple[int, int, int], dict[str, object]]:
    rows = {}
    for line in open(path, errors='replace'):
        m = GO4_RE.search(line) if detail else GO_RE.search(line)
        if not m:
            continue
        frame = int(m['frame'])
        if frame_filter is not None and frame != frame_filter:
            continue
        if detail:
            xcell, ycell = int(m['xcell']), int(m['ycell'])
            if xcell not in (0, 3) or ycell not in (0, 3):
                continue
            part = (1 if xcell == 3 else 0) + (2 if ycell == 3 else 0)
        else:
            part = int(m['part'])
        key = (int(m['poc']), int(m['mb']), part)
        rows[key] = {
            'frame': frame,
            'mbtype': int(m['mbtype']),
            'ref_mv': (int(m['ref']), int(m['x']), int(m['y'])),
        }
    return rows


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument('ffcolzero')
    ap.add_argument('gomotsave')
    ap.add_argument('--go-colpoc', type=int, required=True, help='Go saved-frame POC to compare')
    ap.add_argument('--go-frame', type=int, help='Go saved-frame frame_num to disambiguate repeated POCs')
    ap.add_argument('--use-detail', action='store_true', help='compare against GOMOTSAVE4 x8*3/y8*3 representative cells')
    ap.add_argument('--mb', type=int)
    ap.add_argument('--part', type=int)
    ap.add_argument('--ff-ref0', type=int)
    ap.add_argument('--ff-ref1', type=int)
    ap.add_argument('--zero-eligible', action='store_true', help='only FF rows with small colocated MV')
    ap.add_argument('--limit', type=int, default=20)
    ap.add_argument('--fail-on-diff', action='store_true')
    args = ap.parse_args()
    ff = load_ff(args.ffcolzero)
    go = load_go(args.gomotsave, args.go_frame, args.use_detail)
    compared = diffs = 0
    seen: set[tuple[int, int, tuple[int, int, int]]] = set()
    for f in ff:
        if args.mb is not None and f['mb'] != args.mb:
            continue
        if args.part is not None and f['part'] != args.part:
            continue
        if args.ff_ref0 is not None and f['ref0'] != args.ff_ref0:
            continue
        if args.ff_ref1 is not None and f['ref1'] != args.ff_ref1:
            continue
        if args.zero_eligible and (abs(f['ref_mv'][1]) > 1 or abs(f['ref_mv'][2]) > 1):
            continue
        dedupe_key = (f['mb'], f['part'], f['ref_mv'])
        if dedupe_key in seen:
            continue
        seen.add(dedupe_key)
        compared += 1
        g = go.get((args.go_colpoc, f['mb'], f['part']))
        if g is None:
            print(f'mb={f["mb"]:04d} part={f["part"]} missing_go ff_ref_mv={f["ref_mv"]}')
            diffs += 1
        elif g['ref_mv'] != f['ref_mv']:
            print(f'mb={f["mb"]:04d} part={f["part"]} ff_ref_mv={f["ref_mv"]} go_ref_mv={g["ref_mv"]} go_frame={g["frame"]} go_mbtype={g["mbtype"]}')
            diffs += 1
        if diffs >= args.limit:
            break
    print(f'compared={compared} diffs={diffs}')
    if args.fail_on_diff and (diffs or compared == 0):
        raise SystemExit(1)


if __name__ == '__main__':
    main()
