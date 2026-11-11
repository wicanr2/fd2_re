package main

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

// Keep FDOTHER#2's raised frame and selection phases around the HD artwork.
// Both artwork modes share this interface painter.
func (g *Game) drawModernActionIcon(target, icon *ebiten.Image, rect image.Rectangle, index int, enabled bool) error {
	if icon == nil || index < 0 || index >= len(g.nativeActionCells) || g.nativeActionCells[index] == nil || rect.Empty() {
		return fmt.Errorf("modern action icon: native frame unavailable")
	}
	frame, err := g.goldActionFrame(index, enabled)
	if err != nil {
		return err
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(rect.Dx())/float64(frame.Bounds().Dx()), float64(rect.Dy())/float64(frame.Bounds().Dy()))
	op.GeoM.Translate(float64(rect.Min.X), float64(rect.Min.Y))
	target.DrawImage(frame, op)
	return nil
}

func (g *Game) drawModernNativeMapRing(screen *ebiten.Image) error {
	state := g.nativeActionOverlayState()
	frame, closing := g.actionOverlayRenderState()
	offsets, err := fdother.ActionOverlayFrameOffsets(frame, closing)
	if err != nil {
		return err
	}
	steady := !g.actionOverlayBlocksInput()
	if steady && g.nativeActionOverlayOpenFrameVariant {
		offsets, err = fdother.ActionOverlayFrameOffsets(3, false)
		steady = false
	} else if steady {
		offsets = [4]int{-0x23a0, 0x378, 0x3a8, 0x2ac0}
	}
	if err != nil {
		return err
	}
	view := g.st.NativeMapViewState
	origin, err := fdother.ActionOverlayOrigin(view.VisibleCursorX, view.VisibleCursorY)
	if err != nil {
		return err
	}
	base := indexedmap.NativeUnitPresentStripLayoutFor(0, 0, 0, 0).WorkOffset
	for direction, offset := range offsets {
		index, err := state.CellIndex(direction)
		enabled := fdother.ActionOverlayAcceptsDirection(state.Availability, direction)
		if err == nil && steady && direction == g.ringSel && enabled {
			index, err = state.SelectedCellIndex(direction, g.actionOverlayBlink.Phase)
		}
		if err != nil {
			return err
		}
		position := origin + offset
		x := (position%fdicon.NativeMapStride - base%fdicon.NativeMapStride + 4) * 2
		y := (position/fdicon.NativeMapStride - base/fdicon.NativeMapStride + 4) * 2
		if err := g.drawModernActionIcon(screen, g.ringIcons[direction], image.Rect(x, y, x+56, y+52), index, enabled); err != nil {
			return err
		}
	}
	if !closing {
		if err := g.drawNativeRingCursorUnit(screen, base); err != nil {
			return err
		}
	}
	g.markActionOverlayDrawn()
	return nil
}

// Reuse the native sprite, placement and palette-band writer. The normalized
// map sprite painter uses a different origin and must not redraw this unit.
func (g *Game) drawNativeRingCursorUnit(screen *ebiten.Image, base int) error {
	view := g.st.NativeMapViewState
	for _, raw := range g.st.Units {
		if raw == nil || raw.X != view.CursorX || raw.Y != view.CursorY {
			continue
		}
		entry, ok := raw.NativeUnitLayerEntry()
		if !ok {
			return fmt.Errorf("modern action icon: native cursor unit unavailable")
		}
		if entry.Inactive {
			continue
		}
		cycles := g.st.NativeMapCycleState
		cycle, err := fdicon.NativeFrameIndex(entry.MotionOffset, entry.ForceBase, cycles.Idle, cycles.Moving)
		if err != nil {
			return err
		}
		a := g.nativeMapAssets
		sprite, err := a.Units.SpriteForNativeSlot(g.st.NativeMapSelectorCache, entry.Slot, entry.Pose, cycle)
		if err != nil {
			return err
		}
		offset, err := fdicon.NativePlacementOffset(entry.X, entry.Y, view.CameraX, view.CameraY, entry.Pose,
			entry.MotionOffset, g.st.NativeUnitPixelShiftState.Value, entry.ForceBase)
		if err != nil {
			return err
		}
		work := make([]byte, indexedmap.NativeUnitPresentWorkSize)
		if err := a.Units.BlitNativeUnitLayer(work, fdicon.NativeMapStride, g.st.NativeMapSelectorCache,
			[]fdicon.NativeUnitLayerEntry{entry}, view.CameraX, view.CameraY, 12, 7, cycles.Idle, cycles.Moving,
			g.st.NativeUnitPixelShiftState.Value); err != nil {
			return err
		}
		palette, err := fdother.VGAPaletteFromDAC(g.nativeMapDAC)
		if err != nil {
			return err
		}
		picture := image.NewRGBA(image.Rect(0, 0, 24, 24))
		for y := 0; y < 24; y++ {
			for x := 0; x < 24; x++ {
				if sprite.Mask[y*24+x] != 0 {
					picture.Set(x, y, palette[work[offset+y*fdicon.NativeMapStride+x]])
				}
			}
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		op.GeoM.Translate(float64((offset%fdicon.NativeMapStride-base%fdicon.NativeMapStride+4)*2),
			float64((offset/fdicon.NativeMapStride-base/fdicon.NativeMapStride+4)*2))
		screen.SubImage(image.Rect(8, 8, 632, 392)).(*ebiten.Image).DrawImage(ebiten.NewImageFromImage(picture), op)
		return nil
	}
	return nil
}
