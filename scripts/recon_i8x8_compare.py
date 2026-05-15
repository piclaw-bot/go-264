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

GO_RE = re.compile(r"GORECON part=i8x8 (?:frame=(\d+) )?.*?mb=(\d+) b8=(\d+) x=(\d+) y=(\d+) .*?syntax_mode=(-?\d+) recon_mode=(-?\d+) .*?predsum=(-?\d+) .*?outsum=(-?\d+) .*?block_out=\[([^\]]*)\]")
FF_RE = re.compile(r"FFRECON part=i8x8 (?:frame=(\d+) )?.*?mb=(\d+) b8=(\d+) x=(\d+) y=(\d+) .*?mode=(-?\d+) .*?predsum=(-?\d+) .*?outsum=(-?\d+) .*?block_out=\[([^\]]*)\]")


def parse_blocks(path: Path, pattern: re.Pattern[str]) -> list[dict[str, object]]:
    blocks: list[dict[str, object]] = []
    occurrence_by_key: dict[tuple[int, int], int] = {}
    for line in path.read_text(errors="replace").splitlines():
        match = pattern.search(line)
        if not match:
            continue
        frame = int(match.group(1)) if match.group(1) is not None else None
        mb = int(match.group(2))
        b8 = int(match.group(3))
        x = int(match.group(4))
        y = int(match.group(5))
        key = (mb, b8)
        occurrence = occurrence_by_key.get(key, 0)
        occurrence_by_key[key] = occurrence + 1
        if pattern is GO_RE:
            syntax_mode = int(match.group(6))
            recon_mode = int(match.group(7))
            ff_mode = None
            predsum = int(match.group(8))
            outsum = int(match.group(9))
            block_out = [int(v) for v in match.group(10).split()]
        else:
            syntax_mode = None
            recon_mode = None
            ff_mode = int(match.group(6))
            predsum = int(match.group(7))
            outsum = int(match.group(8))
            block_out = [int(v) for v in match.group(9).split()]
        blocks.append({
            "key": key,
            "frame_key": frame if frame is not None else occurrence,
            "frame": frame,
            "occurrence": occurrence,
            "x": x,
            "y": y,
            "syntax_mode": syntax_mode,
            "recon_mode": recon_mode,
            "ff_mode": ff_mode,
            "predsum": predsum,
            "outsum": outsum,
            "block_out": block_out,
            "line": line,
        })
    return blocks


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("go_log", type=Path)
    parser.add_argument("ffmpeg_log", type=Path)
    parser.add_argument("--limit", type=int, default=20)
    parser.add_argument("--frame", type=int, help="only compare one decoded frame index when present in Go trace")
    parser.add_argument("--mb", type=int, help="only compare one macroblock address")
    parser.add_argument("--b8", type=int, choices=range(4), metavar="0..3", help="only compare one luma 8x8 block")
    parser.add_argument("--occurrence", type=int, help="only compare one occurrence index for repeated mb/b8 keys")
    parser.add_argument("--max-pred-delta", type=int, help="only show rows whose absolute prediction delta is at most this value")
    parser.add_argument("--min-pred-delta", type=int, help="only show rows whose absolute prediction delta is at least this value")
    parser.add_argument("--summary-by-mode", action="store_true", help="summarize absolute deltas by Go/FFmpeg predictor mode tuple")
    parser.add_argument("--sort", choices=("out", "pred", "res"), default="out", help="sort rows by absolute output, prediction, or residual delta")
    args = parser.parse_args()

    go_blocks = parse_blocks(args.go_log, GO_RE)
    ff_blocks = parse_blocks(args.ffmpeg_log, FF_RE)
    go = {(block["frame_key"], block["key"], block["occurrence"]): block for block in go_blocks}
    ff = {(block["frame_key"], block["key"], block["occurrence"]): block for block in ff_blocks}
    common = sorted(set(go) & set(ff))
    print(f"go_blocks={len(go_blocks)} ffmpeg_blocks={len(ff_blocks)} common={len(common)}")
    if not common:
        return 1

    rows = []
    for key in common:
        frame_key, (mb, b8), occurrence = key
        occurrence = int(occurrence)
        frame = go[key]["frame"]
        if args.frame is not None and frame != args.frame:
            continue
        if args.mb is not None and mb != args.mb:
            continue
        if args.b8 is not None and b8 != args.b8:
            continue
        if args.occurrence is not None and occurrence != args.occurrence:
            continue
        g = go[key]
        f = ff[key]
        out_delta = int(g["outsum"]) - int(f["outsum"])
        pred_delta = int(g["predsum"]) - int(f["predsum"])
        gb = g["block_out"]
        fb = f["block_out"]
        if args.max_pred_delta is not None and abs(pred_delta) > args.max_pred_delta:
            continue
        if args.min_pred_delta is not None and abs(pred_delta) < args.min_pred_delta:
            continue
        res_delta = out_delta - pred_delta
        block_delta = [int(gb[i]) - int(fb[i]) for i in range(min(len(gb), len(fb)))]
        rows.append((abs(out_delta), key, out_delta, pred_delta, res_delta, block_delta, g, f))

    if not rows:
        print("no matching blocks after filters")
        return 1

    if args.summary_by_mode:
        groups: dict[tuple[object, object, object], dict[str, int]] = {}
        for _, _key, out_delta, pred_delta, res_delta, _block_delta, g, f in rows:
            mode_key = (g["syntax_mode"], g["recon_mode"], f["ff_mode"])
            group = groups.setdefault(mode_key, {"count": 0, "abs_out": 0, "abs_pred": 0, "abs_res": 0, "signed_out": 0, "signed_res": 0})
            group["count"] += 1
            group["abs_out"] += abs(out_delta)
            group["abs_pred"] += abs(pred_delta)
            group["abs_res"] += abs(res_delta)
            group["signed_out"] += out_delta
            group["signed_res"] += res_delta
        for (syntax_mode, recon_mode, ff_mode), group in sorted(groups.items(), key=lambda item: item[1]["abs_out"], reverse=True)[: args.limit]:
            print(
                f"go_mode={syntax_mode}/{recon_mode} ff_mode={ff_mode} "
                f"count={group['count']} abs_out={group['abs_out']} "
                f"abs_pred={group['abs_pred']} abs_res={group['abs_res']} "
                f"signed_out={group['signed_out']} signed_res={group['signed_res']}"
            )
        return 0

    sort_index = {"out": 0, "pred": 3, "res": 4}[args.sort]
    rows.sort(reverse=True, key=lambda item: abs(item[sort_index]))
    for _, (frame_key, (mb, b8), _occurrence), out_delta, pred_delta, res_delta, block_delta, g, f in rows[: args.limit]:
        occurrence = int(g["occurrence"])
        frame = g["frame"]
        frame_label = f"frame={frame}" if frame is not None else f"occ={occurrence}"
        print(
            f"{frame_label} occ={occurrence} mb={mb:04d} b8={b8} x={g['x']} y={g['y']} "
            f"go_mode={g['syntax_mode']}/{g['recon_mode']} ff_mode={f['ff_mode']} "
            f"out_delta={out_delta:+d} pred_delta={pred_delta:+d} res_delta={res_delta:+d} "
            f"block_delta={block_delta} go_out={g['outsum']} ff_out={f['outsum']}"
        )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
