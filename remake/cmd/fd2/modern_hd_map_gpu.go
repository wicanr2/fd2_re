package main

import (
	"errors"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

type modernHDMapGPU struct {
	World, Background, UI *ebiten.Image
	Foreground            []*ebiten.Image
	Units                 map[int]map[int]*ebiten.Image
	Theme                 *modernStoryPortraitSet
	Banner                *ebiten.Image
	BannerName            string
	Mosaic                *ebiten.Shader
	Pixel                 *ebiten.Image
	Dialogue              *ebiten.Image
	TerrainImages         map[*image.RGBA]*ebiten.Image
	TerrainKey            [32]byte
	Range                 *ebiten.Image
}

func (r *modernHDMapGPU) terrainImage(picture *image.RGBA) *ebiten.Image {
	if r.TerrainImages == nil {
		r.TerrainImages = make(map[*image.RGBA]*ebiten.Image)
	}
	if r.TerrainImages[picture] == nil {
		r.TerrainImages[picture] = ebiten.NewImageFromImage(picture)
	}
	return r.TerrainImages[picture]
}

func (r *modernHDMapGPU) drawTerrain(picture *image.RGBA, rect image.Rectangle) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(rect.Min.X), float64(rect.Min.Y))
	r.World.SubImage(modernHDMapViewport()).(*ebiten.Image).DrawImage(r.terrainImage(picture), op)
}

const modernHDMapMosaicSource = `//kage:unit pixels
package main
var BlockSize float
var Dim float
func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
    origin:=imageSrc0Origin()
    p:=texCoord-origin
    sample:=texCoord
    if p.x>=24 && p.y>=24 && p.x<1896 && p.y<1176 {
        sample=origin+vec2(24,24)+floor((p-vec2(24,24))/BlockSize)*BlockSize
    }
    c:=imageSrc0At(sample)
    return vec4(max(c.rgb-vec3(Dim)*c.a,vec3(0)),c.a)
}`

func (r *modernHDMapGPU) unit(group, frame int) *ebiten.Image {
	if r.Units[group] == nil {
		r.Units[group] = make(map[int]*ebiten.Image)
	}
	if r.Units[group][frame] == nil {
		r.Units[group][frame] = ebiten.NewImageFromImage(r.Theme.mapSpritesHD[group][frame])
	}
	return r.Units[group][frame]
}

func (r *modernHDMapGPU) drawUnit(unit modernHDMapUnitDraw) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(unit.Rect.Min.X), float64(unit.Rect.Min.Y))
	if unit.PaletteBand {
		op.ColorScale.Scale(0.55, 0.55, 0.6, 1)
	}
	r.World.SubImage(modernHDMapViewport()).(*ebiten.Image).DrawImage(r.unit(unit.Group, unit.Frame), op)
}

