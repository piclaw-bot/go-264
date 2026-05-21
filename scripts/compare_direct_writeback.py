#!/usr/bin/env python3
"""Compare GODIRECT direct sub-MV representatives with GOMOTWRITE cache output.

This isolates cases where direct prediction derived one representative but B motion
cache write-back stored a different value for the same 8x8 part.
"""
from __future__ import annotations
import argparse
import re

DIRECT_RE = re.compile(
    r'GODIRECT mb=(?P<mb>\d+).*?poc=(?P<poc>-?\d+)\b.*?spatial=(?P<spatial>\d+)\b.*?'
    r'submv0=\{(?P<x0>-?\d+),(?P<y0>-?\d+)\} submv1=\{(?P<x1>-?\d+),(?P<y1>-?\d+)\} '
    r'submv2=\{(?P<x2>-?\d+),(?P<y2>-?\d+)\} submv3=\{(?P<x3>-?\d+),(?P<y3>-?\d+)\}'
)
WRITE_RE = re.compile(
    r'GOMOTWRITE mb=(?P<mb>\d+).*?(?:poc=(?P<poc>-?\d+) )?type=(?P<mbtype>\d+) part=(?P<part>\d+) ref0=(?P<ref0>-?\d+) mv0=\{(?P<mvx>-?\d+),(?P<mvy>-?\d+)\}.*?'
    r'sub0=\{(?P<subx>-?\d+),(?P<suby>-?\d+)\}'
)

FF_DIRECT_RE = re.compile(
    r'FFDIRECT mb=(?P<mb>\d+).*?frame=(?P<frame>-?\d+)\b.*?spatial=(?P<spatial>\d+)\b.*?'
    r'submv0=\{(?P<x0>-?\d+),(?P<y0>-?\d+)\} submv1=\{(?P<x1>-?\d+),(?P<y1>-?\d+)\} '
    r'submv2=\{(?P<x2>-?\d+),(?P<y2>-?\d+)\} submv3=\{(?P<x3>-?\d+),(?P<y3>-?\d+)\}'
)


def load_direct(path: str, poc_filter: int | None = None, spatial_filter: int | None = None, occurrence: int = 0) -> dict[int, tuple[tuple[int, int], ...]]:
    out = {}
    seen: dict[int, int] = {}
    for line in open(path, errors='replace'):
        m = DIRECT_RE.search(line)
        if not m:
            continue
        if poc_filter is not None and int(m['poc']) != poc_filter:
            continue
        if spatial_filter is not None and int(m['spatial']) != spatial_filter:
            continue
        mb = int(m['mb'])
        occ = seen.get(mb, 0)
        seen[mb] = occ + 1
        if occ != occurrence:
            continue
        out[mb] = tuple((int(m[f'x{i}']), int(m[f'y{i}'])) for i in range(4))
    return out

def load_ff_direct(path: str | None, frame_filter: int | None = None, spatial_filter: int | None = None, occurrence: int = 0) -> dict[int, tuple[tuple[int, int], ...]]:
    if not path:
        return {}
    out = {}
    seen: dict[int, int] = {}
    for line in open(path, errors='replace'):
        m = FF_DIRECT_RE.search(line)
        if not m:
            continue
        if frame_filter is not None and int(m['frame']) != frame_filter:
            continue
        if spatial_filter is not None and int(m['spatial']) != spatial_filter:
            continue
        mb = int(m['mb'])
        occ = seen.get(mb, 0)
        seen[mb] = occ + 1
        if occ != occurrence:
            continue
        out[mb] = tuple((int(m[f'x{i}']), int(m[f'y{i}'])) for i in range(4))
    return out


def load_write(path: str, poc_filter: int | None = None, occurrence: int = 0) -> dict[tuple[int, int], dict[str, object]]:
    out = {}
    seen: dict[tuple[int, int], int] = {}
    for line in open(path, errors='replace'):
        m = WRITE_RE.search(line)
        if not m:
            continue
        if poc_filter is not None:
            # POC-qualified comparisons must not consume older GOMOTWRITE rows
            # that predate poc= tracing; MB addresses repeat across pictures.
            if m['poc'] is None or int(m['poc']) != poc_filter:
                continue
        key = (int(m['mb']), int(m['part']))
        occ = seen.get(key, 0)
        seen[key] = occ + 1
        if occ != occurrence:
            continue
        out[key] = {
            'poc': int(m['poc']) if m['poc'] is not None else None,
            'mbtype': int(m['mbtype']),
            'ref0': int(m['ref0']),
            'mv0': (int(m['mvx']), int(m['mvy'])),
            'sub0': (int(m['subx']), int(m['suby'])),
        }
    return out

def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument('godirect')
    ap.add_argument('gomotwrite')
    ap.add_argument('--ffdirect', help='optional FFDIRECT rows; when set, only report write MVs that differ from FF sub-MVs')
    ap.add_argument('--ff-frame', type=int, help='FF frame filter for --ffdirect')
    ap.add_argument('--mb', type=int)
    ap.add_argument('--from-mb', type=int, dest='from_mb')
    ap.add_argument('--to-mb', type=int, dest='to_mb')
    ap.add_argument('--poc', type=int, help='only compare rows for this Go POC')
    ap.add_argument('--spatial', type=int, choices=(0, 1), help='only compare GODIRECT rows with this spatial flag')
    ap.add_argument('--direct-occurrence', type=int, default=0, help='nth matching GODIRECT occurrence per MB after filters')
    ap.add_argument('--write-occurrence', type=int, default=0, help='nth matching GOMOTWRITE occurrence per MB/part after filters')
    ap.add_argument('--mb-type', type=int, dest='mb_type', help='only compare write rows with this Go MB type')
    ap.add_argument('--ref0', type=int, help='only compare write rows with this ref0')
    ap.add_argument('--only-zero-direct', action='store_true', help='only compare parts whose direct sub-MV representative is zero')
    ap.add_argument('--limit', type=int, default=50)
    ap.add_argument('--fail-on-diff', action='store_true')
    args = ap.parse_args()
    direct = load_direct(args.godirect, args.poc, args.spatial, args.direct_occurrence)
    writes = load_write(args.gomotwrite, args.poc, args.write_occurrence)
    ff_direct = load_ff_direct(args.ffdirect, args.ff_frame, args.spatial, args.direct_occurrence)
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
            if args.mb_type is not None and w['mbtype'] != args.mb_type:
                continue
            if args.ref0 is not None and w['ref0'] != args.ref0:
                continue
            if args.only_zero_direct and dmv != (0, 0):
                continue
            compared += 1
            ffmv = ff_direct.get(mb, (None, None, None, None))[part] if ff_direct else None
            mismatch = (w['mv0'] != dmv or w['sub0'] != dmv)
            if ffmv is not None:
                mismatch = w['mv0'] != ffmv
            if mismatch:
                ff_text = f' ff_sub={ffmv}' if ffmv is not None else ''
                print(f'mb={mb:04d} poc={w["poc"]} type={w["mbtype"]} part={part} direct_sub={dmv} write_sub={w["sub0"]} write_mv={w["mv0"]}{ff_text} ref0={w["ref0"]}')
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
