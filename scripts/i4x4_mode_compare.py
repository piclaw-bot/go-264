#!/usr/bin/env python3
"""Compare Go trace264 I4x4 raw/final modes with FFmpeg FFMODE I4x4/writeback rows."""

from __future__ import annotations

import argparse
import re
from pathlib import Path

GO_RE = re.compile(r"mb=(\d+) x=(\d+) y=(\d+) .*?8x8=false .*?i4mode=\[([^\]]+)\](?: i4pred=\[([^\]]+)\])? i4final=\[([^\]]+)\]")
FF_I4_RE = re.compile(r"FFMODE part=i4x4 mb=(\d+) blk=(\d+) x=(\d+) y=(\d+) pred=(-?\d+) mode=(-?\d+)")
FF_WRITE_RE = re.compile(r"FFMODE part=i4write mb=(\d+) x=(\d+) y=(\d+) stored=\[([^\]]+)\]")


def ints(text: str) -> list[int]:
    return [int(v) for v in text.split()]


def parse_go(path: Path) -> dict[int, dict[str, object]]:
    rows: dict[int, dict[str, object]] = {}
    for line in path.read_text(errors="replace").splitlines():
        match = GO_RE.search(line)
        if not match:
            continue
        mb = int(match.group(1))
        rows[mb] = {
            "x": int(match.group(2)),
            "y": int(match.group(3)),
            "raw": ints(match.group(4)),
            "pred": ints(match.group(5)) if match.group(5) else [],
            "final": ints(match.group(6)),
        }
    return rows


def parse_ff(path: Path) -> tuple[dict[int, dict[int, dict[str, int]]], dict[int, list[int]]]:
    modes: dict[int, dict[int, dict[str, int]]] = {}
    writes: dict[int, list[int]] = {}
    for line in path.read_text(errors="replace").splitlines():
        m = FF_I4_RE.search(line)
        if m:
            mb = int(m.group(1))
            blk = int(m.group(2))
            modes.setdefault(mb, {}).setdefault(blk, {"pred": int(m.group(5)), "mode": int(m.group(6))})
            continue
        m = FF_WRITE_RE.search(line)
        if m:
            mb = int(m.group(1))
            writes.setdefault(mb, ints(m.group(4)))
    return modes, writes


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("go_trace", type=Path)
    parser.add_argument("ffmpeg_trace", type=Path)
    parser.add_argument("--mb", type=int)
    parser.add_argument("--limit", type=int, default=50)
    parser.add_argument("--mismatches-only", action="store_true")
    args = parser.parse_args()

    go = parse_go(args.go_trace)
    ff_modes, ff_writes = parse_ff(args.ffmpeg_trace)
    common = sorted(set(go) & (set(ff_modes) | set(ff_writes)))
    print(f"go_mbs={len(go)} ff_i4_mbs={len(ff_modes)} ff_write_mbs={len(ff_writes)} common={len(common)}")
    shown = 0
    for mb in common:
        if args.mb is not None and mb != args.mb:
            continue
        g = go[mb]
        raw = g["raw"]
        pred = g["pred"]
        final = g["final"]
        ff = ff_modes.get(mb, {})
        write = ff_writes.get(mb)
        mismatch = False
        parts = [f"mb={mb:04d} x={g['x']} y={g['y']}"]
        if write is not None:
            parts.append(f"ff_write={write}")
            # FFmpeg's compact writeback table has seven edge/cache entries, not Go's 16 raster modes.
        for blk in sorted(ff):
            if blk < len(final) and final[blk] != ff[blk]["mode"]:
                mismatch = True
            if pred and blk < len(pred) and pred[blk] != ff[blk]["pred"]:
                mismatch = True
        if args.mismatches_only and not mismatch:
            continue
        parts.append(f"go_raw={raw}")
        if pred:
            parts.append(f"go_pred={pred}")
        parts.append(f"go_final={final}")
        if ff:
            parts.append("ff_modes=" + str([(blk, row["pred"], row["mode"]) for blk, row in sorted(ff.items())]))
        print(" ".join(parts))
        shown += 1
        if shown >= args.limit:
            break
    return 0 if shown or not args.mismatches_only else 1


if __name__ == "__main__":
    raise SystemExit(main())
