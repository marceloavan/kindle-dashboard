package main

import (
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strings"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

//go:embed fonts/DejaVuSans.ttf
var fontRegular []byte

//go:embed fonts/DejaVuSans-Bold.ttf
var fontBold []byte

// Native screen resolution of the Kindle (8th gen, 2016).
const (
	width  = 600
	height = 800

	// Supersampling factor: draw at Nx, then downsample for smooth
	// anti-aliased edges before (optionally) dithering to 1-bit.
	scale = 2
)

// Layout constants, in native (600x800) pixels.
const (
	pad       = 30
	headerH   = 76
	titleSize = 32
	stampSize = 20
	cardTitle = 24
	bodySize  = 22
	bodyLead  = 4
	cardGap   = 10
)

func loadFace(bold bool, size float64) font.Face {
	data := fontRegular
	if bold {
		data = fontBold
	}
	f, err := opentype.Parse(data)
	if err != nil {
		panic(fmt.Sprintf("parse font: %v", err))
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		panic(fmt.Sprintf("new face: %v", err))
	}
	return face
}

func faceH(face font.Face) int {
	m := face.Metrics()
	return (m.Ascent + m.Descent).Ceil()
}

func fillRect(img *image.Gray, x0, y0, x1, y1 int, v uint8) {
	draw.Draw(img, image.Rect(x0, y0, x1, y1),
		image.NewUniform(color.Gray{Y: v}), image.Point{}, draw.Src)
}

// drawText draws a single line of anti-aliased text with its top at topY.
func drawText(img *image.Gray, face font.Face, x, topY int, s string, v uint8) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.Gray{Y: v}),
		Face: face,
		Dot:  fixed.P(x, topY+face.Metrics().Ascent.Ceil()),
	}
	d.DrawString(s)
}

func wrap(face font.Face, text string, maxW int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	cur := ""
	for _, w := range words {
		cand := w
		if cur != "" {
			cand = cur + " " + w
		}
		if int(font.MeasureString(face, cand)>>6) <= maxW {
			cur = cand
		} else {
			if cur != "" {
				lines = append(lines, cur)
			}
			cur = w
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}

// render draws the dashboard at 2x resolution, downsamples it to 600x800, and
// (unless grayscale) dithers it to 1-bit black/white.
func render(cards []Card, grayscale bool) image.Image {
	s := scale
	w := width * s
	h := height * s
	p := pad * s
	hdr := headerH * s
	gap := cardGap * s

	img := image.NewGray(image.Rect(0, 0, w, h))
	fillRect(img, 0, 0, w, h, 255) // white background

	titleFace := loadFace(true, float64(titleSize*s))
	stampFace := loadFace(true, float64(stampSize*s))
	cardFace := loadFace(true, float64(cardTitle*s))
	bodyFace := loadFace(false, float64(bodySize*s))

	// Header bar (black, with white text).
	fillRect(img, 0, 0, w, hdr, 0)
	drawText(img, titleFace, p, (hdr-faceH(titleFace))/2, "Kindle Dashboard", 255)
	stamp := ptStamp(currentTime())
	stampW := int(font.MeasureString(stampFace, stamp) >> 6)
	drawText(img, stampFace, w-p-stampW, (hdr-faceH(stampFace))/2, stamp, 255)

	y := hdr
	bodyLine := (bodySize + bodyLead) * s

	for _, card := range cards {
		// Card title.
		drawText(img, cardFace, p, y+12*s, strings.ToUpper(card.Title), 0)
		ty := y + 12*s + cardTitle*s + 8*s
		// Divider line under the title.
		fillRect(img, p, ty, w-p, ty+2*s, 0)
		ty += 12 * s

		maxW := w - 2*p
		for _, row := range card.Rows {
			if row.Label != "" {
				labelW := int(font.MeasureString(bodyFace, row.Label) >> 6)
				drawText(img, bodyFace, p, ty, row.Label, 0)
				valueX := p + labelW + 16*s
				for _, line := range wrap(bodyFace, row.Value, w-valueX-p) {
					drawText(img, bodyFace, valueX, ty, line, 0)
					ty += bodyLine
				}
			} else {
				for _, line := range wrap(bodyFace, row.Value, maxW) {
					drawText(img, bodyFace, p, ty, line, 0)
					ty += bodyLine
				}
			}
		}

		y = ty + gap
	}

	if y > h {
		fmt.Printf("warning: content overflows by ~%dpx and will be clipped\n", y-h)
	}

	// Downsample to native resolution.
	dst := image.NewGray(image.Rect(0, 0, width, height))
	xdraw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, img.Bounds(), xdraw.Src, nil)

	if grayscale {
		return dst
	}
	return dither(dst)
}

// dither converts an 8-bit grayscale image to a 1-bit palette image using
// Floyd–Steinberg error diffusion.
func dither(src *image.Gray) *image.Paletted {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()

	buf := make([]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			buf[y*w+x] = float64(src.GrayAt(x, y).Y)
		}
	}

	pal := color.Palette{color.Gray{Y: 0}, color.Gray{Y: 255}}
	dst := image.NewPaletted(image.Rect(0, 0, w, h), pal)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			old := buf[y*w+x]
			var nv uint8
			if old < 128 {
				nv = 0
			} else {
				nv = 255
			}
			err := old - float64(nv)
			buf[y*w+x] = float64(nv)

			if x+1 < w {
				buf[y*w+x+1] += err * 7.0 / 16.0
			}
			if y+1 < h {
				if x > 0 {
					buf[(y+1)*w+x-1] += err * 3.0 / 16.0
				}
				buf[(y+1)*w+x] += err * 5.0 / 16.0
				if x+1 < w {
					buf[(y+1)*w+x+1] += err * 1.0 / 16.0
				}
			}

			idx := uint8(0)
			if nv == 255 {
				idx = 1
			}
			dst.SetColorIndex(x, y, idx)
		}
	}
	return dst
}
