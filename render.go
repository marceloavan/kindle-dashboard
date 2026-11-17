package main

import (
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"sort"
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
	pad        = 30
	headerH    = 76
	titleSize  = 32
	stampSize  = 20
	cardTitle  = 24
	bodySize   = 22
	bodyLead   = 4
	cardGap    = 12
	heroSize   = 56
	heroSub    = 20
	cardHero   = 30
	footerH    = 38
	footerSize = 18
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

func fillCircle(img *image.Gray, cx, cy, r int, v uint8) {
	for y := -r; y <= r; y++ {
		dx := int(math.Sqrt(float64(r*r - y*y)))
		fillRect(img, cx-dx, cy+y, cx+dx+1, cy+y+1, v)
	}
}

func fillPoly(img *image.Gray, pts [][2]int, v uint8) {
	if len(pts) < 3 {
		return
	}
	minY, maxY := pts[0][1], pts[0][1]
	for _, p := range pts {
		if p[1] < minY {
			minY = p[1]
		}
		if p[1] > maxY {
			maxY = p[1]
		}
	}
	n := len(pts)
	for y := minY; y <= maxY; y++ {
		var xs []int
		for i := 0; i < n; i++ {
			x1, y1 := pts[i][0], pts[i][1]
			x2, y2 := pts[(i+1)%n][0], pts[(i+1)%n][1]
			if (y1 <= y && y2 > y) || (y2 <= y && y1 > y) {
				xs = append(xs, x1+(y-y1)*(x2-x1)/(y2-y1))
			}
		}
		sort.Ints(xs)
		for i := 0; i+1 < len(xs); i += 2 {
			fillRect(img, xs[i], y, xs[i+1]+1, y+1, v)
		}
	}
}

func thickLine(img *image.Gray, x0, y0, x1, y1, w int, v uint8) {
	dx := x1 - x0
	if dx < 0 {
		dx = -dx
	}
	sx := 1
	if x0 > x1 {
		sx = -1
	}
	dy := y1 - y0
	if dy > 0 {
		dy = -dy
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	x, y := x0, y0
	for {
		fillRect(img, x-w/2, y-w/2, x-w/2+w, y-w/2+w, v)
		if x == x1 && y == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x += sx
		}
		if e2 <= dx {
			err += dx
			y += sy
		}
	}
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

// --- weather icons (stylized, black or white fills) ---

func drawCloud(img *image.Gray, cx, cy, size int, col uint8) {
	u := size / 48
	if u < 1 {
		u = 1
	}
	fillCircle(img, cx-14*u, cy+2*u, 12*u, col)
	fillCircle(img, cx, cy-6*u, 14*u, col)
	fillCircle(img, cx+14*u, cy+2*u, 12*u, col)
	fillRect(img, cx-26*u, cy-2*u, cx+26*u, cy+10*u, col)
}

func drawIcon(img *image.Gray, cx, cy, size int, name string, col uint8) {
	u := size / 48
	if u < 1 {
		u = 1
	}
	switch name {
	case "clear": // sun with rays
		fillCircle(img, cx, cy, 14*u, col)
		for i := 0; i < 8; i++ {
			a := float64(i) * math.Pi / 4
			x1 := cx + int(math.Cos(a)*float64(19*u))
			y1 := cy + int(math.Sin(a)*float64(19*u))
			x2 := cx + int(math.Cos(a)*float64(23*u))
			y2 := cy + int(math.Sin(a)*float64(23*u))
			thickLine(img, x1, y1, x2, y2, 2*u, col)
		}
	case "partly": // small sun peeking behind a cloud
		fillCircle(img, cx-11*u, cy-10*u, 9*u, col)
		drawCloud(img, cx+8*u, cy+6*u, size, col)
	case "cloudy":
		drawCloud(img, cx, cy+2*u, size, col)
	case "fog":
		drawCloud(img, cx, cy-8*u, size, col)
		thickLine(img, cx-20*u, cy+12*u, cx+20*u, cy+12*u, 2*u, col)
		thickLine(img, cx-14*u, cy+18*u, cx+14*u, cy+18*u, 2*u, col)
	case "drizzle":
		drawCloud(img, cx, cy-8*u, size, col)
		thickLine(img, cx-8*u, cy+12*u, cx-8*u, cy+19*u, 2*u, col)
		thickLine(img, cx+6*u, cy+12*u, cx+6*u, cy+19*u, 2*u, col)
	case "rain":
		drawCloud(img, cx, cy-10*u, size, col)
		thickLine(img, cx-12*u, cy+12*u, cx-17*u, cy+22*u, 2*u, col)
		thickLine(img, cx-2*u, cy+12*u, cx-7*u, cy+24*u, 2*u, col)
		thickLine(img, cx+8*u, cy+12*u, cx+3*u, cy+26*u, 2*u, col)
	case "showers":
		drawCloud(img, cx, cy-10*u, size, col)
		thickLine(img, cx-14*u, cy+12*u, cx-20*u, cy+23*u, 2*u, col)
		thickLine(img, cx-4*u, cy+12*u, cx-10*u, cy+26*u, 2*u, col)
		thickLine(img, cx+6*u, cy+12*u, cx+0*u, cy+28*u, 2*u, col)
		thickLine(img, cx+16*u, cy+12*u, cx+10*u, cy+26*u, 2*u, col)
	case "storm":
		drawCloud(img, cx, cy-8*u, size, col)
		fillPoly(img, [][2]int{
			{cx - 5*u, cy + 10*u}, {cx + 7*u, cy + 10*u},
			{cx + 1*u, cy + 22*u}, {cx + 9*u, cy + 22*u},
			{cx - 7*u, cy + 36*u}, {cx - 3*u, cy + 22*u},
		}, col)
	case "snow":
		drawCloud(img, cx, cy-8*u, size, col)
		for _, dx := range []int{-10, 0, 10} {
			thickLine(img, cx+dx*u-3*u, cy+17*u, cx+dx*u+3*u, cy+17*u, 2*u, col)
			thickLine(img, cx+dx*u, cy+14*u, cx+dx*u, cy+20*u, 2*u, col)
		}
	default:
		drawCloud(img, cx, cy, size, col)
	}
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
	heroFace := loadFace(true, float64(heroSize*s))
	subFace := loadFace(false, float64(heroSub*s))
	cardHeroFace := loadFace(true, float64(cardHero*s))
	footerFace := loadFace(false, float64(footerSize*s))

	// Header bar (black, with white text).
	fillRect(img, 0, 0, w, hdr, 0)
	stamp := ptStamp(currentTime())
	stampW := int(font.MeasureString(stampFace, stamp) >> 6)
	if t := dashboardTitle(); t != "" {
		drawText(img, titleFace, p, (hdr-faceH(titleFace))/2, t, 255)
		drawText(img, stampFace, w-p-stampW, (hdr-faceH(stampFace))/2, stamp, 255)
	} else {
		// No title yet: center the timestamp in the bar.
		drawText(img, stampFace, (w-stampW)/2, (hdr-faceH(stampFace))/2, stamp, 255)
	}

	y := hdr
	bodyLine := (bodySize + bodyLead) * s

	for _, card := range cards {
		if card.Invert {
			// Inverted hero card (white on black).
			ch := (16+heroSize+2+heroSub+8+14)*s + len(card.Rows)*bodyLine
			fillRect(img, 0, y, w, y+ch, 0)
			drawText(img, heroFace, p, y+16*s, card.Hero, 255)
			drawText(img, subFace, p, y+16*s+heroSize*s+2*s, card.Sub, 255)
			if card.Icon != "" {
				is := 48 * s
				drawIcon(img, w-p-is/2, y+16*s+is/2, is, card.Icon, 255)
			}
			ty := y + (16+heroSize+2+heroSub+8)*s
			for _, row := range card.Rows {
				drawText(img, bodyFace, p, ty, row.Label, 255)
				if row.Label != "" {
					lw := int(font.MeasureString(bodyFace, row.Label) >> 6)
					drawText(img, bodyFace, p+lw+16*s, ty, row.Value, 255)
				} else {
					drawText(img, bodyFace, p, ty, row.Value, 255)
				}
				ty += bodyLine
			}
			y += ch + gap
			continue
		}

		// Normal card: title with a rule to the right edge.
		title := strings.ToUpper(card.Title)
		drawText(img, cardFace, p, y+12*s, title, 0)
		tw := int(font.MeasureString(cardFace, title) >> 6)
		cyc := y + 12*s + faceH(cardFace)/2
		fillRect(img, p+tw+12*s, cyc-1*s, w-p, cyc+1*s, 0)
		ty := y + (12+cardTitle+8)*s

		// Icon + big value block.
		if card.Icon != "" {
			is := 48 * s
			drawIcon(img, p+is/2, ty+is/2, is, card.Icon, 0)
			hx := p + is + 16*s
			drawText(img, cardHeroFace, hx, ty, card.Hero, 0)
			if card.Sub != "" {
				drawText(img, subFace, hx, ty+faceH(cardHeroFace)+2*s, card.Sub, 0)
			}
			ty += faceH(cardHeroFace) + 2*s + faceH(subFace) + 6*s
		}

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

	if y+footerH*s > h {
		fmt.Printf("warning: content overflows by ~%dpx and will be clipped\n", (y+footerH*s-h)/s)
	}

	// Footer: divider + update time.
	fy := h - footerH*s
	fillRect(img, 0, fy-1*s, w, fy+1*s, 0)
	ftxt := "Atualizado às " + currentTime().Format("15:04")
	fw := int(font.MeasureString(footerFace, ftxt) >> 6)
	drawText(img, footerFace, (w-fw)/2, fy+(footerH*s-faceH(footerFace))/2, ftxt, 0)

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
