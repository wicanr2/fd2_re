package main

import (
	"errors"
	"image"
	"image/color"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

type modernHDMapForegroundDraw struct {
	Image       *image.RGBA
	Rect        image.Rectangle
	WorkOffset  int
	SourceFrame int
	BlitMode    byte
	HDImage     *image.RGBA
}

// Source masks, rather than pixel differences, preserve even same-color
// foreground writes. Native 0x4dd52 leaves destination-remap spans untouched.
func buildModernHDMapForegroundPlan(in indexedmap.FrameInput, viewportBase int, palette color.Palette) ([]modernHDMapForegroundDraw, error) {
	return buildModernHDMapForegroundRegionPlan(in, viewportBase, palette, 12, 8)
}

func buildModernHDMapForegroundRegionPlan(in indexedmap.FrameInput, viewportBase int, palette color.Palette, xmax, ymax int) ([]modernHDMapForegroundDraw, error) {
	stride := fdicon.NativeMapStride
	view := modernHDMapViewport()
	if in.ForegroundBank == nil || in.MapWidth <= 0 || len(in.Cells)%in.MapWidth != 0 ||
		(in.Flip != 0 && in.Flip != 1) || len(palette) != 256 || viewportBase < 0 ||
		viewportBase%stride+view.Dx()/6 > stride ||
		viewportBase+(view.Dy()/6-1)*stride+view.Dx()/6 > indexedmap.NativeUnitPresentWorkSize {
		return nil, errors.New("modern HD foreground: incomplete native input or viewport")
	}
	commands := make([]modernHDMapForegroundDraw, 0)
	for _, unit := range in.ForegroundUnits {
		if !fdicon.NativeForegroundRedrawEligible(unit.Inactive, unit.Unit7, unit.Race, unit.Class) {
			continue
		}
		coords, count := fdicon.NativeForegroundRedrawCells(unit.X, unit.Y, unit.Pose, unit.MotionOffset)
		for _, coord := range coords[:count] {
			if coord.X < in.CameraX-1 || coord.X > in.CameraX+xmax || coord.Y < in.CameraY-1 || coord.Y > in.CameraY+ymax ||
				coord.X < 0 || coord.Y < 0 || coord.X >= in.MapWidth || coord.Y >= len(in.Cells)/in.MapWidth {
				continue
			}
			cell := in.Cells[coord.Y*in.MapWidth+coord.X]
			tile := int(cell.Tile & 0x3ff)
			if tile*4 >= len(in.Controls) {
				return nil, errors.New("modern HD foreground: incomplete native terrain controls")
			}
			frame, present, err := fdicon.NativeForegroundFrameIndex(tile, in.Controls[tile*4], in.Flip)
			if err != nil {
				return nil, err
			}
			if !present {
				continue
			}
			sprite, err := in.ForegroundBank.SpriteFor(frame/12, (frame%12)/3, frame%3)
			if err != nil {
				return nil, err
			}
			if len(sprite.Pixels) != 576 || len(sprite.Mask) != 576 || (cell.BlitMode != 0xff && len(in.LUT) != 256) {
				return nil, errors.New("modern HD foreground: incomplete source or LUT")
			}
			offset := 0x8088 + (coord.Y-in.CameraY)*24*stride + (coord.X-in.CameraX)*24
			if offset < 0 || offset+23*stride+24 > indexedmap.NativeUnitPresentWorkSize {
				return nil, errors.New("modern HD foreground: native destination exceeds work buffer")
			}
			img := image.NewRGBA(image.Rect(0, 0, 24, 24))
			for index, mask := range sprite.Mask {
				if mask == 0 {
					continue
				}
				value := sprite.Pixels[index]
				if cell.BlitMode != 0xff {
					value = in.LUT[value]
				}
				if palette[value] == nil {
					return nil, errors.New("modern HD foreground: missing palette color")
				}
				red, green, blue, _ := palette[value].RGBA()
				img.SetRGBA(index%24, index/24, color.RGBA{uint8(red >> 8), uint8(green >> 8), uint8(blue >> 8), 255})
			}
			x, y := offset%stride-viewportBase%stride, offset/stride-viewportBase/stride
			rect := image.Rect(view.Min.X+x*6, view.Min.Y+y*6, view.Min.X+(x+24)*6, view.Min.Y+(y+24)*6)
			commands = append(commands, modernHDMapForegroundDraw{Image: img, Rect: rect, WorkOffset: offset,
				SourceFrame: frame, BlitMode: cell.BlitMode})
		}
	}
	return commands, nil
}
