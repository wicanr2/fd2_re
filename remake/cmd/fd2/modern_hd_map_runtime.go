package main

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

// A frame is prepared inside the original timing transaction and published
// only with its native work/VGA/state. TerrainSeed never contains objects/UI.
type modernHDMapFrame struct {
	Background       *image.RGBA
	UI               *image.RGBA
	Units            []modernHDMapUnitDraw
	Foreground       []modernHDMapForegroundDraw
	TerrainSeed      []byte
	TerrainUnderlay  []byte
	Terrain          []modernHDTerrainDraw
	Range            *image.RGBA
	TerrainColourKey [32]byte
	Overlays         []modernHDMapOverlayDraw
	Viewport         image.Rectangle
	OverlaysAboveUI  bool
	CompoundOwner    *modernHDCompoundMapScene
	HUDUnit          *modernHDMapHUDUnit
	TransientTerrain bool
	PaletteAdd6      int
}

type modernHDMapOverlayDraw struct {
	Entry int
	Image *image.RGBA
	Rect  image.Rectangle
}

func (g *Game) modernHDMapFrameAdmission(story, battleView bool) bool {
	if story && g.hasModernHDStoryMapSource() {
		return g.modernStoryPortraits != nil && len(g.modernStoryPortraits.mapSpritesHD) == 125
	}
	if g == nil || story || !battleView || g.modernStoryPortraits == nil ||
		len(g.modernStoryPortraits.mapSpritesHD) != 125 || !nativeMapAssetsAvailable(g.nativeMapAssets) ||
		g.st == nil || !g.st.HasNativeMapViewState {
		return false
	}
	if state := g.nativeSystemEndTurnUI; state != nil && state.rewardMessage != nil &&
		state.rewardMessage.hdWorld != nil && state.rewardMessage.hdTheme == g.modernStoryPortraits {
		return true
	}
	if g.spellOpen || g.itemOpen || g.castSp != nil {
		return false
	}
	return g.sel == nil || g.nativeCommand0Targeting || g.nativeItemTargeting ||
		g.nativeItemRelocating || g.ring || g.nativeCommandOpen || g.nativeMovePlan != nil
}

// The admitted UI blitters only overwrite source bytes or skip transparent
// pixels. Two independent initial values prove writes even when the actual
// background equals the source color, including palette index zero. Any
// detected destination-dependent operation fails closed.
func modernHDOverwriteLayer(size int, palette color.Palette, render func([]byte) error) ([]color.RGBA, error) {
	if size <= 0 || len(palette) != 256 || render == nil {
		return nil, errors.New("modern HD UI: incomplete overwrite input")
	}
	first, second := make([]byte, size), bytes.Repeat([]byte{255}, size)
	if err := render(first); err != nil {
		return nil, err
	}
	if err := render(second); err != nil {
		return nil, err
	}
	result := make([]color.RGBA, size)
	for i, value := range first {
		switch {
		case value == second[i]:
			if palette[value] == nil {
				return nil, errors.New("modern HD UI: missing palette color")
			}
			r, g, b, _ := palette[value].RGBA()
			result[i] = color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), 255}
		case value == 0 && second[i] == 255:
		default:
			return nil, fmt.Errorf("modern HD UI: destination-dependent write at %d", i)
		}
	}
	return result, nil
}

func modernHDNativeViewportPixels(work []color.RGBA, base int) (*image.RGBA, error) {
	result := image.NewRGBA(image.Rect(0, 0, 320, 200))
	for y := 0; y < 192; y++ {
		for x := 0; x < 312; x++ {
			i := base + y*fdicon.NativeMapStride + x
			if i < 0 || i >= len(work) {
				return nil, errors.New("modern HD UI: viewport exceeds source")
			}
			result.SetRGBA(x+4, y+4, work[i])
		}
	}
	return result, nil
}

func (g *Game) prepareModernHDMapFrame(in indexedmap.NativeFrameInput, seed []byte, palette color.Palette) (*modernHDMapFrame, error) {
	return g.prepareModernHDMapFrameMode(in, seed, palette, false)
}

func (g *Game) prepareModernHDMapFrameMode(in indexedmap.NativeFrameInput, seed []byte, palette color.Palette, event bool) (*modernHDMapFrame, error) {
	return g.prepareModernHDMapFrameLayout(in, seed, palette, event, -1)
}

