package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

const unitPresentHDBase = 0x8088

type modernHDUnitPresentSource struct {
	Call                          fdother.NativeUnitPresentCall
	Cursor                        image.Point
	MapIndex                      int
	Before, After                 indexedmap.NativeFrameInput
	Terrain, Snapshot, InitialVGA []byte
	Entries                       []fdother.LMI1Entry
	Passes                        []fdother.UnitPresentLUTFrame
	LUTs                          [][]byte
	BridgeLUT                     []byte
}

func (g *Game) unitPresentHDInput(in indexedmap.NativeTransitionFrameInput) (indexedmap.NativeFrameInput, error) {
	f := indexedmap.FrameInput{TerrainBank: in.TerrainBank, RangeBank: g.nativeMapAssets.Range,
		UnitBank: in.UnitBank, ForegroundBank: in.ForegroundBank, SelectorCache: in.SelectorCache.Clone(),
		Cells: in.Cells, Controls: in.Controls, LUT: in.TerrainLUT, MapWidth: in.MapWidth,
		CameraX: in.CameraX, CameraY: in.CameraY, Flip: in.Flip, TerrainCycle: in.TerrainCycle,
		IdleCycle: in.IdleCycle, MovingCycle: in.MovingCycle, PixelShift: in.PixelShift,
		Units: in.Units, ForegroundUnits: in.ForegroundUnits, Parallax: in.Parallax,
		ParallaxScrollX: in.ParallaxScrollX, ParallaxScrollY: in.ParallaxScrollY}
	return freezeModernHDCommandGeometry(indexedmap.NativeFrameInput{Frame: f})
}

func (g *Game) freezeModernHDUnitPresentSource(call fdother.NativeUnitPresentCall, view battle.NativeMapViewState,
	before, after indexedmap.NativeTransitionFrameInput, terrain, snapshot, initialVGA []byte,
	passes []fdother.UnitPresentLUTFrame, bridgeLUT []byte) (*modernHDUnitPresentSource, error) {
	a, err := g.unitPresentHDInput(before)
	if err != nil {
		return nil, err
	}
	b, err := g.unitPresentHDInput(after)
	if err != nil {
		return nil, err
	}
	s := &modernHDUnitPresentSource{Call: call, Cursor: image.Pt(view.VisibleCursorX, view.VisibleCursorY), MapIndex: g.nativeMapAssets.MapIndex,
		Before: a, After: b, Terrain: append([]byte(nil), terrain...), Snapshot: append([]byte(nil), snapshot...), InitialVGA: append([]byte(nil), initialVGA...),
		Passes: append([]fdother.UnitPresentLUTFrame(nil), passes...), BridgeLUT: append([]byte(nil), bridgeLUT...)}
	for index := 114; index <= 124; index++ {
		e := g.nativeMapAssets.FDOTHER6[index]
		e.Pixels = append([]byte(nil), e.Pixels...)
		s.Entries = append(s.Entries, e)
	}
	for _, p := range passes {
		s.LUTs = append(s.LUTs, append([]byte(nil), g.nativeMapAssets.LUTs[p.LUTIndex]...))
	}
	return s, nil
}

// The typed identity excludes immutable source banks and includes every selected
// cache key. Native byte reconstruction independently verifies bank consumers.
func (s *modernHDUnitPresentSource) signature(palette color.Palette) ([32]byte, error) {
	if s == nil {
		return [32]byte{}, errors.New("modern HD unit present: source absent")
	}
	identity := func(in indexedmap.NativeFrameInput) any {
		f := in.Frame
		keys := make([]int, len(f.Units))
		if f.SelectorCache == nil {
			return nil
		}
		for i, u := range f.Units {
			key, err := f.SelectorCache.KeyForSlot(u.Slot)
			if err != nil {
				key = -1
			}
			keys[i] = key
		}
		return []any{f.Cells, f.Controls, f.LUT, f.Units, f.ForegroundUnits, keys, f.MapWidth, f.CameraX, f.CameraY, f.Flip, f.TerrainCycle, f.IdleCycle, f.MovingCycle, f.PixelShift, f.ParallaxScrollX, f.ParallaxScrollY}
	}
	raw, err := json.Marshal([]any{s.Call, s.Cursor, s.MapIndex, identity(s.Before), identity(s.After), s.Entries, s.Passes, s.LUTs, s.BridgeLUT, palette})
	if err != nil {
		return [32]byte{}, err
	}
	h := sha256.New()
	h.Write(raw)
	h.Write(s.Terrain)
	h.Write(s.Snapshot)
	h.Write(s.InitialVGA)
	var result [32]byte
	copy(result[:], h.Sum(nil))
	return result, nil
}

