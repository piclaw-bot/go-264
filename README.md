# go-264

H.264/AVC encoder and decoder in pure Go with SIMD assembly and optional GPU acceleration.

## Status: Design Phase

See [docs/design.md](docs/design.md) for the full implementation approach.

## Goals

- **Pure Go** — no CGo, static binary, cross-platform
- **SIMD assembly** — AVX2+FMA (amd64), NEON (arm64) for hot paths
- **GPU compute** — optional PTX kernels via purego (no CUDA toolkit needed)
- **Conformant** — pass ITU H.264 decoder conformance tests
- **Practical** — encode 1080p in real-time on modern hardware

## Architecture

```
go-264/
├── nal/          NAL parser (SPS, PPS, SEI, bitstream)
├── slice/        Slice/macroblock decoding
├── pred/         Intra/inter prediction
├── transform/    DCT, quantization (+ SIMD assembly)
├── entropy/      CAVLC, CABAC
├── filter/       Deblocking filter (+ SIMD)
├── frame/        YUV frame management, DPB
├── encode/       Encoder pipeline, rate control
├── me/           Motion estimation (+ SIMD + GPU)
├── gpu/          CUDA via purego (reused from go-tinygrad)
└── cmd/
    ├── decode/   CLI decoder
    └── encode/   CLI encoder
```

## Leveraging go-tinygrad

This project reuses the GPU compute framework from [go-tinygrad](https://github.com/rcarmo/go-tinygrad):
- CUDA bindings via purego (no CGo)
- DevBuf device-agnostic buffers
- PTX kernel compilation at runtime
- Graceful CPU fallback when no GPU

## License

MIT
