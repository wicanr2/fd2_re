package main

import (
	"errors"
	"image"
	"image/color"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

// The original HUD puts the sprite above terrain/AP/DP and below HP. Its
// 24x24 panel interior is opaque, so the whole patch can also restore the
// original HUD-over-ring order without accumulating translucent sprite edges.
type modernHDMapHUDUnit struct {
	Unit        modernHDMapUnitDraw
	Base, Above *image.RGBA
}

func (g *Game) prepareModernHDMapHUDUnit(in indexedmap.NativeFrameInput, palette color.Palette) (*modernHDMapHUDUnit, error) {
	hud := in.HUD
	if !hud.DisplayGateA || !hud.DisplayGateB || hud.OptionalUnit == nil {
		return nil, nil
	}
	u := hud.OptionalUnit
	if in.HUDUnits == nil || in.HUDCache == nil {
		return nil, errors.New("modern HD HUD: native unit source absent")
	}
	if _, err := fdicon.NativeMapHUDUnitFrameIndex(u.SelectorSlot, u.RawState); err != nil {
		return nil, err
	}
	cycle := u.RawState
	if cycle == 3 {
		cycle = 1
	}
	if _, err := in.HUDUnits.SpriteForNativeSlot(in.HUDCache, u.SelectorSlot, 0, cycle); err != nil {
		return nil, err
	}
	group, err := in.HUDCache.KeyForSlot(u.SelectorSlot)
	if err != nil {
		return nil, err
	}
	if !fdicon.NativeSourceGroupHasPixels(group) {
		return nil, nil
	}
	if g.modernStoryPortraits == nil {
		return nil, errors.New("modern HD HUD: theme absent")
	}
	frames := g.modernStoryPortraits.mapSpritesHD[group]
	if cycle >= len(frames) || frames[cycle] == nil || frames[cycle].Bounds() != image.Rect(0, 0, 144, 144) {
		return nil, errors.New("modern HD HUD: selected HD unit source absent or wrong size")
	}
	layout, err := fdicon.NativeMapHUDLayoutFor(hud.AnchorX, fdicon.NativeMapStride)
	if err != nil {
		return nil, err
	}
	baseInput := hud
	baseInput.OptionalUnit = nil
	base, err := modernHDOverwriteLayer(fdicon.NativeMapStride*200, palette, func(dst []byte) error {
		return indexedmap.BlitNativeMapHUD(in.Frames, in.HUDTerrain, in.HUDUnits, in.HUDCache, dst, baseInput)
	})
	if err != nil {
		return nil, err
	}
	above, err := modernHDOverwriteLayer(fdicon.NativeMapStride*200, palette, func(dst []byte) error {
		return indexedmap.BlitNativeMapHUDHP(in.Frames, dst, hud.AnchorX, u.Current, u.Maximum)
	})
	if err != nil {
		return nil, err
	}
	patch := &modernHDMapHUDUnit{Base: image.NewRGBA(image.Rect(0, 0, 24, 24)), Above: image.NewRGBA(image.Rect(0, 0, 24, 24))}
	for y := 0; y < 24; y++ {
		for x := 0; x < 24; x++ {
			at := layout.Unit + y*fdicon.NativeMapStride + x
			if base[at].A != 255 {
				return nil, errors.New("modern HD HUD: native panel icon interior is not opaque")
			}
			patch.Base.SetRGBA(x, y, base[at])
			patch.Above.SetRGBA(x, y, above[at])
		}
	}
	x, y := layout.Unit%fdicon.NativeMapStride+4, layout.Unit/fdicon.NativeMapStride+4
	patch.Unit = modernHDMapUnitDraw{Group: group, Frame: cycle, Rect: image.Rect(x*6, y*6, (x+24)*6, (y+24)*6)}
	return patch, nil
}

func clearModernHDHUDUnitPixels(ui *image.RGBA, patch *modernHDMapHUDUnit) {
	if patch == nil {
		return
	}
	r := patch.Unit.Rect
	for y := r.Min.Y / 6; y < r.Max.Y/6; y++ {
		for x := r.Min.X / 6; x < r.Max.X/6; x++ {
			ui.SetRGBA(x, y, color.RGBA{})
		}
	}
}