func (g *Game) validateModernHDUnitPresentSource(j *nativeUnitPresentJob) error {
	s := j.hdSource
	if s == nil || s.MapIndex != g.nativeMapAssets.MapIndex || len(s.Entries) != 11 || len(s.Passes) != 16 || len(s.LUTs) != 16 || len(s.BridgeLUT) != 256 || len(s.InitialVGA) != indexedmap.NativeMapVGASize || len(j.palette) != 256 {
		return errors.New("modern HD unit present: incomplete frozen producer")
	}
	digest, err := s.signature(j.palette)
	if err != nil || digest != j.hdSignature {
		return errors.New("modern HD unit present: frozen source changed")
	}
	if s.Call.UnitSlot != j.unitSlot || int(s.Call.NewX) != j.newX || int(s.Call.NewY) != j.newY || j.mutationAt != 17 {
		return errors.New("modern HD unit present: mutation boundary changed")
	}
	passes, err := fdother.NativeUnitPresentLUTFrames(s.Cursor.X, s.Cursor.Y)
	if err != nil {
		return err
	}
	for i, p := range passes {
		if p != s.Passes[i] || len(s.LUTs[i]) != 256 {
			return errors.New("modern HD unit present: LUT schedule changed")
		}
	}
	before, after := nativeTransitionInput(s.Before), nativeTransitionInput(s.After)
	terrain := make([]byte, indexedmap.NativeUnitPresentWorkSize)
	if err := indexedmap.ComposeNativeUnitPresentTerrainSnapshot(terrain, before); err != nil {
		return err
	}
	if !bytes.Equal(terrain, s.Terrain) {
		return errors.New("modern HD unit present: terrain snapshot changed")
	}
	snapshot := make([]byte, len(terrain))
	if err := indexedmap.ComposeNativeUnitPresentLUTSnapshot(snapshot, terrain, before, s.Entries[10], int(s.Call.VisualX), int(s.Call.VisualY)); err != nil {
		return err
	}
	if !bytes.Equal(snapshot, s.Snapshot) {
		return errors.New("modern HD unit present: LMI snapshot changed")
	}
	layout := indexedmap.NativeUnitPresentStripLayoutFor(int(s.Call.VisualX), int(s.Call.VisualY), before.CameraX, before.CameraY)
	count := 27 + layout.Rows
	if len(j.workFrames) != count || len(j.vgaFrames) != count || len(j.waits) != count {
		return errors.New("modern HD unit present: native frame count changed")
	}
	index := 0
	check := func(work, vga []byte, wait int) error {
		if index >= count || !bytes.Equal(work, j.workFrames[index]) || !bytes.Equal(vga, j.vgaFrames[index]) || j.waits[index] != wait {
			return fmt.Errorf("modern HD unit present: native frame %d changed", index)
		}
		index++
		return nil
	}
	work := make([]byte, len(terrain))
	vga := append([]byte(nil), s.InitialVGA...)
	for _, entry := range s.Entries {
		if err := indexedmap.ComposeNativeUnitPresentIntroFrame(work, vga, terrain, before, entry, int(s.Call.VisualX), int(s.Call.VisualY)); err != nil {
			return err
		}
		if err := check(work, vga, 3); err != nil {
			return err
		}
	}
	for i, p := range passes[:6] {
		if err := indexedmap.ComposeNativeUnitPresentLUTFrame(work, vga, snapshot, s.LUTs[i], before, p); err != nil {
			return err
		}
		if err := check(work, vga, nativeDelayTicks(p.DelayMs)+p.DelayTicks*3); err != nil {
			return err
		}
	}
	if err := indexedmap.ComposeNativeUnitPresentStripBridge(work, vga, snapshot, s.BridgeLUT, after, int(s.Call.VisualX), int(s.Call.VisualY), s.Cursor.X, s.Cursor.Y,
		func(int) error { return check(work, vga, nativeDelayTicks(10)) }); err != nil {
		return err
	}
	for i, p := range passes[6:] {
		if err := indexedmap.ComposeNativeUnitPresentLUTFrame(work, vga, snapshot, s.LUTs[i+6], after, p); err != nil {
			return err
		}
		if err := check(work, vga, p.DelayTicks*3); err != nil {
			return err
		}
	}
	return nil
}

