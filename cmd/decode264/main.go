package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"time"

	"github.com/rcarmo/go-264/decode"
)

func main() {
	input := flag.String("i", "", "input H.264 Annex B file")
	outDir := flag.String("o", ".", "output directory for decoded frames")
	format := flag.String("f", "png", "output format: png or yuv")
	flag.Parse()

	if *input == "" {
		fmt.Fprintln(os.Stderr, "Usage: decode264 -i input.h264 [-o outdir] [-f png|yuv]")
		os.Exit(1)
	}

	data, err := os.ReadFile(*input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Input: %s (%d bytes)\n", *input, len(data))

	dec := decode.NewDecoder()
	start := time.Now()
	frames, err := dec.Decode(data)
	elapsed := time.Since(start)

	if err != nil {
		fmt.Fprintf(os.Stderr, "decode: %v\n", err)
		os.Exit(1)
	}

	// Print SPS info
	for _, sps := range dec.SPS {
		fmt.Printf("Stream: %dx%d, profile=%d, level=%d\n",
			sps.Width, sps.Height, sps.ProfileIDC, sps.LevelIDC)
	}

	fmt.Printf("Decoded %d frames in %v (%.1f fps)\n",
		len(frames), elapsed.Round(time.Millisecond),
		float64(len(frames))/elapsed.Seconds())

	// Write output frames
	os.MkdirAll(*outDir, 0755)
	for i, f := range frames {
		switch *format {
		case "png":
			outPath := filepath.Join(*outDir, fmt.Sprintf("frame_%04d.png", i))
			if err := writeFramePNG(f, outPath); err != nil {
				fmt.Fprintf(os.Stderr, "write frame %d: %v\n", i, err)
			} else {
				fmt.Printf("  %s (%dx%d)\n", outPath, f.Width, f.Height)
			}
		case "yuv":
			outPath := filepath.Join(*outDir, fmt.Sprintf("frame_%04d.yuv", i))
			if err := writeFrameYUV(f, outPath); err != nil {
				fmt.Fprintf(os.Stderr, "write frame %d: %v\n", i, err)
			} else {
				fmt.Printf("  %s (%dx%d)\n", outPath, f.Width, f.Height)
			}
		}
	}
}

func writeFramePNG(f *decode.DecodedFrame, path string) error {
	img := image.NewGray(image.Rect(0, 0, f.Width, f.Height))
	for y := 0; y < f.Height; y++ {
		for x := 0; x < f.Width; x++ {
			img.Pix[y*img.Stride+x] = f.PixelY(x, y)
		}
	}
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	return png.Encode(out, img)
}

func writeFrameYUV(f *decode.DecodedFrame, path string) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	// Write Y plane
	for y := 0; y < f.Height; y++ {
		out.Write(f.Y[y*f.StrideY : y*f.StrideY+f.Width])
	}
	// Write U plane
	for y := 0; y < f.Height/2; y++ {
		out.Write(f.U[y*f.StrideC : y*f.StrideC+f.Width/2])
	}
	// Write V plane
	for y := 0; y < f.Height/2; y++ {
		out.Write(f.V[y*f.StrideC : y*f.StrideC+f.Width/2])
	}
	return nil
}
