package main

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/draw"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

type modernHDCommand1012Frames struct {
	Main    [4]*modernHDMapFrame
	Results []*modernHDMapFrame
	Hold    *modernHDMapFrame
}

// These producers build an unadorned typed input. Selection closures belong
// to the separately frozen steady snapshot, never the sampled terrain.
func freezeModernHDCommandGeometry(in indexedmap.NativeFrameInput) (indexedmap.NativeFrameInput, error) {
	if in.Frame.SelectionOverlay != nil {
		return in, errors.New("modern HD command10-12: unexpected sampled selection closure")
	}
	f := &in.Frame
	f.Cells = append([]fdicon.NativeTerrainCell(nil), f.Cells...)
	f.Units = append([]fdicon.NativeUnitLayerEntry(nil), f.Units...)
	f.ForegroundUnits = append([]fdicon.NativeForegroundLayerEntry(nil), f.ForegroundUnits...)
	f.Controls, f.LUT = append([]byte(nil), f.Controls...), append([]byte(nil), f.LUT...)
	if in.HUD.OptionalUnit != nil {
		unit := *in.HUD.OptionalUnit
		in.HUD.OptionalUnit = &unit
	}
	return in, nil
}

func (g *Game) prepareModernHDCommandSteady(in indexedmap.NativeFrameInput, work, vga []byte, palette color.Palette, set *modernStoryPortraitSet) (*modernHDMapFrame, error) {
	view := *g
	view.modernStoryPortraits = set
	frame, err := view.prepareModernHDMapFrameMode(in, work, palette, true)
	if err != nil {
		return nil, err
	}
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			if x < 4 || x >= 316 || y < 4 || y >= 196 {
				frame.Background.Set(x, y, palette[vga[y*320+x]])
			}
		}
	}
	frame.OverlaysAboveUI = true
	return frame, nil
}

func commandHDObjectFrame(normal *modernHDMapFrame, terrainVGA []byte, palette color.Palette) *modernHDMapFrame {
	frame := modernPlayer9EventFrame(normal, terrainVGA, palette)
	frame.UI = image.NewRGBA(image.Rect(0, 0, 320, 200))
	frame.Range, frame.Terrain = nil, nil
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			frame.Background.Set(x, y, palette[terrainVGA[y*320+x]])
		}
	}
	return frame
}

func commandHDFloor(value, divisor int) (int, int) {
	q, r := value/divisor, value%divisor
	if r < 0 {
		q--
		r += divisor
	}
	return q, r
}

// Native sampling is defined in 1/128-pixel coordinates. Multiplying its
// domain by six exposes HD subpixels without changing any native anchor.
func (g *Game) commandHDSampledTerrain(in indexedmap.NativeFrameInput, xOrigin, yOrigin, increment int, palette color.Palette, set *modernStoryPortraitSet) (*image.RGBA, error) {
	f := in.Frame
	if increment <= 0 || f.MapWidth <= 0 || len(f.Cells)%f.MapWidth != 0 {
		return nil, errors.New("modern HD sampled terrain: invalid geometry")
	}
	assets := set.terrainHD[g.nativeMapAssets.MapIndex]
	colours, err := terrainColourContext(g.nativeMapAssets.MapIndex, g.nativeMapAssets.Palette, palette, f.LUT)
	if err != nil {
		return nil, err
	}
	grid := make([]*image.RGBA, len(f.Cells))
	for i, cell := range f.Cells {
		tile := int(cell.Tile & 0x3ff)
		if tile*4 >= len(f.Controls) {
			return nil, errors.New("modern HD sampled terrain: missing control")
		}
		frame, err := fdicon.NativeTerrainFrameIndex(tile, f.Controls[tile*4], f.Flip, f.TerrainCycle)
		if err != nil {
			return nil, err
		}
		if a := assets[frame]; a.supports("terrain") {
			// 0x1F558 reads raw terrain pixels, without the ordinary LUT.
			grid[i] = a.shade(colours, true)
		}
	}
	startX := f.CameraX*3072 + 13*1536 + xOrigin - 156*increment
	startY := f.CameraY*3072 + 8*1536 + yOrigin - 96*increment
	picture := image.NewRGBA(image.Rect(0, 0, 312*6, 192*6))
	for y := 0; y < picture.Rect.Dy(); y++ {
		ty, ry := commandHDFloor(startY*6+y*increment, 3072*6)
		if ty < 0 || ty >= len(f.Cells)/f.MapWidth {
			continue
		}
		for x := 0; x < picture.Rect.Dx(); x++ {
			tx, rx := commandHDFloor(startX*6+x*increment, 3072*6)
			if tx < 0 || tx >= f.MapWidth {
				continue
			}
			if tile := grid[ty*f.MapWidth+tx]; tile != nil {
				picture.SetRGBA(x, y, tile.RGBAAt(rx/128, ry/128))
			}
		}
	}
	return picture, nil
}

