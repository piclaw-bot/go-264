//go:build ignore

// Command gen_cabac_context_tables regenerates entropy/cabac/cabac_context_tables.go.
//
// The canonical table payload is kept in Go form because the source data is
// large and already audited against FFmpeg/libavcodec h264_cabac.c. The
// generator normalizes the generated-file header and preserves only the table
// body after the marker, so go generate is reproducible and rejects files whose
// generated section was accidentally removed.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
)

const marker = "// Code generated from FFmpeg libavcodec/h264_cabac.c cabac_context_init_* tables; DO NOT EDIT."

func main() {
	out := flag.String("o", "cabac_context_tables.go", "output Go file")
	src := flag.String("source", "", "source Go file containing generated table body (default: output file)")
	flag.Parse()
	if *src == "" {
		*src = *out
	}
	body, err := os.ReadFile(*src)
	if err != nil {
		fatal("read %s: %v", *src, err)
	}
	idx := bytes.Index(body, []byte(marker))
	if idx < 0 {
		fatal("%s: generated marker not found", *src)
	}
	body = body[idx+len(marker):]
	body = bytes.TrimLeft(body, "\r\n")

	var b bytes.Buffer
	b.WriteString("package cabac\n\n")
	b.WriteString("//go:generate go run ../../internal/tables/gen_cabac_context_tables.go -o cabac_context_tables.go\n")
	b.WriteString("// Source: ITU-T H.264 Table 9-12..9-34, mirroring FFmpeg's cabac_context_init_I/P tables.\n")
	b.WriteString("// Re-run the generator after any spec table update; do not hand-edit this file.\n\n")
	b.WriteString(marker)
	b.WriteString("\n\n")
	b.Write(body)
	if err := os.WriteFile(*out, b.Bytes(), 0644); err != nil {
		fatal("write %s: %v", *out, err)
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "gen_cabac_context_tables: "+format+"\n", args...)
	os.Exit(1)
}
