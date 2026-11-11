package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

type goldActionKey struct {
	index   int
	enabled bool
}
type goldStop struct {
	at     float64
	colour color.RGBA
}

func goldGradient(stops []goldStop, t float64) color.RGBA {
	if t <= stops[0].at {
		return stops[0].colour
	}
	for i := 1; i < len(stops); i++ {
		if t <= stops[i].at {
			a, b := stops[i-1], stops[i]
			f := (t - a.at) / (b.at - a.at)
			mix := func(x, y byte) byte { return byte(math.Round(float64(x) + (float64(y)-float64(x))*f)) }
			return color.RGBA{mix(a.colour.R, b.colour.R), mix(a.colour.G, b.colour.G), mix(a.colour.B, b.colour.B), 255}
		}
	}
	return stops[len(stops)-1].colour
}

func goldPolygon(dst *image.RGBA, rect image.Rectangle, cut int, colour func(int, int) color.RGBA) {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			left, right, top, bottom := x-rect.Min.X, rect.Max.X-1-x, y-rect.Min.Y, rect.Max.Y-1-y
			if left+top < cut || right+top < cut || left+bottom < cut || right+bottom < cut {
				continue
			}
			dst.SetRGBA(x, y, colour(x, y))
		}
	}
}

// The admitted A prototype uses the original pixels as a glyph mask, not a font.
// Only the base cell supplies strokes; native index variants retain selection.
func buildGoldActionFrame(cells []fdother.RawCell, palette color.Palette, index int, enabled bool) (*image.RGBA, error) {
	if index < 0 || index >= len(cells) || len(palette) != 256 {
		return nil, fmt.Errorf("gold command: original cell unavailable")
	}
	base := cells[index-index%3]
	if base.Width != 24 || (base.Height != 20 && base.Height != 16) || len(base.Pixels) != base.Width*base.Height {
		return nil, fmt.Errorf("gold command: original glyph geometry unavailable")
	}
	img := image.NewRGBA(image.Rect(0, 0, 144, 120))
	solid := func(c color.RGBA) func(int, int) color.RGBA { return func(int, int) color.RGBA { return c } }
	goldPolygon(img, image.Rect(2, 2, 142, 118), 6, solid(color.RGBA{16, 25, 21, 255}))
	rim := []goldStop{{0, color.RGBA{255, 240, 188, 255}}, {.22, color.RGBA{208, 166, 90, 255}}, {.55, color.RGBA{124, 86, 43, 255}}, {.77, color.RGBA{208, 166, 95, 255}}, {1, color.RGBA{64, 46, 28, 255}}}
	goldPolygon(img, image.Rect(4, 4, 140, 116), 5, func(x, y int) color.RGBA { return goldGradient(rim, (.7*float64(x)/144+float64(y)/120)/1.49) })
	goldPolygon(img, image.Rect(10, 10, 134, 110), 2, solid(color.RGBA{69, 46, 25, 255}))
	face := []goldStop{{0, color.RGBA{37, 60, 88, 255}}, {.55, color.RGBA{21, 43, 71, 255}}, {1, color.RGBA{10, 27, 49, 255}}}
	if index%3 == 1 && enabled {
		face = []goldStop{{0, color.RGBA{146, 44, 35, 255}}, {.55, color.RGBA{103, 33, 24, 255}}, {1, color.RGBA{61, 20, 20, 255}}}
	}
	goldPolygon(img, image.Rect(12, 12, 132, 108), 0, func(x, y int) color.RGBA { return goldGradient(face, (.2*float64(x)/144+float64(y)/120)/1.04) })
	fill := func(r image.Rectangle, c color.RGBA) {
		draw.Draw(img, r, &image.Uniform{color.NRGBA{R: c.R, G: c.G, B: c.B, A: c.A}}, image.Point{}, draw.Over)
	}
	fill(image.Rect(14, 13, 130, 14), color.RGBA{248, 226, 163, 115})
	fill(image.Rect(13, 15, 14, 106), color.RGBA{248, 226, 163, 115})
	fill(image.Rect(131, 14, 133, 108), color.RGBA{7, 14, 20, 255})
	fill(image.Rect(15, 107, 132, 109), color.RGBA{7, 14, 20, 255})
	letter := []goldStop{{0, color.RGBA{255, 246, 196, 255}}, {.35, color.RGBA{227, 185, 94, 255}}, {.67, color.RGBA{165, 107, 45, 255}}, {1, color.RGBA{242, 212, 135, 255}}}
	var mask []image.Rectangle
	glyphY := 60 - float64(base.Height-4)*4.7/2
	for y := 2; y < base.Height-2; y++ {
		for x := 2; x < 22; x++ {
			r, g, b, _ := palette[base.Pixels[y*24+x]].RGBA()
			if r>>8 > 200 && g>>8 > 100 && b>>8 < 120 {
				mask = append(mask, image.Rect(int(math.Round(23+float64(x-2)*4.9)), int(math.Round(glyphY+float64(y-2)*4.7)), int(math.Round(23+float64(x-1)*4.9)), int(math.Round(glyphY+float64(y-1)*4.7))))
			}
		}
	}
	if len(mask) == 0 {
		return nil, fmt.Errorf("gold command: original gold strokes unavailable for cell %d", index)
	}
	for _, r := range mask {
		fill(r.Add(image.Pt(1, 2)), color.RGBA{36, 23, 14, 255})
	}
	for _, r := range mask {
		goldPolygon(img, r, 0, func(x, y int) color.RGBA { return goldGradient(letter, (.15*float64(x)/144+float64(y)/120)/1.0225) })
	}
	corner := color.RGBA{214, 180, 107, 255}
	if index%3 == 1 && enabled {
		corner = color.RGBA{255, 242, 191, 255}
	}
	for _, r := range []image.Rectangle{image.Rect(6, 6, 15, 8), image.Rect(6, 6, 8, 15), image.Rect(129, 6, 138, 8), image.Rect(136, 6, 138, 15), image.Rect(136, 105, 138, 114), image.Rect(129, 112, 138, 114), image.Rect(6, 112, 15, 114), image.Rect(6, 105, 8, 114)} {
		fill(r, corner)
	}
	if !enabled || index%3 == 2 {
		fill(image.Rect(11, 12, 133, 109), color.RGBA{13, 27, 41, 161})
	}
	return img, nil
}

func (g *Game) goldActionFrame(index int, enabled bool) (*ebiten.Image, error) {
	key := goldActionKey{index, enabled}
	if img := g.goldActionFrames[key]; img != nil {
		return img, nil
	}
	frame, err := buildGoldActionFrame(g.nativeActionCellsRaw, g.nativeUIPalette, index, enabled)
	if err != nil {
		return nil, err
	}
	if g.goldActionFrames == nil {
		g.goldActionFrames = make(map[goldActionKey]*ebiten.Image)
	}
	img := ebiten.NewImageFromImage(frame)
	g.goldActionFrames[key] = img
	return img, nil
}

func (g *Game) resetGoldActionFrames() {
	for _, img := range g.goldActionFrames {
		img.Dispose()
	}
	g.goldActionFrames = nil
}
