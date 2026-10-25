package main

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
)

// Text uses the same logical coordinates as the game, with a separate raster
// surface at the selected HD resolution. Game state and line wrapping stay
// independent of display resolution.
type modernHDTextRun struct {
	font         *Font
	text         string
	x, y, pixels float64
	color        color.RGBA
	clip         image.Rectangle
	inkTop       bool
}

type modernHDTextFrame struct {
	scene *ebiten.Image
	runs  []modernHDTextRun
}

type modernHDIndexedText struct {
	runs    []modernHDTextRun
	palette color.Palette
	writes  *[]modernHDGlyphWrite
}

func modernHDNativeTextColor(palette color.Palette, index byte) color.RGBA {
	switch index {
	case 0xcd:
		return color.RGBA{240, 244, 255, 255}
	case 0x4c:
		return color.RGBA{20, 30, 60, 255}
	default:
		return nativePaletteColor(palette, index, color.RGBA{240, 244, 255, 255})
	}
}

func (g *Game) beginModernHDText(screen *ebiten.Image) func() {
	g.modernHDText = nil
	if g.modernStoryPortraits == nil || !g.modernStoryPortraits.hasHD || g.font == nil {
		return func() {}
	}
	g.modernHDText = &modernHDTextFrame{scene: screen}
	for _, f := range []*Font{g.font, g.fontNm} {
		if f != nil {
			f.hd = g.modernHDText
		}
	}
	return func() {
		for _, f := range []*Font{g.font, g.fontNm} {
			if f != nil {
				f.hd = nil
			}
		}
	}
}

func (p *modernHDPresentation) drawModernHDText(screen ebiten.FinalScreen, geometry ebiten.GeoM, brightness float64) {
	frame := p.Game.modernHDText
	if frame == nil || len(frame.runs) == 0 {
		return
	}
	if p.textSurface == nil {
		p.textSurface = ebiten.NewImage(modernHDWidth, modernHDHeight)
	}
	p.textSurface.Clear()
	const factor = float64(modernHDWidth) / logicalW
	for _, run := range frame.runs {
		renderFont := run.font
		if regional := run.font.hdFonts[p.Game.localeID]; regional != nil {
			renderFont = regional
		}
		face, asc := renderFont.faceFor(int(math.Round(run.pixels * factor)))
		if face == nil {
			continue
		}
		target := p.textSurface
		if !run.clip.Empty() {
			r := image.Rect(int(float64(run.clip.Min.X)*factor), int(float64(run.clip.Min.Y)*factor),
				int(float64(run.clip.Max.X)*factor), int(float64(run.clip.Max.Y)*factor)).Intersect(target.Bounds())
			if r.Empty() {
				continue
			}
			target = target.SubImage(r).(*ebiten.Image)
		}
		op := &ebiten.DrawImageOptions{}
		baseline := math.Round(run.y*factor) + asc
		if run.inkTop {
			baseline = math.Round(run.y*factor) - float64(text.BoundString(face, run.text).Min.Y)
		}
		op.GeoM.Translate(math.Round(run.x*factor), baseline)
		op.ColorScale.ScaleWithColor(run.color)
		text.DrawWithOptions(target, run.text, face, op)
	}
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(1/factor, 1/factor)
	op.GeoM.Concat(geometry)
	v := float32(math.Max(0, math.Min(1, brightness)))
	op.ColorScale.Scale(v, v, v, 1)
	screen.DrawImage(p.textSurface, op)
}

func (g *Game) clearModernHDText() {
	if g.modernHDText != nil {
		g.modernHDText.runs = nil
	}
}

func (g *Game) modernHDShopText() bool {
	return g != nil && g.font != nil && g.font.indexedHD != nil
}

func (g *Game) clearModernHDShopText() {
	if g.modernHDShopText() {
		runs := g.font.indexedHD.runs[:0]
		for _, run := range g.font.indexedHD.runs {
			if run.y < 220 {
				runs = append(runs, run)
			}
		}
		g.font.indexedHD.runs = runs
	}
}
