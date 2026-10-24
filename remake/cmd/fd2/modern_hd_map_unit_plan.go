package main

import (
	"errors"
	"fmt"
	"image"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

type modernHDMapUnitDraw struct {
	Group, Frame int
	Rect         image.Rectangle
	WorkOffset   int
	PaletteBand  bool
}

func modernHDMapViewport() image.Rectangle {
	return image.Rect(indexedmap.NativeMapViewportX0*6, indexedmap.NativeMapViewportY0*6,
		(indexedmap.NativeMapViewportX1+1)*6, (indexedmap.NativeMapViewportY1+1)*6)
}

// The caller owns the viewport source, including known shifted walking copies.
// This plan neither advances animation clocks nor publishes game state.
func buildModernHDMapUnitPlan(in indexedmap.FrameInput, viewportBase int, frameCounts map[int]int) ([]modernHDMapUnitDraw, error) {
	return buildModernHDMapUnitRegionPlan(in, viewportBase, frameCounts, 12, 8)
}

func buildModernHDMapUnitRegionPlan(in indexedmap.FrameInput, viewportBase int, frameCounts map[int]int, xmax, ymax int) ([]modernHDMapUnitDraw, error) {
	stride := fdicon.NativeMapStride
	view := modernHDMapViewport()
	if in.UnitBank == nil || in.SelectorCache == nil || viewportBase < 0 ||
		viewportBase%stride+view.Dx()/6 > stride ||
		viewportBase+(view.Dy()/6-1)*stride+view.Dx()/6 > indexedmap.NativeUnitPresentWorkSize {
		return nil, errors.New("modern HD units: incomplete native input or viewport")
	}
	commands := make([]modernHDMapUnitDraw, 0, len(in.Units))
	for _, unit := range in.Units {
		if unit.Inactive || unit.X < in.CameraX-1 || unit.X > in.CameraX+xmax || unit.Y < in.CameraY-1 || unit.Y > in.CameraY+ymax {
			continue
		}
		cycle, err := fdicon.NativeFrameIndex(unit.MotionOffset, unit.ForceBase, in.IdleCycle, in.MovingCycle)
		if err != nil {
			return nil, err
		}
		sprite, err := in.UnitBank.SpriteForNativeSlot(in.SelectorCache, unit.Slot, unit.Pose, cycle)
		if err != nil {
			return nil, err
		}
		offset, err := fdicon.NativePlacementOffset(unit.X, unit.Y, in.CameraX, in.CameraY, unit.Pose, unit.MotionOffset, in.PixelShift, unit.ForceBase)
		if err != nil {
			return nil, err
		}
		if offset < 0 {
			continue
		}
		if offset+23*stride+24 > indexedmap.NativeUnitPresentWorkSize {
			return nil, errors.New("modern HD units: native unit exceeds work buffer")
		}
		if len(sprite.Pixels) != 24*24 || len(sprite.Mask) != 24*24 {
			return nil, errors.New("modern HD units: incomplete original sprite")
		}
		writes := false
		for _, value := range sprite.Mask {
			if value != 0 {
				writes = true
				break
			}
		}
		if !writes {
			continue
		}
		group, err := in.SelectorCache.KeyForSlot(unit.Slot)
		if err != nil {
			return nil, err
		}
		frame := unit.Pose*3 + cycle
		if count, ok := frameCounts[group]; !ok || frame < 0 || frame >= count {
			return nil, fmt.Errorf("modern HD units: missing group %d frame %d", group, frame)
		}
		x := offset%stride - viewportBase%stride
		y := offset/stride - viewportBase/stride
		rect := image.Rect(view.Min.X+x*6, view.Min.Y+y*6, view.Min.X+(x+24)*6, view.Min.Y+(y+24)*6)
		commands = append(commands, modernHDMapUnitDraw{Group: group, Frame: frame, Rect: rect, WorkOffset: offset, PaletteBand: unit.Flags&0x80 != 0})
	}
	return commands, nil
}
