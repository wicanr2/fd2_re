package main

import (
	"errors"
	"image"
	"image/color"
	"math"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

// The camera below is a private rendering projection. The story controller
// alone publishes raw views at its original completion boundaries.
func (g *Game) prepareModernHDStoryCamera() error {
	if math.IsNaN(g.camX) || math.IsNaN(g.camY) || math.IsInf(g.camX, 0) || math.IsInf(g.camY, 0) ||
		g.camX < 0 || g.camY < 0 || g.camX > float64((g.m.W-13)*24) || g.camY > float64((g.m.H-8)*24) {
		return errors.New("modern HD story: camera outside drawable map")
	}
	g.storyNativeMapView.CameraX = int(math.Floor(g.camX / 24))
	g.storyNativeMapView.CameraY = int(math.Floor(g.camY / 24))
	return nil
}

func storyHDPixel(value float64) int { return int(math.Round(value * 6)) }

// Interpolation consumes the existing controller position, never its raw
// pending destination. Native stationary anchoring is retained at both ends.
func (view *Game) prepareModernHDStoryMotion(source *Game, frame *modernHDMapFrame, set *modernStoryPortraitSet) error {
	if view.nativeMapFrameSource == nil || len(frame.TerrainSeed) != indexedmap.NativeUnitPresentWorkSize {
		return errors.New("modern HD story: immutable terrain source absent")
	}
	in := view.nativeMapFrameSource.Frame
	if len(in.Units) != len(source.storyActors) || len(in.ForegroundUnits) != len(source.storyActors) {
		return errors.New("modern HD story: visual roster provenance mismatch")
	}
	palette, err := fdother.VGAPaletteFromDAC(view.nativeMapDAC)
	if err != nil {
		return err
	}
	dx, dy := source.camX-float64(in.CameraX*24), source.camY-float64(in.CameraY*24)
	shift := image.Pt(-storyHDPixel(dx), -storyHDPixel(dy))
	cols, rows := min(14, in.MapWidth-in.CameraX), min(9, len(in.Cells)/in.MapWidth-in.CameraY)
	if len(frame.TerrainUnderlay) != indexedmap.NativeUnitPresentWorkSize {
		return errors.New("modern HD story: terrain underlay absent")
	}
	work := append([]byte(nil), frame.TerrainUnderlay...)
	base := indexedmap.NativeUnitPresentStripLayoutFor(0, 0, 0, 0).WorkOffset
	if in.ChapterAux != nil {
		if err := fdother.BlitNativeChapterAuxViewport(work[base:], fdicon.NativeMapStride, in.ChapterAux, in.ChapterAuxPhase); err != nil {
			return err
		}
	}
	// Special chapter underlays are screen-local. Compensate their origin
	// before translating the terrain plane, without replaying any remap pass.
	underlay := append([]byte(nil), work...)
	for y := 0; y < rows*24; y++ {
		for x := 0; x < cols*24; x++ {
			sx, sy := max(0, min(311, int(math.Floor(float64(x)-dx)))), max(0, min(191, int(math.Floor(float64(y)-dy))))
			work[base+y*fdicon.NativeMapStride+x] = underlay[base+sy*fdicon.NativeMapStride+sx]
		}
	}
	// Overscan includes the incoming east/south cells. Only pixels inside the
	// final viewport are observable; hidden parallax samples are edge-clamped.
	if err := seedModernHDStoryParallax(work, base, cols, rows, in, dx, dy); err != nil {
		return err
	}
	if err := in.TerrainBank.BlitNativeTerrainRegion(work, fdicon.NativeMapStride, base%fdicon.NativeMapStride, base/fdicon.NativeMapStride,
		in.MapWidth, in.Cells, in.Controls, in.CameraX, in.CameraY, cols, rows, in.Flip, in.TerrainCycle, in.LUT); err != nil {
		return err
	}
	backdrop := image.NewRGBA(image.Rect(0, 0, cols*24, rows*24))
	for y := 0; y < backdrop.Bounds().Dy(); y++ {
		for x := 0; x < backdrop.Bounds().Dx(); x++ {
			backdrop.Set(x, y, palette[work[base+y*fdicon.NativeMapStride+x]])
		}
	}
	if in.Parallax != nil {
		backdrop, err = modernHDStoryParallaxBackdrop(in, cols, rows, dx, dy, palette)
		if err != nil {
			return err
		}
	}
	viewport := modernHDMapViewport()
	backdropRect := image.Rect(viewport.Min.X, viewport.Min.Y, viewport.Min.X+cols*144, viewport.Min.Y+rows*144).Add(shift)
	assets := set.terrainHD[view.nativeMapAssets.MapIndex]
	colours, err := terrainColourContext(view.nativeMapAssets.MapIndex, view.nativeMapAssets.Palette, palette, in.LUT)
	if err != nil {
		return err
	}
	terrain, err := buildModernHDTerrainRegionPlan(in, base, in.CameraY, cols, rows, base,
		viewport.Sub(shift), assets, colours)
	if err != nil {
		return err
	}
	for i := range terrain {
		terrain[i].Rect = terrain[i].Rect.Add(shift)
	}
	frame.Terrain = append([]modernHDTerrainDraw{{Image: backdrop, Rect: backdropRect, ScaleToRect: true}}, terrain...)
	frame.TransientTerrain = true
	frame.Units = nil
	in.ForegroundUnits = append([]fdicon.NativeForegroundLayerEntry(nil), in.ForegroundUnits...)
	counts := make(map[int]int, len(set.mapSpritesHD))
	for group, images := range set.mapSpritesHD {
		counts[group] = len(images)
	}
	walking := make(map[int]bool, len(source.storyWalks))
	for _, job := range source.storyWalks {
		if job == nil || job.actor < 0 || job.actor >= len(source.storyActors) {
			return errors.New("modern HD story: invalid walking actor")
		}
		walking[job.actor] = true
	}
	for index := range source.storyActors {
		u := &source.storyActors[index]
		fg := &in.ForegroundUnits[index]
		if !walking[index] {
			one := in
			one.Units = in.Units[index : index+1]
			units, err := buildModernHDMapUnitRegionPlan(one, base, counts, 13, 9)
			if err != nil {
				return err
			}
			for _, unit := range units {
				unit.Rect = unit.Rect.Add(shift)
				if unit.Rect.Overlaps(viewport) {
					frame.Units = append(frame.Units, unit)
				}
			}
			continue
		}
		fg.Inactive = !u.OnField || !u.Alive()
		if fg.Inactive {
			continue
		}
		px, py := float64(u.X*24)+u.OffX, float64(u.Y*24)+u.OffY
		if math.IsNaN(px) || math.IsNaN(py) || math.IsInf(px, 0) || math.IsInf(py, 0) || u.Dir < 0 || u.Dir > 3 {
			return errors.New("modern HD story: invalid visual placement")
		}
		fg.X, fg.Y, fg.Pose, fg.MotionOffset = int(math.Floor(px/24)), int(math.Floor(py/24)), byte(u.Dir), 0
		// Cover both adjacent cells while interpolating through a cell border.
		fg.MotionOffset = 1
		rect := image.Rect(0, 0, 144, 144).Add(image.Pt(viewport.Min.X+storyHDPixel(px-source.camX),
			viewport.Min.Y+storyHDPixel(py-source.camY-6)))
		if !rect.Overlaps(viewport) {
			continue
		}
		raw := in.Units[index]
		group, err := in.SelectorCache.KeyForSlot(raw.Slot)
		if err != nil {
			return err
		}
		cycle := (source.frame / 8) % 3
		sprite, err := in.UnitBank.SpriteForNativeSlot(in.SelectorCache, raw.Slot, u.Dir, cycle)
		if err != nil {
			return err
		}
		writes := false
		for _, value := range sprite.Mask {
			writes = writes || value != 0
		}
		if !writes {
			continue
		}
		selected := u.Dir*3 + cycle
		if len(set.mapSpritesHD[group]) == 3 {
			selected = cycle
		}
		frame.Units = append(frame.Units, modernHDMapUnitDraw{Group: group, Frame: selected, Rect: rect, PaletteBand: u.Acted})
	}
	foreground, err := buildModernHDMapForegroundRegionPlan(in, base, palette, 13, 9)
	if err != nil {
		return err
	}
	for i := range foreground {
		fg := &foreground[i]
		fg.Rect = fg.Rect.Add(shift)
		if asset := assets[fg.SourceFrame]; asset.supports("foreground") {
			fg.HDImage = asset.shade(colours, fg.BlitMode == 0xff)
		}
	}
	frame.Foreground = foreground
	frame.TerrainSeed = work
	return nil
}

func seedModernHDStoryParallax(work []byte, base, cols, rows int, in indexedmap.FrameInput, dx, dy float64) error {
	if in.Parallax == nil {
		return nil
	}
	if len(in.Parallax.Pixels) != fdother.NativeMapParallaxWidth*fdother.NativeMapParallaxHeight {
		return errors.New("modern HD story: incomplete parallax source")
	}
	for y := 0; y < rows*24; y++ {
		for x := 0; x < cols*24; x++ {
			sx := int(math.Floor(float64(3*in.CameraX+x+in.ParallaxScrollX/2) - dx + dx/8))
			sy := int(math.Floor(float64(2*in.CameraY+y+in.ParallaxScrollY/3) - dy + dy/12))
			sx = max(0, min(fdother.NativeMapParallaxWidth-1, sx))
			sy = max(0, min(fdother.NativeMapParallaxHeight-1, sy))
			work[base+y*fdicon.NativeMapStride+x] = in.Parallax.Pixels[sy*fdother.NativeMapParallaxWidth+sx]
		}
	}
	return nil
}

// Sample the parallax at output-pixel precision. Sampling it on the 24px
// terrain grid first would introduce a six-HD-pixel wobble whenever the
// compensated background crosses one native pixel. Native source-write and
// destination-remap masks remain distinct; remaps are applied exactly once.
func modernHDStoryParallaxBackdrop(in indexedmap.FrameInput, cols, rows int, dx, dy float64, palette color.Palette) (*image.RGBA, error) {
	if in.Parallax == nil || len(in.Parallax.Pixels) != fdother.NativeMapParallaxWidth*fdother.NativeMapParallaxHeight || len(palette) != 256 {
		return nil, errors.New("modern HD story: incomplete subpixel parallax input")
	}
	out := image.NewRGBA(image.Rect(0, 0, cols*144, rows*144))
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			cell := in.Cells[(in.CameraY+row)*in.MapWidth+in.CameraX+col]
			tile := int(cell.Tile & 0x3ff)
			frame, err := fdicon.NativeTerrainFrameIndex(tile, in.Controls[tile*4], in.Flip, in.TerrainCycle)
			if err != nil {
				return nil, err
			}
			sprite, err := in.TerrainBank.SpriteFor(frame/12, (frame%12)/3, frame%3)
			if err != nil {
				return nil, err
			}
			for y := 0; y < 144; y++ {
				py := row*144 + y
				sy := max(0, min(fdother.NativeMapParallaxHeight-1, int(math.Floor(float64(2*in.CameraY+in.ParallaxScrollY/3)+float64(py)/6-dy+dy/12))))
				for x := 0; x < 144; x++ {
					px := col*144 + x
					i := y/6*24 + x/6
					var value byte
					if sprite.Mask[i] != 0 {
						value = sprite.Pixels[i]
						if cell.BlitMode != 0xff {
							value = in.LUT[value]
						}
					} else {
						sx := max(0, min(fdother.NativeMapParallaxWidth-1, int(math.Floor(float64(3*in.CameraX+in.ParallaxScrollX/2)+float64(px)/6-dx+dx/8))))
						value = in.Parallax.Pixels[sy*fdother.NativeMapParallaxWidth+sx]
						if cell.BlitMode != 0xff && sprite.RemapMask[i] != 0 {
							value = in.LUT[value]
						}
					}
					out.Set(px, py, palette[value])
				}
			}
		}
	}
	return out, nil
}
