#!/usr/bin/env python3
"""Compare FFmpeg FFBSTATE and Go GOBSTATE CABAC state rows."""
from __future__ import annotations
import argparse
import re
from collections import defaultdict

FF_RE = re.compile(r'FFBSTATE mb=(?P<mb>\d+).*?frame=(?P<frame>\d+) kind=(?P<kind>\w+) low=(?P<low>\d+) range=(?P<range>\d+)')
GO_RE = re.compile(r'GOBSTATE mb=(?P<mb>\d+).*?poc=(?P<frame>\d+) kind=(?P<kind>\w+) low=(?P<low>\d+) range=(?P<range>\d+)')

def iv(m: re.Match[str], name: str) -> int:
    return int(m.group(name))

def load(path: str, regex: re.Pattern[str], frame: int, occurrence: int) -> dict[int, dict[str, object]]:
    out = {}
    seen: defaultdict[int, int] = defaultdict(int)
    for line in open(path, errors='replace'):
        m = regex.search(line)
        if not m or iv(m, 'frame') != frame:
            continue
        mb = iv(m, 'mb')
        occ = seen[mb]; seen[mb] += 1
        if occ != occurrence:
            continue
        out[mb] = {'kind': m.group('kind'), 'low': iv(m, 'low'), 'range': iv(m, 'range')}
    return out

def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument('ffbstate')
    ap.add_argument('gobstate')
    ap.add_argument('--ff-frame', type=int, required=True)
    ap.add_argument('--go-poc', type=int, required=True)
    ap.add_argument('--ff-occurrence', type=int, default=0)
    ap.add_argument('--go-occurrence', type=int, default=0)
    ap.add_argument('--mb', type=int)
    ap.add_argument('--from-mb', type=int, dest='from_mb')
    ap.add_argument('--to-mb', type=int, dest='to_mb')
    ap.add_argument('--compare-low', action='store_true', help='also compare raw low values; off by default because normalizations differ')
    ap.add_argument('--limit', type=int, default=50)
    ap.add_argument('--fail-on-diff', action='store_true')
    args = ap.parse_args()
    ff = load(args.ffbstate, FF_RE, args.ff_frame, args.ff_occurrence)
    go = load(args.gobstate, GO_RE, args.go_poc, args.go_occurrence)
    if not ff:
        print(f'no_ff_bstate_rows frame={args.ff_frame} occurrence={args.ff_occurrence}')
        if args.fail_on_diff:
            raise SystemExit(1)
        return
    compared = diffs = 0
    for mb in sorted(ff):
        if args.mb is not None and mb != args.mb:
            continue
        if args.from_mb is not None and mb < args.from_mb:
            continue
        if args.to_mb is not None and mb > args.to_mb:
            continue
        f = ff[mb]
        g = go.get(mb)
        if g is None:
            print(f'mb={mb:04d} missing_go ff_kind={f["kind"]} ff_range={f["range"]}')
            diffs += 1
        else:
            compared += 1
            fields = []
            if f['kind'] != g['kind']:
                fields.append('kind')
            if f['range'] != g['range']:
                fields.append('range')
            if args.compare_low and f['low'] != g['low']:
                fields.append('low')
            if fields:
                print(f'mb={mb:04d} fields={",".join(fields)} ff_kind={f["kind"]} go_kind={g["kind"]} ff_low={f["low"]} go_low={g["low"]} ff_range={f["range"]} go_range={g["range"]}')
                diffs += 1
        if diffs >= args.limit:
            break
    print(f'compared={compared} diffs={diffs}')
    if args.fail_on_diff and (diffs or compared == 0):
        raise SystemExit(1)

if __name__ == '__main__':
    main()