func (g *Game) prepareModernHDCommand1012(j *nativeCommand1012Job, set *modernStoryPortraitSet) (*modernHDCommand1012Frames, error) {
	if j == nil || set == nil {
		return nil, errors.New("modern HD command10-12: owner/theme unavailable")
	}
	if len(set.mapSpritesHD) != 125 {
		if set.mapEffects != nil {
			return nil, errors.New("modern HD command10-12: incomplete HD world")
		}
		return nil, nil
	}
	if j.hdInputErr != nil {
		return nil, j.hdInputErr
	}
	if set.mapOverlays == nil {
		return nil, errors.New("modern HD command10-12: result bank unavailable")
	}
	for _, index := range modernMapOverlayHDIndices(5, 13) {
		entries := g.nativeMapAssets.CommandHealDigits
		if index >= len(entries) || set.mapOverlays.native[index] != modernLMIHDDescription(index, entries[index]) || set.mapOverlays.frames[index] == nil {
			return nil, errors.New("modern HD command10-12: incomplete result bank")
		}
		if set.mapOverlays.frames[index].Bounds() != image.Rect(0, 0, entries[index].Width*6, entries[index].Height*6) {
			return nil, errors.New("modern HD command10-12: result bank geometry changed")
		}
	}
	steady, err := g.prepareModernHDCommandSteady(j.hdBaseline, j.baselineWork, j.baselineVGA, j.palette, set)
	if err != nil {
		return nil, err
	}
	objects, err := g.prepareModernHDCommandSteady(j.hdInput, j.baselineWork, j.baselineVGA, j.palette, set)
	if err != nil {
		return nil, err
	}
	frames := &modernHDCommand1012Frames{Hold: steady}
	frames.Main[0] = steady
	transition := nativeTransitionInput(j.hdInput)
	transition.Units, transition.ForegroundUnits = nil, nil
	for i := 0; i < 3; i++ {
		work, vga := append([]byte(nil), j.baselineWork...), append([]byte(nil), j.baselineVGA...)
		if err := indexedmap.ComposeNativeCommand10To12Surface(work, vga, transition, j.schedule.XOrigins[i], j.schedule.YOrigins[i], j.schedule.SamplingIncrements[i]); err != nil {
			return nil, err
		}
		frame := commandHDObjectFrame(objects, vga, j.palette)
		plane, err := g.commandHDSampledTerrain(j.hdInput, j.schedule.XOrigins[i], j.schedule.YOrigins[i], j.schedule.SamplingIncrements[i], j.palette, set)
		if err != nil {
			return nil, err
		}
		frame.Terrain = []modernHDTerrainDraw{{Image: plane, Rect: plane.Bounds()}}
		frames.Main[i+1] = frame
	}
	base := modernPlayer9EventFrame(steady, j.baselineVGA, j.palette)
	for _, sources := range j.resultSources {
		frame := *base
		for _, source := range sources {
			entry := source.Entry
			if source.Resource != 5 || entry < 0 || set.mapOverlays.frames[entry] == nil || entry >= len(g.nativeMapAssets.CommandHealDigits) || set.mapOverlays.native[entry] != modernLMIHDDescription(entry, g.nativeMapAssets.CommandHealDigits[entry]) {
				return nil, errors.New("modern HD command10-12: result source changed")
			}
			picture := set.mapOverlays.frames[entry]
			frame.Overlays = append(frame.Overlays, modernHDMapOverlayDraw{Entry: entry, Image: picture, Rect: picture.Bounds().Add(image.Pt(source.X*6, source.Y*6))})
		}
		frames.Results = append(frames.Results, &frame)
	}
	if len(frames.Results) != 22 {
		return nil, errors.New("modern HD command10-12: incomplete result producer")
	}
	return frames, nil
}