func unitPresentHDBank(j *nativeUnitPresentJob, set *modernStoryPortraitSet) (bool, error) {
	if set == nil {
		return false, nil
	}
	has := hasHealHDBank(set.mapEffects, 114, 124)
	if !has {
		return false, nil
	}
	if len(j.hdSource.Entries) != 11 {
		return false, errors.New("modern HD unit present: incomplete native LMI source")
	}
	for n := 114; n <= 124; n++ {
		entry := j.hdSource.Entries[n-114]
		if set.mapEffects.frames[n] == nil || set.mapEffects.native[n] != modernLMIHDDescription(n, entry) || set.mapEffects.frames[n].Bounds() != image.Rect(0, 0, entry.Width*6, entry.Height*6) {
			return false, errors.New("modern HD unit present: partial or changed declared LMI group")
		}
	}
	return true, nil
}

func unitPresentHDGround(s *modernHDUnitPresentSource, set *modernStoryPortraitSet, palette color.Palette, index int, has bool) modernHDRadialGroundLayer {
	entry := s.Entries[index-114]
	origin := fdother.NativeUnitPresentByteOrigin(int(s.Call.VisualX), int(s.Call.VisualY), s.Before.Frame.CameraX, s.Before.Frame.CameraY)
	x, y := origin%456-unitPresentHDBase%456, origin/456-unitPresentHDBase/456
	layer := modernHDRadialGroundLayer{Rect: image.Rect(x*6, y*6, (x+entry.Width)*6, (y+entry.Height)*6)}
	if has {
		layer.Image = set.mapEffects.frames[index]
		return layer
	}
	layer.Native = &entry
	layer.Image = image.NewRGBA(image.Rect(0, 0, entry.Width*6, entry.Height*6))
	for yy := 0; yy < entry.Height*6; yy++ {
		for xx := 0; xx < entry.Width*6; xx++ {
			v := entry.Pixels[yy/6*entry.Width+xx/6]
			if v != 0 {
				layer.Image.Set(xx, yy, palette[v])
			}
		}
	}
	return layer
}

func unitPresentHDBorder(frame *modernHDMapFrame, vga []byte, palette color.Palette) *modernHDMapFrame {
	f := *frame
	f.Background = image.NewRGBA(frame.Background.Bounds())
	copy(f.Background.Pix, frame.Background.Pix)
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			if x >= 312 || y >= 192 {
				f.Background.Set(x, y, palette[vga[y*320+x]])
			}
		}
	}
	return &f
}

