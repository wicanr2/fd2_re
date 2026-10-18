package main

import (
	"errors"
	"image"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

type modernHDTerrainDraw struct {
	Image      *image.RGBA
	Rect       image.Rectangle
	Frame      int
	WorkOffset int
}

// Layout arguments are the actual native terrain and viewport work addresses,
// including the extra north row in NativeUpStepLayout. No second camera exists.
func buildModernHDTerrainPlan(in indexedmap.FrameInput, terrainBase, cameraY, rows, viewportBase int,
	assets map[int]*modernTerrainHDAsset, colours modernTerrainColourContext) ([]modernHDTerrainDraw, error) {
	if len(assets) == 0 {
		return nil, nil
	}
	if in.MapWidth <= 0 || len(in.Cells)%in.MapWidth != 0 || in.CameraX < 0 || cameraY < 0 ||
		in.CameraX+13 > in.MapWidth || cameraY+rows > len(in.Cells)/in.MapWidth || rows <= 0 ||
		terrainBase < 0 || viewportBase < 0 {
		return nil, errors.New("modern HD terrain: invalid native region")
	}
	view := modernHDMapViewport()
	result := make([]modernHDTerrainDraw, 0)
	for row := 0; row < rows; row++ {
		for col := 0; col < 13; col++ {
			cell := in.Cells[(cameraY+row)*in.MapWidth+in.CameraX+col]
			tile := int(cell.Tile & 0x3ff)
			if tile*4 >= len(in.Controls) {
				return nil, errors.New("modern HD terrain: control record missing")
			}
			frame, err := fdicon.NativeTerrainFrameIndex(tile, in.Controls[tile*4], in.Flip, in.TerrainCycle)
			if err != nil {
				return nil, err
			}
			asset := assets[frame]
			if !asset.supports("terrain") {
				continue
			}
			offset := terrainBase + row*24*fdicon.NativeMapStride + col*24
			if offset+23*fdicon.NativeMapStride+24 > indexedmap.NativeUnitPresentWorkSize {
				return nil, errors.New("modern HD terrain: work destination out of bounds")
			}
			x, y := offset%fdicon.NativeMapStride-viewportBase%fdicon.NativeMapStride,
				offset/fdicon.NativeMapStride-viewportBase/fdicon.NativeMapStride
			rect := image.Rect(view.Min.X+x*6, view.Min.Y+y*6, view.Min.X+(x+24)*6, view.Min.Y+(y+24)*6)
			if rect.Overlaps(view) {
				result = append(result, modernHDTerrainDraw{Image: asset.shade(colours, cell.BlitMode == 0xff),
					Rect: rect, Frame: frame, WorkOffset: offset})
			}
		}
	}
	return result, nil
}
