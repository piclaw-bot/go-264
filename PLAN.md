# go-264

H.264/AVC encoder and decoder in pure Go with SIMD assembly.

## x264 API Surface

x264 exposes ~20 public functions. The core pipeline is simple:

```c
// Setup
x264_param_default(&param);
x264_param_apply_profile(&param, "high");
x264_t *enc = x264_encoder_open(&param);

// Encode loop
x264_picture_t pic_in, pic_out;
x264_nal_t *nals;
int i_nal;
while (have_frames) {
    x264_picture_init(&pic_in);
    pic_in.img.i_csp = X264_CSP_I420;
    pic_in.img.plane[0] = y_data;  // Y plane
    pic_in.img.plane[1] = u_data;  // U plane
    pic_in.img.plane[2] = v_data;  // V plane
    
    x264_encoder_encode(enc, &nals, &i_nal, &pic_in, &pic_out);
    // nals[] contains NAL units ready for muxing
}

// Flush + close
x264_encoder_close(enc);
```

Key types:
- `x264_param_t` — 200+ fields: resolution, profile, rate control, threading
- `x264_picture_t` — input/output frame (YUV planes + metadata)
- `x264_nal_t` — output NAL unit (type, priority, payload bytes)
- `x264_t` — opaque encoder state

## Go API Design

```go
// Encoder
enc, _ := h264.NewEncoder(h264.Config{
    Width: 1920, Height: 1080,
    Profile: h264.ProfileHigh,
    Preset:  h264.PresetMedium,
    RateControl: h264.CRF(23),
})
defer enc.Close()

nals, _ := enc.Encode(frame)  // frame is *h264.YUVFrame
for _, nal := range nals {
    out.Write(nal.Bytes())     // write to file/network
}

// Decoder
dec := h264.NewDecoder()
for {
    frame, err := dec.Decode(nalBytes)
    if err == h264.ErrNeedMoreData { continue }
    // frame.Y, frame.U, frame.V contain decoded planes
}
```

## Implementation Approach

### Phase 1: Decoder (Baseline Profile)

The decoder is simpler and exercises all the core data structures.

```
nal/           NAL unit parser (start codes, emulation prevention)
  bitstream.go   — exp-Golomb, CABAC/CAVLC bit reader
  nal.go         — NAL unit types, header parsing
  sps.go         — Sequence Parameter Set
  pps.go         — Picture Parameter Set
  sei.go         — Supplemental Enhancement Info

slice/         Slice layer decoding
  slice.go       — slice header, macroblock loop
  mb.go          — macroblock types, sub-mb partition

pred/          Prediction
  intra.go       — intra 4x4 (9 modes), 8x8, 16x16
  inter.go       — inter prediction, motion compensation
  mv.go          — motion vector decoding, spatial/temporal prediction

transform/     Transform + quantization
  dct.go         — 4x4 integer DCT, inverse DCT
  quant.go       — quantization, dequantization, QP scaling
  dct_amd64.s    — SIMD: 4-wide butterfly
  dct_arm64.s    — NEON: 4-wide butterfly

entropy/       Entropy coding
  cavlc.go       — CAVLC (Context-Adaptive Variable-Length Coding)
  cabac.go       — CABAC (Context-Adaptive Binary Arithmetic Coding)
  tables.go      — coding tables (zig-zag scan, level/run tables)

filter/        In-loop deblocking
  deblock.go     — edge filtering (luma/chroma, strong/weak)
  deblock_amd64.s — SIMD deblocking

frame/         Frame management
  frame.go       — YUV frame, plane access
  dpb.go         — decoded picture buffer (reference frames)
  reorder.go     — frame reordering (POC)
```

### Phase 2: Encoder (Baseline → High Profile)

```
encode/        Encoder pipeline
  encoder.go     — top-level encode loop
  ratecontrol.go — CRF, CBR, VBR rate control
  lookahead.go   — frame type decision

me/            Motion estimation (biggest compute cost)
  fullsearch.go  — full-pixel exhaustive search
  diamond.go     — diamond/hexagon search patterns
  subpel.go      — half-pel, quarter-pel refinement
  sad_amd64.s    — SIMD SAD (sum of absolute differences)
  satd_amd64.s   — SIMD SATD (Hadamard transform)
  sad_arm64.s    — NEON SAD
  me_gpu.go      — GPU-accelerated motion search (PTX)

analysis/      Mode decision
  intra.go       — intra mode analysis (RDO)
  inter.go       — inter mode analysis (RDO)
  rdo.go         — rate-distortion optimization
```

### Phase 3: GPU Acceleration

Reuse the go-tinygrad GPU framework:

```
gpu/           GPU compute (from go-tinygrad)
  cuda_purego.go  — CUDA via dlopen (no CGo)
  devbuf.go       — device-agnostic buffers
  me_ptx.go       — motion estimation PTX kernel
  dct_ptx.go      — batch DCT/IDCT on GPU
  deblock_ptx.go  — parallel deblocking
```

GPU-acceleratable operations and expected speedup:

| Operation | CPU (SIMD) | GPU (PTX) | Speedup |
|---|---|---|---|
| SAD 16×16 (full search) | ~1 cycle/pixel | 1024 blocks parallel | 50-100× |
| SATD 4×4 (Hadamard) | 8×8 butterfly | batch all MBs | 20-50× |
| DCT 4×4 batch | 4-wide SIMD | 1 MB per thread | 10-20× |
| Deblocking | sequential edges | parallel edges | 5-10× |
| CABAC | sequential (cannot GPU) | — | — |

### What transfers from go-tinygrad

| Component | Reuse | Adaptation |
|---|---|---|
| `gpu/cuda_purego.go` | Direct | Same CUDA bindings |
| `gpu/devbuf.go` | Direct | Same CPU/GPU dispatch |
| `gpu/sgemm_ptx.go` | Reference | Replace with SAD/DCT PTX |
| `gpu/nv_ioctl.go` | Direct | Same device init |
| SIMD assembly pattern | Template | SAD/SATD instead of dot/gemm |
| Build system | Direct | Same pure Go + assembly |

### Profiles and Levels

Start with Baseline (most common for real-time):
- **Baseline**: I + P slices, CAVLC only, no B-frames
- **Main**: + B-frames, CABAC, weighted prediction
- **High**: + 8×8 transform, 8×8 intra, adaptive quantization

Levels (resolution/bitrate caps):
- Level 3.0: 720p30
- Level 4.0: 1080p30
- Level 4.1: 1080p60
- Level 5.1: 4K30

### Integer-only arithmetic

H.264's transform is specifically designed for integer arithmetic (no floating-point drift):
- 4×4 integer DCT (not true DCT — scaled Hadamard-like)
- All prediction is integer pixel operations
- Deblocking uses clipping and integer thresholds
- Only rate-distortion analysis uses floating-point (encoder-side)

This maps perfectly to Go's integer SIMD:
- `VPADDD` (AVX2) for 8-wide int32 add
- `VPABSD` for absolute difference
- `VPMADDWD` for multiply-accumulate
- ARM NEON: `VABD`, `VADDL`, `VMLA`

### Testing Strategy

1. **Bitstream conformance**: Decode ITU conformance test vectors
2. **Round-trip**: Encode → Decode → PSNR check
3. **Numpy reference**: DCT/IDCT/quantize verified against scipy
4. **SIMD verification**: Assembly output matches scalar reference
5. **GPU verification**: GPU output matches CPU for all kernels