func (g *Game) prepareModernHDUnitPresent(j *nativeUnitPresentJob, set *modernStoryPortraitSet) ([]*modernHDMapFrame, error) {
	if set == nil {
		return nil, nil
	}
	if j == nil || j.hdSource == nil {
		return nil, errors.New("modern HD unit present: owner absent")
	}
	has, err := unitPresentHDBank(j, set)
	if err != nil {
		return nil, err
	}
	if len(set.mapSpritesHD) != 125 {
		if has {
			return nil, errors.New("modern HD unit present: declared effects without HD world")
		}
		return nil, nil
	}
	if err := g.validateModernHDUnitPresentSource(j); err != nil {
		return nil, err
	}
	counts := make(map[int]int, len(set.mapSpritesHD))
	for group, frames := range set.mapSpritesHD {
		counts[group] = len(frames)
	}
	for _, input := range []indexedmap.NativeFrameInput{j.hdSource.Before, j.hdSource.After} {
		plans, err := buildModernHDMapUnitPlan(input.Frame, unitPresentHDBase, counts)
		if err != nil {
			return nil, err
		}
		for _, plan := range plans {
			source := set.mapSpritesHD[plan.Group][plan.Frame]
			if source == nil || source.Bounds() != image.Rect(0, 0, 144, 144) {
				return nil, errors.New("modern HD unit present: selected HD world unit absent or wrong size")
			}
		}
	}
	if j.hdCachedTheme == set && len(j.hdCached) == len(j.vgaFrames) {
		return j.hdCached, nil
	}
	s := j.hdSource
	before, err := g.prepareModernHDCommandSteady(s.Before, s.Terrain, s.InitialVGA, j.palette, set)
	if err != nil {
		return nil, err
	}
	after, err := g.prepareModernHDCommandSteady(s.After, s.Terrain, s.InitialVGA, j.palette, set)
	if err != nil {
		return nil, err
	}
	terrainVGA := append([]byte(nil), s.InitialVGA...)
	if err := indexedmap.CopyNativeUnitPresentViewport(terrainVGA, s.Terrain); err != nil {
		return nil, err
	}
	frames := make([]*modernHDMapFrame, 0, len(j.vgaFrames))
	for index := 114; index <= 124; index++ {
		f := commandHDObjectFrame(before, terrainVGA, j.palette)
		for _, tile := range before.Terrain {
			tile.Rect = tile.Rect.Add(image.Pt(-24, -24))
			f.Terrain = append(f.Terrain, tile)
		}
		layer := unitPresentHDGround(s, set, j.palette, index, has)
		f.Terrain = append(f.Terrain, modernHDTerrainDraw{Image: layer.Image, Rect: layer.Rect})
		frames = append(frames, unitPresentHDBorder(f, j.vgaFrames[len(frames)], j.palette))
	}
	weights := make(map[uint32]modernTerrainColourWeights)
	ground := unitPresentHDGround(s, set, j.palette, 124, has)
	radial := func(input indexedmap.NativeFrameInput, normal *modernHDMapFrame, pass fdother.UnitPresentLUTFrame, lut []byte) (*modernHDMapFrame, error) {
		in := nativeTransitionInput(input)
		in.Units, in.ForegroundUnits = nil, nil
		work, vga := make([]byte, indexedmap.NativeUnitPresentWorkSize), append([]byte(nil), s.InitialVGA...)
		if err := indexedmap.ComposeNativeUnitPresentLUTFrame(work, vga, s.Terrain, lut, in, pass); err != nil {
			return nil, err
		}
		return g.prepareModernHDRadialMap(modernHDRadialMapSpec{Input: input, Palette: j.palette, Pass: pass.Pass, LUT: lut, WorkOffset: unitPresentHDBase, Background: vga, Ground: []modernHDRadialGroundLayer{ground}}, set, normal, weights)
	}
	for i, p := range s.Passes[:6] {
		f, err := radial(s.Before, before, p, s.LUTs[i])
		if err != nil {
			return nil, err
		}
		frames = append(frames, f)
	}
	pass, err := fdother.NativeUnitPresentLUTPass(s.Cursor.X*24+12, s.Cursor.Y*24+15, 0)
	if err != nil {
		return nil, err
	}
	bridge, err := radial(s.After, after, fdother.UnitPresentLUTFrame{Pass: pass}, s.BridgeLUT)
	if err != nil {
		return nil, err
	}
	layout := indexedmap.NativeUnitPresentStripLayoutFor(int(s.Call.VisualX), int(s.Call.VisualY), s.Before.Frame.CameraX, s.Before.Frame.CameraY)
	rows, err := unitPresentHDStripRows(bridge, layout, j.workFrames[j.mutationAt], j.palette)
	if err != nil {
		return nil, err
	}
	contract := frames[len(frames)-1]
	var overlays []modernHDMapOverlayDraw
	for n, row := range rows {
		overlays = append(overlays, row...)
		f := unitPresentHDBorder(contract, j.vgaFrames[j.mutationAt+n], j.palette)
		f.Overlays = append([]modernHDMapOverlayDraw(nil), overlays...)
		frames = append(frames, f)
	}
	for i, p := range s.Passes[6:] {
		f, err := radial(s.After, after, p, s.LUTs[i+6])
		if err != nil {
			return nil, err
		}
		frames = append(frames, f)
	}
	if len(frames) != len(j.vgaFrames) {
		return nil, errors.New("modern HD unit present: incomplete HD schedule")
	}
	j.hdCached, j.hdCachedTheme = frames, set
	j.hdCachedBridge = bridge
	return frames, nil
}

