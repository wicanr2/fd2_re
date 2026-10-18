package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

// This candidate compositor has no Game.Draw entry. Its explicit seed belongs
// to terrain/parallax/staging, never a previous frame containing unit pixels.
// UI, walking copies and phase banners remain with their existing callers.
func composeModernHDMapWorld(in indexedmap.FrameInput, seed []byte, palette color.Palette, frames map[int][]image.Image) (*image.RGBA, error) {
	if len(seed) != indexedmap.NativeUnitPresentWorkSize || len(palette) != 256 {
		return nil, errors.New("modern HD world: incomplete seed or palette")
	}
	for _, entry := range palette {
		if entry == nil {
			return nil, errors.New("modern HD world: missing palette color")
		}
	}
	counts := make(map[int]int, len(frames))
	for group, images := range frames {
		counts[group] = len(images)
	}
	base := indexedmap.NativeUnitPresentStripLayoutFor(0, 0, 0, 0).WorkOffset
	units, err := buildModernHDMapUnitPlan(in, base, counts)
	if err != nil {
		return nil, err
	}
	for _, unit := range units {
		img := frames[unit.Group][unit.Frame]
		if img == nil || img.Bounds().Dx() != 144 || img.Bounds().Dy() != 144 {
			return nil, fmt.Errorf("modern HD world: invalid group %d frame %d", unit.Group, unit.Frame)
		}
	}
	// ComposeFrame's selector 6 changes the cell used by the later foreground
	// pass. Work on private cells, and build the foreground only after it.
	in.Cells = append([]fdicon.NativeTerrainCell(nil), in.Cells...)
	background := in
	background.Units, background.ForegroundUnits, background.SelectionOverlay = nil, nil, nil
	work, vga := append([]byte(nil), seed...), make([]byte, indexedmap.NativeMapVGASize)
	if err := indexedmap.ComposeFrame(work, vga, background, func([]byte) error { return nil }); err != nil {
		return nil, err
	}
	foreground, err := buildModernHDMapForegroundPlan(in, base, palette)
	if err != nil {
		return nil, err
	}
	world := image.NewRGBA(image.Rect(0, 0, modernHDWidth, modernHDHeight))
	draw.Draw(world, world.Bounds(), image.NewUniform(color.Black), image.Point{}, draw.Src)
	viewport := modernHDMapViewport()
	target := world.SubImage(viewport).(*image.RGBA)
	for y := indexedmap.NativeMapViewportY0; y <= indexedmap.NativeMapViewportY1; y++ {
		for x := indexedmap.NativeMapViewportX0; x <= indexedmap.NativeMapViewportX1; x++ {
			rect := image.Rect(x*6, y*6, (x+1)*6, (y+1)*6)
			draw.Draw(target, rect, image.NewUniform(palette[vga[y*320+x]]), image.Point{}, draw.Src)
		}
	}
	for _, unit := range units {
		source := frames[unit.Group][unit.Frame]
		if unit.PaletteBand {
			source = dimModernHDMapUnit(source)
		}
		draw.Draw(target, unit.Rect, source, source.Bounds().Min, draw.Over)
	}
	for _, command := range foreground {
		for y := 0; y < 24; y++ {
			for x := 0; x < 24; x++ {
				pixel := command.Image.RGBAAt(x, y)
				if pixel.A == 0 {
					continue
				}
				left, top := command.Rect.Min.X+x*6, command.Rect.Min.Y+y*6
				draw.Draw(target, image.Rect(left, top, left+6, top+6), image.NewUniform(pixel), image.Point{}, draw.Src)
			}
		}
	}
	return world, nil
}

func dimModernHDMapUnit(source image.Image) *image.RGBA {
	result := image.NewRGBA(source.Bounds())
	for y := source.Bounds().Min.Y; y < source.Bounds().Max.Y; y++ {
		for x := source.Bounds().Min.X; x < source.Bounds().Max.X; x++ {
			pixel := color.RGBAModel.Convert(source.At(x, y)).(color.RGBA)
			pixel.R, pixel.G, pixel.B = uint8(float64(pixel.R)*0.55), uint8(float64(pixel.G)*0.55), uint8(float64(pixel.B)*0.6)
			result.SetRGBA(x, y, pixel)
		}
	}
	return result
}