func (g *Game) drawModernHDMapFrame(screen *ebiten.Image) bool {
	g.modernHDMapComposing = true
	err := g.composeNativeMapFrame()
	g.modernHDMapComposing = false
	if err != nil {
		g.loadErr = err.Error()
		screen.Clear()
		return false
	}
	frame := g.modernHDMapFrame
	if frame == nil {
		g.loadErr = "modern HD world: transaction did not publish a frame"
		screen.Clear()
		return false
	}
	frame, err = g.freezeModernHDTimedDialogue(frame)
	if err != nil {
		g.loadErr = err.Error()
		screen.Clear()
		return false
	}
	r := g.modernHDMapGPU
	if r == nil || r.Theme != g.modernStoryPortraits {
		r = &modernHDMapGPU{World: ebiten.NewImage(modernHDWidth, modernHDHeight),
			Background: ebiten.NewImage(320, 200), UI: ebiten.NewImage(320, 200),
			Units: make(map[int]map[int]*ebiten.Image), Theme: g.modernStoryPortraits}
		r.Pixel = ebiten.NewImage(1, 1)
		r.Pixel.Fill(color.White)
		g.modernHDMapGPU = r
	}
	r.World.Clear()
	if r.TerrainKey != frame.TerrainColourKey {
		for _, texture := range r.TerrainImages {
			texture.Dispose()
		}
		r.TerrainImages = nil
		r.TerrainKey = frame.TerrainColourKey
	}
	r.Background.WritePixels(frame.Background.Pix)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(6, 6)
	r.World.DrawImage(r.Background, op)
	for _, terrain := range frame.Terrain {
		r.drawTerrain(terrain.Image, terrain.Rect)
	}
	if frame.Range != nil {
		if r.Range == nil {
			r.Range = ebiten.NewImage(320, 200)
		}
		r.Range.WritePixels(frame.Range.Pix)
		r.World.DrawImage(r.Range, op)
	}
	for _, unit := range frame.Units {
		r.drawUnit(unit)
	}
	for i, foreground := range frame.Foreground {
		if foreground.HDImage != nil {
			r.drawTerrain(foreground.HDImage, foreground.Rect)
			continue
		}
		for i >= len(r.Foreground) {
			r.Foreground = append(r.Foreground, ebiten.NewImage(24, 24))
		}
		r.Foreground[i].WritePixels(foreground.Image.Pix)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(6, 6)
		op.GeoM.Translate(float64(foreground.Rect.Min.X), float64(foreground.Rect.Min.Y))
		r.World.SubImage(modernHDMapViewport()).(*ebiten.Image).DrawImage(r.Foreground[i], op)
	}
	if g.bannerT > 0 && g.banner != "" {
		if r.Banner == nil {
			r.Banner = ebiten.NewImage(modernHDWidth, modernHDHeight)
		}
		if r.BannerName != g.banner {
			r.Banner.Clear()
			r.Banner.DrawImage(r.World, nil)
			r.BannerName = g.banner
		}
		step := g.phaseBannerStep()
		r.World.Clear()
		if step < 0 {
			r.World.DrawImage(r.Banner, nil)
		} else {
			if r.Mosaic == nil {
				r.Mosaic, err = ebiten.NewShader([]byte(modernHDMapMosaicSource))
				if err != nil {
					g.loadErr = err.Error()
					return false
				}
			}
			op := &ebiten.DrawRectShaderOptions{}
			op.Images[0] = r.Banner
			op.Uniforms = map[string]interface{}{"BlockSize": float32(indexedmap.PhaseBannerBlock(step) * 6), "Dim": float32(indexedmap.PhaseBannerDim(step)) / 63}
			r.World.DrawRectShader(modernHDWidth, modernHDHeight, r.Mosaic, op)
		}
	} else {
		r.BannerName = ""
	}
	screen.Clear()
	g.clearModernHDText()
	r.UI.WritePixels(frame.UI.Pix)
	uiop := &ebiten.DrawImageOptions{}
	uiop.GeoM.Scale(2, 2)
	screen.DrawImage(r.UI, uiop)
	g.modernHDWorldVisible = true
	g.modernHDMapPresentedFrame = g.frame
	g.modernHDMapPresented = frame
	return true
}