// Only the bounded, UI-free after-world patch is rastered on the CPU. The
// normal GPU renderer still owns every full frame and final presentation.
func rasterUnitPresentHDRegion(frame *modernHDMapFrame, region image.Rectangle) (*image.RGBA, error) {
	if len(frame.Units) != 0 || frame.Range != nil || frame.HUDUnit != nil || len(frame.Overlays) != 0 {
		return nil, errors.New("modern HD unit present: unsupported strip plane")
	}
	for i := 3; i < len(frame.UI.Pix); i += 4 {
		if frame.UI.Pix[i] != 0 {
			return nil, errors.New("modern HD unit present: strip source contains UI")
		}
	}
	region = region.Intersect(image.Rect(0, 0, 312*6, 192*6))
	out := image.NewRGBA(region)
	paint := func(src image.Image, rect image.Rectangle) {
		at := rect.Intersect(region)
		for y := at.Min.Y; y < at.Max.Y; y++ {
			for x := at.Min.X; x < at.Max.X; x++ {
				v := color.RGBAModel.Convert(src.At(src.Bounds().Min.X+(x-rect.Min.X)*src.Bounds().Dx()/rect.Dx(), src.Bounds().Min.Y+(y-rect.Min.Y)*src.Bounds().Dy()/rect.Dy())).(color.RGBA)
				if v.A == 0 {
					continue
				}
				b := out.RGBAAt(x, y)
				a := 255 - int(v.A)
				mix := func(s, d byte) byte { return byte(min(255, int(s)+(int(d)*a+127)/255)) }
				out.SetRGBA(x, y, color.RGBA{mix(v.R, b.R), mix(v.G, b.G), mix(v.B, b.B), mix(v.A, b.A)})
			}
		}
	}
	paint(frame.Background, image.Rect(0, 0, 1920, 1200))
	for _, t := range frame.Terrain {
		paint(t.Image, t.Rect)
	}
	for _, f := range frame.Foreground {
		if f.HDImage != nil {
			paint(f.HDImage, f.Rect)
		} else {
			paint(f.Image, f.Rect)
		}
	}
	return out, nil
}

func unitPresentHDStripRows(after *modernHDMapFrame, layout indexedmap.NativeUnitPresentStripLayout, work []byte, palette color.Palette) ([][]modernHDMapOverlayDraw, error) {
	view := image.Rect(0, 0, 312, 192)
	source := image.Rectangle{}
	position := func(index int) image.Point {
		return image.Pt(index%456-unitPresentHDBase%456, index/456-unitPresentHDBase/456)
	}
	for row := 0; row < layout.Rows; row++ {
		for col := 0; col < 24; col++ {
			p := position(layout.WorkOffset + row*456 + col)
			if p.In(view) {
				source = source.Union(image.Rect(p.X, p.Y, p.X+1, p.Y+1))
			}
		}
	}
	region := image.Rect(source.Min.X*6, source.Min.Y*6, source.Max.X*6, source.Max.Y*6)
	picture, err := rasterUnitPresentHDRegion(after, region)
	if err != nil {
		return nil, err
	}
	result := make([][]modernHDMapOverlayDraw, layout.Rows)
	for row := 0; row < layout.Rows; row++ {
		for col := 0; col < 24; {
			dst := layout.VGAOffset + row*320 + col
			x, y := dst%320, dst/320
			width := min(24-col, 320-x)
			patch := image.NewRGBA(image.Rect(0, 0, width*6, 6))
			for n := 0; n < width; n++ {
				index := layout.WorkOffset + row*456 + col + n
				if index < 0 || index >= len(work) {
					return nil, errors.New("modern HD unit present: strip source outside work")
				}
				p := position(index)
				for yy := 0; yy < 6; yy++ {
					for xx := 0; xx < 6; xx++ {
						var c color.Color = palette[work[index]]
						if p.In(view) {
							c = picture.RGBAAt(p.X*6+xx, p.Y*6+yy)
						}
						patch.Set(n*6+xx, yy, c)
						if patch.RGBAAt(n*6+xx, yy).A != 255 {
							return nil, errors.New("modern HD unit present: strip background not opaque")
						}
					}
				}
			}
			result[row] = append(result[row], modernHDMapOverlayDraw{Entry: -1, Image: patch, Rect: image.Rect(x*6, y*6, (x+width)*6, (y+1)*6)})
			col += width
		}
	}
	return result, nil
}

func (g *Game) stageModernHDUnitPresent(set *modernStoryPortraitSet) (func(), error) {
	roots := []*nativeUnitPresentJob{g.nativeUnitPresent}
	if j := g.nativeModifierPresentation; j != nil {
		roots = append(roots, j.unitPresent)
	}
	if j := g.focusJob; j != nil {
		roots = append(roots, j.unitPresent)
	}
	seen := make(map[*nativeUnitPresentJob]bool)
	var jobs []*nativeUnitPresentJob
	var frames [][]*modernHDMapFrame
	for _, root := range roots {
		for j := root; j != nil && !seen[j]; j = j.hdNext {
			seen[j] = true
			prepared, err := g.prepareModernHDUnitPresent(j, set)
			if err != nil {
				return nil, err
			}
			jobs = append(jobs, j)
			frames = append(frames, prepared)
		}
	}
	return func() {
		for i, j := range jobs {
			j.hdFrames, j.hdTheme = frames[i], set
		}
	}, nil
}
