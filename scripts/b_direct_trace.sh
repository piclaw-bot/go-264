#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
INPUT="${1:-/workspace/tmp/bbb_annexb.h264}"
OUTDIR="${2:-/workspace/tmp/go264-b-direct-trace}"
FRAMES="${FRAMES:-10}"
MB_LIMIT="${MB_LIMIT:-40}"
FFSRC="${FFMPEG_SRC:-/workspace/tmp/ffmpeg-7.1.3}"
FFMPEG="${FFMPEG:-$FFSRC/ffmpeg}"

patch_ffmpeg_direct_trace() {
  python3 - "$FFSRC/libavcodec/h264_direct.c" "$MB_LIMIT" <<'PY'
from pathlib import Path
import sys
p = Path(sys.argv[1])
mb_limit = sys.argv[2]
s = p.read_text()
if 'GO264_FFMPEG_DIRECT_TRACE' in s:
    sys.exit(0)
if '#include <stdlib.h>' not in s:
    s = s.replace('#include "h264dec.h"', '#include "h264dec.h"\n#include <stdlib.h>\n#include <stdio.h>')
old = '''void ff_h264_pred_direct_motion(const H264Context *const h, H264SliceContext *sl,
                                int *mb_type)
{
    if (sl->direct_spatial_mv_pred)
        pred_spatial_direct_motion(h, sl, mb_type);
    else
        pred_temp_direct_motion(h, sl, mb_type);
}'''
new = f'''void ff_h264_pred_direct_motion(const H264Context *const h, H264SliceContext *sl,
                                int *mb_type)
{{
    if (sl->direct_spatial_mv_pred)
        pred_spatial_direct_motion(h, sl, mb_type);
    else
        pred_temp_direct_motion(h, sl, mb_type);
    if (getenv("GO264_FFMPEG_DIRECT_TRACE") && sl->mb_x < {mb_limit}) {{
        int mb = sl->mb_x + sl->mb_y * h->mb_width;
        int s0 = scan8[0];
        int s1 = scan8[4];
        int s2 = scan8[8];
        int s3 = scan8[12];
        fprintf(stderr,
                "FFDIRECT mb=%04d x=%02d y=%02d frame=%d spatial=%d mb_type=%d "
                "ref0=%d ref1=%d mv0={{%d,%d}} mv1={{%d,%d}} "
                "sub0=%d sub1=%d sub2=%d sub3=%d "
                "submv0={{%d,%d}} submv1={{%d,%d}} submv2={{%d,%d}} submv3={{%d,%d}}\\n",
                mb, sl->mb_x, sl->mb_y, h->poc.frame_num, sl->direct_spatial_mv_pred, *mb_type,
                sl->ref_cache[0][s0], sl->ref_cache[1][s0],
                sl->mv_cache[0][s0][0], sl->mv_cache[0][s0][1],
                sl->mv_cache[1][s0][0], sl->mv_cache[1][s0][1],
                sl->sub_mb_type[0], sl->sub_mb_type[1], sl->sub_mb_type[2], sl->sub_mb_type[3],
                sl->mv_cache[0][s0][0], sl->mv_cache[0][s0][1],
                sl->mv_cache[0][s1][0], sl->mv_cache[0][s1][1],
                sl->mv_cache[0][s2][0], sl->mv_cache[0][s2][1],
                sl->mv_cache[0][s3][0], sl->mv_cache[0][s3][1]);
    }}
}}'''
if old not in s:
    raise SystemExit('ffmpeg h264_direct.c direct-motion hook target not found')
p.write_text(s.replace(old, new))
PY
}

mkdir -p "$OUTDIR"
patch_ffmpeg_direct_trace
(cd "$FFSRC" && make -j"${MAKE_JOBS:-$(nproc 2>/dev/null || echo 2)}" ffmpeg >/tmp/go264-ffmpeg-direct-build.log)
GO264_FFMPEG_DIRECT_TRACE=1 "$FFMPEG" -y -threads 1 -hide_banner \
  -i "$INPUT" -frames:v "$FRAMES" -pix_fmt yuv420p -f rawvideo /dev/null \
  >"$OUTDIR/ffmpeg.stdout" 2>"$OUTDIR/ffmpeg.direct.trace" || true

grep '^FFDIRECT' "$OUTDIR/ffmpeg.direct.trace" >"$OUTDIR/ffdirect.rows" || true
python3 - "$OUTDIR/ffdirect.rows" <<'PY'
import re, sys
from collections import Counter, defaultdict
rows = []
pat = re.compile(r'FFDIRECT mb=(\d+)(?: x=(\d+) y=(\d+))? frame=(\d+) spatial=(\d+) mb_type=([^ ]+) ref0=([^ ]+) ref1=([^ ]+) mv0=\{(-?\d+),(-?\d+)\} mv1=\{(-?\d+),(-?\d+)\}')
for line in open(sys.argv[1], errors='replace'):
    m = pat.search(line)
    if m:
        mb, x, y, frame, spatial, mbtype, ref0, ref1, mv0x, mv0y, mv1x, mv1y = m.groups()
        rows.append({
            'mb': int(mb), 'frame': int(frame), 'spatial': int(spatial),
            'mbtype': mbtype, 'ref0': int(ref0), 'ref1': int(ref1),
            'mv0': (int(mv0x), int(mv0y)), 'mv1': (int(mv1x), int(mv1y)),
        })
print(f'direct_rows={len(rows)}')
by_frame = Counter(r['frame'] for r in rows)
for frame, count in sorted(by_frame.items())[:20]:
    spatial = Counter(r['spatial'] for r in rows if r['frame'] == frame)
    refs = Counter((r['ref0'], r['ref1']) for r in rows if r['frame'] == frame)
    mv0 = Counter(r['mv0'] for r in rows if r['frame'] == frame)
    print(f'frame={frame} rows={count} spatial={dict(spatial)} refs={dict(refs.most_common(4))} mv0={dict(mv0.most_common(4))}')
PY

echo "trace=$OUTDIR/ffdirect.rows"
