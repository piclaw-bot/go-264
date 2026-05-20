#!/usr/bin/env python3
"""Compare GODIRECT direct sub-MV representatives with GOMOTWRITE cache output.

This isolates cases where direct prediction derived one representative but B motion
cache write-back stored a different value for the same 8x8 part.
"""
from __future__ import annotations
import argparse
import re

DIRECT_RE = re.compile(
    r'GODIRECT mb=(?P<mb>\d+).*?poc=(?P<poc>-?\d+)\b.*?'
    r'submv0=\{(?P<x0>-?\d+),(?P<y0>-?\d+)\} submv1=\{(?P<x1>-?\d+),(?P<y1>-?\d+)\} '
    r'submv2=\{(?P<x2>-?\d+),(?P<y2>-?\d+)\} submv3=\{(?P<x3>-?\d+),(?P<y3>-?\d+)\}'
)
WRITE_RE = re.compile(
    r'GOMOTWRITE mb=(?P<mb>\d+).*?part=(?P<part>\d+) ref0=(?P<ref0>-?\d+) mv0=\{(?P<mvx>-?\d+),(?P<mvy>-?\d+)\}.*?'
    r'sub0=\{(?P<subx>-?\d+),(?P<suby>-?\d+)\}'
)

def load_direct(path: str) -> dict[int, tuple[tuple[int, int], ...]]:
    out = {}
    for line in open(path, errors='replace'):
        m = DIRECT_RE.search(line)
        if not m:
            continue
        out[int(m['mb'])] = tuple((int(m[f'x{i}']), int(m[f'y{i}'])) for i in range(4))
    return out

def load_write(path: str) -> dict[tuple[int, int], dict[str, object]]:
    out = {}
    for line in open(path, errors='replace'):
        m = WRITE_RE.search(line)
        if not m:
            continue
        out[(int(m['mb']), int(m['part']))] = {
            'ref0': int(m['ref0']),
            'mv0': (int(m['mvx']), int(m['mvy'])),
            'sub0': (int(m['subx']), int(m['suby'])),
        }
    return out

def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument('godirect')
    ap.add_argument('gomotwrite')
    ap.add_argument('--mb', type=int)
    ap.add_argument('--from-mb', type=int, dest='from_mb')
    ap.add_argument('--to-mb', type=int, dest='to_mb')
    ap.add_argument('--ref0', type=int, help='only compare write rows with this ref0')
    ap.add_argument('--only-zero-direct', action='store_true', help='only compare parts whose direct sub-MV representative is zero')
    ap.add_argument('--limit', type=int, default=50)
    ap.add_argument('--fail-on-diff', action='store_true')
    args = ap.parse_args()
    direct = load_direct(args.godirect)
    writes = load_write(args.gomotwrite)
    compared = diffs = 0
    for mb in sorted(direct):
        if args.mb is not None and mb != args.mb:
            continue
        if args.from_mb is not None and mb < args.from_mb:
            continue
        if args.to_mb is not None and mb > args.to_mb:
            continue
        for part, dmv in enumerate(direct[mb]):
            w = writes.get((mb, part))
            if w is None:
                continue
            if args.ref0 is not None and w['ref0'] != args.ref0:
                continue
            if args.only_zero_direct and dmv != (0, 0):
                continue
            compared += 1
            if w['mv0'] != dmv or w['sub0'] != dmv:
                print(f'mb={mb:04d} part={part} direct_sub={dmv} write_sub={w["sub0"]} write_mv={w["mv0"]} ref0={w["ref0"]}')
                diffs += 1
                if diffs >= args.limit:
                    print(f'compared={compared} diffs={diffs}')
                    if args.fail_on_diff:
                        raise SystemExit(1)
                    return
    print(f'compared={compared} diffs={diffs}')
    if args.fail_on_diff and diffs:
        raise SystemExit(1)

if __name__ == '__main__':
    main()
