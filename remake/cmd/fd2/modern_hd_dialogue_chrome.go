package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// This is a presentation-only snapshot of FDOTHER#5's three-color bevel.
// The native source keeps its three logical pixels; HD redraws three physical
// pixels and clears the old enlarged border without touching the interior.
type modernHDDialogueChrome struct {
	background color.RGBA
	edges      [4][3]color.RGBA // top, right, bottom, left
	corners    [4][9]color.RGBA // top-left, top-right, bottom-left, bottom-right
	gpu        *ebiten.Image
}

type modernHDDialoguePanel struct {
	chrome *modernHDDialogueChrome
	rect   image.Rectangle // Frozen native 320x200 coordinates, before clipping.
	clip   image.Rectangle
	status bool // Status/item panels remain visible while the status owner is active.
	values *modernHDItemPanelValues
}

func freezeModernHDDialogueChrome(cells []fdother.RawCell, palette color.Palette) (*modernHDDialogueChrome, error) {
	if len(cells) <= 13 || len(palette) != 256 {
		return nil, errors.New("modern HD dialogue: source unavailable")
	}
	for i, size := range map[int]image.Point{1: {3, 3}, 2: {3, 3}, 3: {3, 3}, 4: {3, 3}, 9: {16, 3}, 10: {3, 16}, 11: {3, 16}, 12: {16, 3}, 13: {16, 16}} {
		cell := cells[i]
		if cell.Width != size.X || cell.Height != size.Y || len(cell.Pixels) != size.X*size.Y {
			return nil, fmt.Errorf("modern HD dialogue: raw cell%d shape differs", i)
		}
	}
	bg := cells[13].Pixels[0]
	for _, pixel := range cells[13].Pixels {
		if pixel != bg {
			return nil, errors.New("modern HD dialogue: background is not uniform")
		}
	}
	rgba := func(index byte) (color.RGBA, error) {
		if palette[index] == nil {
			return color.RGBA{}, errors.New("modern HD dialogue: palette color unavailable")
		}
		r, g, b, a := palette[index].RGBA()
		if a != 65535 {
			return color.RGBA{}, errors.New("modern HD dialogue: source color is not opaque")
		}
		return color.RGBA{byte(r >> 8), byte(g >> 8), byte(b >> 8), 255}, nil
	}
	result := &modernHDDialogueChrome{}
	var err error
	result.background, err = rgba(bg)
	if err != nil {
		return nil, err
	}
	for side, source := range []int{9, 11, 12, 10} {
		cell := cells[source]
		for band := 0; band < 3; band++ {
			var index byte
			if cell.Width == 16 {
				index = cell.Pixels[band*16+8]
			} else {
				index = cell.Pixels[8*3+band]
			}
			for along := 0; along < 16; along++ {
				at := band*16 + along
				if cell.Width == 3 {
					at = along*3 + band
				}
				if cell.Pixels[at] != index {
					return nil, fmt.Errorf("modern HD dialogue: raw cell%d bevel is not uniform", source)
				}
			}
			result.edges[side][band], err = rgba(index)
			if err != nil {
				return nil, err
			}
		}
	}
	for corner := range result.corners {
		for i, index := range cells[corner+1].Pixels {
			result.corners[corner][i], err = rgba(index)
			if err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

func (c *modernHDDialogueChrome) patch() *image.RGBA {
	return c.patchFor(310*6, 86*6)
}

func (c *modernHDDialogueChrome) patchFor(width, height int) *image.RGBA {
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	fill := func(rect image.Rectangle, ink color.RGBA) {
		draw.Draw(canvas, rect, image.NewUniform(ink), image.Point{}, draw.Src)
	}
	for _, rect := range []image.Rectangle{image.Rect(0, 0, width, 18), image.Rect(0, height-18, width, height), image.Rect(0, 18, 18, height-18), image.Rect(width-18, 18, width, height-18)} {
		fill(rect, c.background)
	}
	for band := 0; band < 3; band++ {
		fill(image.Rect(3, band, width-3, band+1), c.edges[0][band])
		fill(image.Rect(width-3+band, 3, width-2+band, height-3), c.edges[1][band])
		fill(image.Rect(3, height-3+band, width-3, height-2+band), c.edges[2][band])
		fill(image.Rect(band, 3, band+1, height-3), c.edges[3][band])
	}
	for corner, origin := range []image.Point{{0, 0}, {width - 3, 0}, {0, height - 3}, {width - 3, height - 3}} {
		for i, ink := range c.corners[corner] {
			canvas.SetRGBA(origin.X+i%3, origin.Y+i/3, ink)
		}
	}
	return canvas
}

func (g *Game) appendModernHDDialogueChrome(chrome *modernHDDialogueChrome, shift int) {
	if chrome != nil && g.modernHDText != nil {
		g.modernHDText.panels = append(g.modernHDText.panels, modernHDDialoguePanel{chrome: chrome,
			rect: image.Rect(5, 112+shift, 315, 198+shift)})
	}
}

func (p *modernHDPresentation) drawModernHDDialogueChrome(screen ebiten.FinalScreen, geometry ebiten.GeoM, brightness float64) {
	frame := p.Game.modernHDText
	if frame == nil {
		return
	}
	for _, panel := range frame.panels {
		if !panel.status && p.Game.nativePlayerStatus != nil {
			continue
		}
		visible := panel.rect.Intersect(image.Rect(0, 0, 320, 200))
		if !panel.clip.Empty() {
			visible = visible.Intersect(panel.clip)
		}
		if visible.Empty() || panel.chrome == nil {
			continue
		}
		chrome := panel.chrome
		if chrome.gpu == nil {
			chrome.gpu = ebiten.NewImageFromImage(chrome.patchFor(panel.rect.Dx()*6, panel.rect.Dy()*6))
		}
		local := visible.Sub(panel.rect.Min)
		source := chrome.gpu.SubImage(image.Rect(local.Min.X*6, local.Min.Y*6, local.Max.X*6, local.Max.Y*6)).(*ebiten.Image)
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
		op.GeoM.Scale(1.0/3, 1.0/3)
		op.GeoM.Translate(float64(visible.Min.X*2), float64(visible.Min.Y*2))
		op.GeoM.Concat(geometry)
		v := float32(math.Max(0, math.Min(1, brightness)))
		op.ColorScale.Scale(v, v, v, 1)
		screen.DrawImage(source, op)
		if panel.values != nil {
			values := panel.values
			if values.gpu == nil {
				values.gpu = ebiten.NewImageFromImage(values.picture)
			}
			part := values.gpu.SubImage(image.Rect(local.Min.X*6, local.Min.Y*6, local.Max.X*6, local.Max.Y*6)).(*ebiten.Image)
			screen.DrawImage(part, op)
		}
	}
}
