#!/usr/bin/env python3
"""Compare Go and FFmpeg luma Intra_8x8 reconstruction trace lines.

Reads stderr logs produced with GO264_RECON_TRACE=1 and
GO264_FFMPEG_RECON_TRACE=1 and reports the largest per-block output-sum
mismatches. The trace formats use different field names, but both carry
mb/b8/predsum/outsum/block_out for luma I8x8 blocks.
"""

from __future__ import annotations

import argparse
import re
from pathlib import Path

GO_RE = re.compile(r"GORECON part=i8x8 .*?mb=(\d+) b8=(\d+) .*?predsum=(-?\d+) .*?outsum=(-?\d+) .*?block_out=\[([^\]]*)\]")
FF_RE = re.compile(r"FFRECON part=i8x8 .*?mb=(\d+) b8=(\d+) .*?predsum=(-?\d+) .*?outsum=(-?\d+) .*?block_out=\[([^\]]*)\]")


def parse_blocks(path: Path, pattern: re.Pattern[str]) -> dict[tuple[int, int], dict[str, object]]:
    blocks: dict[tuple[int, int], dict[str, object]] = {}
    for line in path.read_text(errors="replace").splitlines():
        match = pattern.search(line)
        if not match:
            continue
        mb = int(match.group(1))
        b8 = int(match.group(2))
        predsum = int(match.group(3))
        outsum = int(match.group(4))
        block_out = [int(v) for v in match.group(5).split()]
        blocks[(mb, b8)] = {"predsum": predsum, "outsum": outsum, "block_out": block_out, "line": line}
    return blocks


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("go_log", type=Path)
    parser.add_argument("ffmpeg_log", type=Path)
    parser.add_argument("--limit", type=int, default=20)
    args = parser.parse_args()

    go = parse_blocks(args.go_log, GO_RE)
    ff = parse_blocks(args.ffmpeg_log, FF_RE)
    common = sorted(set(go) & set(ff))
    print(f"go_blocks={len(go)} ffmpeg_blocks={len(ff)} common={len(common)}")
    if not common:
        return 1

    rows = []
    for key in common:
        g = go[key]
        f = ff[key]
        out_delta = int(g["outsum"]) - int(f["outsum"])
        pred_delta = int(g["predsum"]) - int(f["predsum"])
        gb = g["block_out"]
        fb = f["block_out"]
        block_delta = [int(gb[i]) - int(fb[i]) for i in range(min(len(gb), len(fb)))]
        rows.append((abs(out_delta), key, out_delta, pred_delta, block_delta, g, f))

    rows.sort(reverse=True, key=lambda item: item[0])
    for _, (mb, b8), out_delta, pred_delta, block_delta, g, f in rows[: args.limit]:
        print(
            f"mb={mb:04d} b8={b8} out_delta={out_delta:+d} "
            f"pred_delta={pred_delta:+d} block_delta={block_delta} "
            f"go_out={g['outsum']} ff_out={f['outsum']}"
        )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