// The modern cells use the same raw origin/offsets, availability and Draw
// acknowledgement as the native action overlay, above foreground but below HUD.
func (g *Game) drawModernHDMapRing() error {
	r := g.modernHDMapGPU
	if r == nil || g.st == nil || !g.st.HasNativeMapViewState || len(g.ringIcons) != 4 {
		return errors.New("modern HD ring: incomplete sources")
	}
	overlay := g.nativeActionOverlayState()
	frame, closing := g.actionOverlayRenderState()
	offsets, err := fdother.ActionOverlayFrameOffsets(frame, closing)
	if err != nil {
		return err
	}
	steady := !g.actionOverlayBlocksInput()
	if steady && g.nativeActionOverlayOpenFrameVariant {
		offsets, err = fdother.ActionOverlayFrameOffsets(3, false)
		steady = false
	}
	if err != nil {
		return err
	}
	if steady {
		offsets = [4]int{-0x23a0, 0x378, 0x3a8, 0x2ac0}
	}
	view := g.st.NativeMapViewState
	origin, err := fdother.ActionOverlayOrigin(view.VisibleCursorX, view.VisibleCursorY)
	if err != nil {
		return err
	}
	base := indexedmap.NativeUnitPresentStripLayoutFor(0, 0, 0, 0).WorkOffset
	for direction, offset := range offsets {
		if _, err := overlay.CellIndex(direction); err != nil {
			return err
		}
		icon := g.ringIcons[direction]
		if icon == nil {
			return errors.New("modern HD ring: missing icon")
		}
		position := origin + offset
		x := (position%fdicon.NativeMapStride - base%fdicon.NativeMapStride + 4) * 6
		y := (position/fdicon.NativeMapStride - base/fdicon.NativeMapStride + 4) * 6
		if steady && direction == g.ringSel && fdother.ActionOverlayAcceptsDirection(overlay.Availability, direction) {
			if _, err := overlay.SelectedCellIndex(direction, g.actionOverlayBlink.Phase); err != nil {
				return err
			}
			shade := color.RGBA{255, 168, 32, 255}
			if g.actionOverlayBlink.Phase != 0 {
				shade = color.RGBA{255, 224, 64, 255}
			}
			for _, rect := range [][4]int{{x - 9, y - 9, 186, 9}, {x - 9, y + 156, 186, 9}, {x - 9, y, 9, 156}, {x + 168, y, 9, 156}} {
				border := &ebiten.DrawImageOptions{}
				border.GeoM.Scale(float64(rect[2]), float64(rect[3]))
				border.GeoM.Translate(float64(rect[0]), float64(rect[1]))
				border.ColorScale.ScaleWithColor(shade)
				r.World.SubImage(modernHDMapViewport()).(*ebiten.Image).DrawImage(r.Pixel, border)
			}
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(6, 6)
		op.GeoM.Translate(float64(x), float64(y))
		if !fdother.ActionOverlayAcceptsDirection(overlay.Availability, direction) {
			op.ColorScale.Scale(0.55, 0.55, 0.6, 1)
		}
		r.World.SubImage(modernHDMapViewport()).(*ebiten.Image).DrawImage(icon, op)
	}
	if !closing && g.modernHDMapFrame != nil {
		for _, raw := range g.st.Units {
			if raw == nil || raw.X != view.CursorX || raw.Y != view.CursorY {
				continue
			}
			entry, ok := raw.NativeUnitLayerEntry()
			if !ok {
				return errors.New("modern HD ring: cursor provenance absent")
			}
			if entry.Inactive {
				continue
			}
			counts := make(map[int]int, len(r.Theme.mapSpritesHD))
			for group, frames := range r.Theme.mapSpritesHD {
				counts[group] = len(frames)
			}
			input := indexedmap.FrameInput{UnitBank: g.nativeMapAssets.Units, SelectorCache: g.st.NativeMapSelectorCache,
				Units: []fdicon.NativeUnitLayerEntry{entry}, CameraX: view.CameraX, CameraY: view.CameraY,
				IdleCycle: g.st.NativeMapCycleState.Idle, MovingCycle: g.st.NativeMapCycleState.Moving, PixelShift: g.st.NativeUnitPixelShiftState.Value}
			plans, err := buildModernHDMapUnitPlan(input, base, counts)
			if err != nil {
				return err
			}
			for _, plan := range plans {
				r.drawUnit(plan)
			}
			g.markActionOverlayDrawn()
			return nil
		}
	}
	g.markActionOverlayDrawn()
	return nil
}

func (g *Game) modernHDMapLogicalImage(screen *ebiten.Image) *ebiten.Image {
	if !g.modernHDWorldVisible || g.modernHDMapGPU == nil {
		return screen
	}
	result := ebiten.NewImage(logicalW, logicalH)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(1.0/3, 1.0/3)
	result.DrawImage(g.modernHDMapGPU.World, op)
	result.DrawImage(screen, nil)
	return result
}
