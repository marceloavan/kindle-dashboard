package main

import (
	_ "embed"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//go:embed index.template.html
var indexTemplate string

func main() {
	out := flag.String("out", "site", "output directory")
	grayscale := flag.Bool("grayscale", false, "keep 16-level grayscale instead of 1-bit black/white")
	flag.Parse()

	if err := os.MkdirAll(*out, 0o755); err != nil {
		fatal(err)
	}

	img := render(buildCards(), *grayscale)

	pngPath := filepath.Join(*out, "dashboard.png")
	if err := writePNG(img, pngPath); err != nil {
		fatal(err)
	}

	version := time.Now().UTC().Format("20060102150405")
	if err := writeHTML(*out, version); err != nil {
		fatal(err)
	}

	fmt.Printf("wrote %s (%dx%d)\n", pngPath, img.Bounds().Dx(), img.Bounds().Dy())
	fmt.Printf("wrote %s (version %s)\n", filepath.Join(*out, "index.html"), version)
}

func writePNG(img image.Image, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func writeHTML(outDir, version string) error {
	html := strings.ReplaceAll(indexTemplate, "{{VERSION}}", version)
	return os.WriteFile(filepath.Join(outDir, "index.html"), []byte(html), 0o644)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