// row is -1 for the steady owner, or 0/1 for the recovered 0x24B4D source row.
func (g *Game) prepareModernHDMapFrameLayout(in indexedmap.NativeFrameInput, seed []byte, palette color.Palette, event bool, row int) (*modernHDMapFrame, error) {
	return g.prepareModernHDMapFrameStageLayout(in, seed, palette, event, row, nil)
}

func (g *Game) prepareModernHDMapFrameStageLayout(in indexedmap.NativeFrameInput, seed []byte, palette color.Palette, event bool, row int, stage *nativeCh23AdapterState) (*modernHDMapFrame, error) {
	if row < -1 || row > 1 || g.modernStoryPortraits == nil || len(palette) != 256 || len(seed) != indexedmap.NativeUnitPresentWorkSize {
		return nil, errors.New("modern HD world: incomplete published sources")
	}
	frame := in.Frame
	frame.Cells = append([]fdicon.NativeTerrainCell(nil), frame.Cells...)
	background := frame
	background.Units, background.ForegroundUnits, background.SelectionOverlay = nil, nil, nil
	work, vga := append([]byte(nil), seed...), make([]byte, indexedmap.NativeMapVGASize)
	base := indexedmap.NativeUnitPresentStripLayoutFor(0, 0, 0, 0).WorkOffset
	viewportBase := base
	var terrainSeed []byte
	observer := func(stage indexedmap.FrameStage, work, vga []byte) error {
		if stage == indexedmap.FrameStageTerrain {
			terrainSeed = append([]byte(nil), work...)
		}
		return nil
	}
	unitY, motion, up := g.nativeCh12UpWalkFrame()
	step := up || g.walk != nil || g.bannerT > 0
	if event {
		unitY, motion, up, step = 0, 0, false, false
	}
	var err error
	terrainBase, terrainCameraY, terrainRows := base, frame.CameraY, 8
	if row >= 0 {
		terrainRows, viewportBase = 9, base+row*fdicon.NativeMapStride
		if err = indexedmap.ComposeNative24B4DStaging(work, nativeTransitionInput(in)); err != nil {
			return nil, err
		}
	}
	if up {
		terrainBase, terrainCameraY, terrainRows, viewportBase, err = indexedmap.NativeUpStepLayout(background, unitY, motion)
		if err != nil {
			return nil, err
		}
	}
	var colours modernTerrainColourContext
	var terrain []modernHDTerrainDraw
	var terrainAssets map[int]*modernTerrainHDAsset
	if len(g.modernStoryPortraits.terrainHD) != 0 {
		if g.nativeMapAssets == nil {
			return nil, errors.New("modern HD terrain: native assets missing")
		}
		mapID := g.nativeMapAssets.MapIndex
		terrainAssets = g.modernStoryPortraits.terrainHD[mapID]
		if len(terrainAssets) != 0 {
			if err = g.modernStoryPortraits.validateTerrainHD(mapID, frame.TerrainBank); err != nil {
				return nil, err
			}
			colours, err = terrainColourContext(mapID, g.nativeMapAssets.Palette, palette, frame.LUT)
			if err != nil {
				return nil, err
			}
			terrain, err = buildModernHDTerrainPlan(frame, terrainBase, terrainCameraY, terrainRows, viewportBase, terrainAssets, colours)
			if err != nil {
				return nil, err
			}
		}
	}
	switch {
	case up:
		_, _, _, viewportBase, err = indexedmap.NativeUpStepLayout(background, unitY, motion)
		if err == nil {
			err = indexedmap.ComposeNativeUpStepFrameObserved(work, vga, background, unitY, motion, observer)
		}
	case step:
		err = indexedmap.ComposeNativeStepFrameObserved(work, vga, background, observer)
	default:
		err = indexedmap.ComposeFrameObserved(work, vga, background, func([]byte) error { return nil }, observer)
	}
	if err != nil {
		return nil, err
	}
	if len(terrainSeed) != len(seed) {
		return nil, errors.New("modern HD world: terrain stage absent")
	}
	counts := make(map[int]int, len(g.modernStoryPortraits.mapSpritesHD))
	for group, images := range g.modernStoryPortraits.mapSpritesHD {
		counts[group] = len(images)
	}
	units, err := buildModernHDMapUnitPlan(frame, viewportBase, counts)
	if err != nil {
		return nil, err
	}
	for _, unit := range units {
		picture := g.modernStoryPortraits.mapSpritesHD[unit.Group][unit.Frame]
		if picture == nil || picture.Bounds() != image.Rect(0, 0, 144, 144) {
			return nil, errors.New("modern HD world: invalid unit source")
		}
	}
	// ComposeFrame commits selector-six cell mutations into its private cells.
	frame.Cells = background.Cells
	foreground, err := buildModernHDMapForegroundPlan(frame, viewportBase, palette)
	if err != nil {
		return nil, err
	}
	for i := range foreground {
		f := &foreground[i]
		if asset := terrainAssets[f.SourceFrame]; asset.supports("foreground") {
			f.HDImage = asset.shade(colours, f.BlitMode == 0xff)
		}
	}
	stageImage, err := g.prepareModernHDMapStage(in.Frame, palette, row, stage)
	if err != nil {
		return nil, err
	}
	var rangeLayer *image.RGBA
	if (len(terrain) > 0 || stageImage != nil) && !step && frame.RangeMode != 0 && frame.RangeMode != 6 {
		pixels, err := modernHDOverwriteLayer(len(work), palette, func(dst []byte) error {
			return fdother.BlitNativeRangeOverlay(frame.RangeBank, dst, frame.CameraX, frame.CameraY, 13, 8,
				frame.RangeMode, frame.CursorX, frame.CursorY)
		})
		if err != nil {
			return nil, err
		}
		rangeLayer, err = modernHDNativeViewportPixels(pixels, viewportBase)
		if err != nil {
			return nil, err
		}
	}
	bg := image.NewRGBA(image.Rect(0, 0, 320, 200))
	draw.Draw(bg, bg.Bounds(), image.NewUniform(color.Black), image.Point{}, draw.Src)
	for y := 4; y < 196; y++ {
		for x := 4; x < 316; x++ {
			if row >= 0 {
				bg.Set(x, y, palette[work[viewportBase+(y-4)*fdicon.NativeMapStride+x-4]])
			} else {
				bg.Set(x, y, palette[vga[y*320+x]])
			}
		}
	}
	ui := image.NewRGBA(bg.Bounds())
	if !step {
		pixels, err := modernHDOverwriteLayer(len(work), palette, func(dst []byte) error {
			if frame.SelectionOverlay != nil {
				if err := frame.SelectionOverlay(dst, fdicon.NativeMapStride, base); err != nil {
					return err
				}
			}
			return indexedmap.BlitNativeMapHUD(in.Frames, in.HUDTerrain, in.HUDUnits, in.HUDCache, dst[base:], in.HUD)
		})
		if err != nil {
			return nil, err
		}
		ui, err = modernHDNativeViewportPixels(pixels, viewportBase)
		if err != nil {
			return nil, err
		}
	}
	if offset, visible := g.phaseBannerSlideOffset(); !event && visible && g.phaseBannerGlyphsAvailable() {
		pixels, err := modernHDOverwriteLayer(indexedmap.NativeMapVGASize, palette, func(dst []byte) error {
			g.blitPhaseBannerGlyphs(dst, offset)
			return nil
		})
		if err != nil {
			return nil, err
		}
		for y := 0; y < 200; y++ {
			for x := 0; x < 320; x++ {
				if pixel := pixels[y*320+x]; pixel.A != 0 {
					ui.SetRGBA(x, y, pixel)
				}
			}
		}
	}
	var hudUnit *modernHDMapHUDUnit
	if !step {
		hudUnit, err = g.prepareModernHDMapHUDUnit(in, palette)
		if err != nil {
			return nil, err
		}
		if row >= 0 && hudUnit != nil {
			hudUnit.Unit.Rect = hudUnit.Unit.Rect.Add(image.Pt(0, -row*6))
		}
		clearModernHDHUDUnitPixels(ui, hudUnit)
	}
	result := &modernHDMapFrame{Background: bg, UI: ui, Units: units, Foreground: foreground, TerrainSeed: terrainSeed,
		TerrainUnderlay: append([]byte(nil), seed...), Terrain: terrain, Range: rangeLayer, TerrainColourKey: colours.Key, HUDUnit: hudUnit}
	if stageImage != nil {
		result.Terrain = append([]modernHDTerrainDraw{{Image: stageImage, Rect: modernHDMapViewport(), Stage: true}}, result.Terrain...)
		result.TransientTerrain = true
	}
	if stage != nil && g.nativeMapAssets != nil && g.nativeMapAssets.MapIndex == 23 {
		result.TransientTerrain = true
		if err := g.colourModernHDStageFigures(result, in, palette); err != nil {
			return nil, err
		}
	}
	return result, nil
}