func (g *Game) prepareModernHDCommandPrelude(j *native2189AJob, set *modernStoryPortraitSet) ([]*modernHDMapFrame, error) {
	if j != nil && j.hdInputErr != nil {
		return nil, j.hdInputErr
	}
	if j == nil || len(j.hdRadial) != 10 || len(j.hdLUTs) != 10 || len(j.vgaFrames) != 11 {
		return nil, errors.New("modern HD prelude: incomplete producer")
	}
	if set == nil || len(set.mapSpritesHD) != 125 {
		return nil, errors.New("modern HD prelude: incomplete HD world")
	}
	normal, err := g.prepareModernHDCommandSteady(j.hdInput, j.hdTerrain, j.vgaFrames[10], j.palette, set)
	if err != nil {
		return nil, err
	}
	result := make([]*modernHDMapFrame, 0, 11)
	assets := set.terrainHD[g.nativeMapAssets.MapIndex]
	for i, spec := range j.hdRadial {
		work, vga := append([]byte(nil), j.hdTerrain...), append([]byte(nil), j.vgaFrames[i]...)
		if err := fdother.ApplyRadialLUTRemap(work[0x8088:], 456, j.hdLUTs[i], spec); err != nil {
			return nil, err
		}
		if err := indexedmap.CopyNativeUnitPresentViewport(vga, work); err != nil {
			return nil, err
		}
		frame := commandHDObjectFrame(normal, vga, j.palette)
		mask := make([]byte, 456*192)
		probe := bytes.Repeat([]byte{1}, 256)
		if err := fdother.ApplyRadialLUTRemap(mask, 456, probe, spec); err != nil {
			return nil, err
		}
		plane := image.NewRGBA(image.Rect(0, 0, 312*6, 192*6))
		for _, terrain := range normal.Terrain {
			rect := terrain.Rect.Add(image.Pt(-24, -24))
			col, row := rect.Min.X/144, rect.Min.Y/144
			cell := j.hdInput.Frame.Cells[(j.hdInput.Frame.CameraY+row)*j.hdInput.Frame.MapWidth+j.hdInput.Frame.CameraX+col]
			lut := make([]byte, 256)
			for n := range lut {
				index := byte(n)
				if cell.BlitMode != 0xff {
					index = j.hdInput.Frame.LUT[n]
				}
				lut[n] = j.hdLUTs[i][index]
			}
			colours, err := terrainColourContext(g.nativeMapAssets.MapIndex, g.nativeMapAssets.Palette, j.palette, lut)
			if err != nil {
				return nil, err
			}
			mapped := assets[terrain.Frame].shade(colours, false)
			clip := rect.Intersect(plane.Rect)
			draw.Draw(plane, clip, terrain.Image, clip.Min.Sub(rect.Min), draw.Src)
			for y := clip.Min.Y; y < clip.Max.Y; y++ {
				for x := clip.Min.X; x < clip.Max.X; x++ {
					if mask[(y/6)*456+x/6] != 0 {
						plane.SetRGBA(x, y, mapped.RGBAAt(x-rect.Min.X, y-rect.Min.Y))
					}
				}
			}
		}
		frame.Terrain = []modernHDTerrainDraw{{Image: plane, Rect: plane.Bounds()}}
		result = append(result, frame)
	}
	return append(result, normal), nil
}
