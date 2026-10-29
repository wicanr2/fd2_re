package main

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// This freezes FDOTHER#5's source validation and native panel colors.
// Status/item panels keep the thin native bevel; dialogue owners explicitly
// select the approved gold material in the same 18px border band.
type modernHDDialogueChrome struct {
	background color.RGBA
	edges      [4][3]color.RGBA // top, right, bottom, left
	corners    [4][9]color.RGBA // top-left, top-right, bottom-left, bottom-right
	gpu        *ebiten.Image
	gold       image.Image
	goldGPU    map[image.Point]*ebiten.Image
}

type modernHDDialoguePanel struct {
	chrome       *modernHDDialogueChrome
	rect         image.Rectangle // Frozen native 320x200 coordinates, before clipping.
	clip         image.Rectangle
	status       bool            // Status/item panels remain visible while the status owner is active.
	gold         bool            // Only dialogue owners use the approved antique-gold material.
	physicalRect image.Rectangle // Legacy story box geometry, snapped to HD pixels.
	values       *modernHDItemPanelValues
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
	tile, err := png.Decode(bytes.NewReader(modernHDGoldDialogueTile))
	if err != nil || tile.Bounds() != image.Rect(0, 0, 48, 48) {
		return nil, errors.New("modern HD dialogue: embedded gold material invalid")
	}
	result := &modernHDDialogueChrome{gold: tile}
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
			rect: image.Rect(5, 112+shift, 315, 198+shift), gold: true})
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
		rect := image.Rectangle{Min: panel.rect.Min.Mul(6), Max: panel.rect.Max.Mul(6)}
		if !panel.physicalRect.Empty() {
			rect = panel.physicalRect
		}
		visible := rect.Intersect(image.Rect(0, 0, 1920, 1200))
		if !panel.clip.Empty() {
			visible = visible.Intersect(image.Rectangle{Min: panel.clip.Min.Mul(6), Max: panel.clip.Max.Mul(6)})
		}
		if visible.Empty() || panel.chrome == nil {
			continue
		}
		chrome := panel.chrome
		texture := chrome.gpu
		if panel.gold {
			size := rect.Size()
			if chrome.goldGPU == nil {
				chrome.goldGPU = make(map[image.Point]*ebiten.Image)
			}
			texture = chrome.goldGPU[size]
			if texture == nil {
				texture = ebiten.NewImageFromImage(chrome.goldPatchFor(size.X, size.Y))
				chrome.goldGPU[size] = texture
			}
		} else if texture == nil {
			texture = ebiten.NewImageFromImage(chrome.patchFor(panel.rect.Dx()*6, panel.rect.Dy()*6))
			chrome.gpu = texture
		}
		local := visible.Sub(rect.Min)
		source := texture.SubImage(local).(*ebiten.Image)
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
		op.GeoM.Scale(1.0/3, 1.0/3)
		op.GeoM.Translate(float64(visible.Min.X)/3, float64(visible.Min.Y)/3)
		op.GeoM.Concat(geometry)
		v := float32(math.Max(0, math.Min(1, brightness)))
		op.ColorScale.Scale(v, v, v, 1)
		screen.DrawImage(source, op)
		if panel.values != nil {
			values := panel.values
			if values.gpu == nil {
				values.gpu = ebiten.NewImageFromImage(values.picture)
			}
			part := values.gpu.SubImage(local).(*ebiten.Image)
			screen.DrawImage(part, op)
		}
	}
}

// Code-generated from the approved SVG, not original game artwork. The 18px
// corners stay fixed; the flat center strips repeat without scaling the bevel.
//
//go:embed modern_hd_gold_dialogue.png
var modernHDGoldDialogueTile []byte

func (g *Game) prepareModernHDStoryDialogueChrome() error {
	assets := g.nativeClassUI
	if assets == nil {
		var err error
		assets, err = loadNativeClassUIAssets()
		if err != nil {
			return err
		}
	}
	chrome, err := freezeModernHDDialogueChrome(assets.dialogue, assets.palette)
	if err != nil {
		return err
	}
	g.nativeClassUI, g.nativeDialogueChrome = assets, chrome
	return nil
}

func (c *modernHDDialogueChrome) goldPatchFor(width, height int) *image.RGBA {
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	axis := func(at, size int) int {
		if at < 18 {
			return at
		}
		if at >= size-18 {
			return 48 - size + at
		}
		return 24
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if x >= 18 && x < width-18 && y >= 18 && y < height-18 {
				continue
			}
			canvas.Set(x, y, c.gold.At(axis(x, width), axis(y, height)))
		}
	}
	return canvas
}
