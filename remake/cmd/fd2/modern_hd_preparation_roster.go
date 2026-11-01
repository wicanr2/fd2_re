package main

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

type modernPreparationRoster struct {
	theme *modernStoryPortraitSet
	units []modernHDMapUnitDraw
}

// This plan consumes exactly the native raw selector keys, positions and cycle.
// A blank original group stays blank; an incomplete HD group retains native art.
func (g *Game) prepareModernHDPreparationRoster() *modernPreparationRoster {
	if g.modernStoryPortraits == nil || g.nativePreparationUI == nil || g.nativePreparationUI.roster == nil ||
		!g.prepSelecting || g.prepConfirm || g.prepRecordSlots || g.nativeClassUIJob != nil ||
		g.camp == nil || g.camp.Node() == nil || g.camp.Node().Type != "preparation" ||
		len(g.prepIDs) == 0 || len(g.prepIDs) > fdother.NativePreparationRosterMaximum ||
		g.prepSel < 0 || g.prepSel >= len(g.prepIDs) {
		return nil
	}
	cycle, err := fdicon.NativeFrameIndex(0, false, g.prepIdleCycle, 0)
	if err != nil {
		return nil
	}
	bank := g.nativePreparationUI.roster.Units
	if bank == nil {
		return nil
	}
	plan := &modernPreparationRoster{theme: g.modernStoryPortraits}
	for i, id := range g.prepIDs {
		unit, ok := g.partyRoster[id]
		if !ok || !unit.HasMapSelectorKey {
			return nil
		}
		group := unit.MapSelectorKey
		sprite, err := bank.SpriteFor(group, 0, cycle)
		if err != nil || len(sprite.Pixels) != 24*24 || len(sprite.Mask) != len(sprite.Pixels) {
			return nil
		}
		if !fdicon.NativeSourceGroupHasPixels(group) {
			continue
		}
		frames := plan.theme.mapSpritesHD[group]
		if cycle >= len(frames) || frames[cycle] == nil || frames[cycle].Bounds() != image.Rect(0, 0, 144, 144) {
			return nil
		}
		selected := g.partyDeploy[id]
		x, y, err := fdother.NativePreparationRosterPosition(i, selected, false)
		if err != nil {
			return nil
		}
		plan.units = append(plan.units, modernHDMapUnitDraw{Group: group, Frame: cycle,
			Rect: image.Rect(x*6, y*6, (x+24)*6, (y+24)*6), PaletteBand: !selected})
	}
	return plan
}

func (g *Game) drawModernHDPreparationRoster(screen *ebiten.Image, capture *modernHDIndexedText) bool {
	plan := g.prepareModernHDPreparationRoster()
	if plan == nil {
		return false
	}
	compose := func(source []byte) ([]byte, bool) { return g.composeNativePreparationOnLayers(source, false) }
	if !g.drawModernHDPreparationBackground(screen, g.prepTownSource, g.prepPromptSource, compose, capture) {
		if capture != nil {
			capture.runs = nil
		}
		frame, ok := g.composeNativePreparationStableFrameLayers(false)
		if !ok {
			return false
		}
		g.presentNativeClassFrame(screen, frame)
		if capture != nil {
			g.modernHDText.runs = append(g.modernHDText.runs, capture.runs...)
		}
	}
	g.preparationRosterHD = plan
	return true
}

func (p *modernHDPresentation) drawModernHDPreparationRoster(screen ebiten.FinalScreen, geometry ebiten.GeoM, brightness float64) {
	g := p.Game
	plan := g.preparationRosterHD
	if plan == nil || plan.theme != g.modernStoryPortraits || !g.prepSelecting ||
		g.prepConfirm || g.prepRecordSlots || g.nativeClassUIJob != nil || g.transientUI || g.nativePlayerStatus != nil {
		return
	}
	set := plan.theme
	if set.townCursorGPU == nil {
		set.townCursorGPU = make(map[[2]int]*ebiten.Image)
	}
	for _, unit := range plan.units {
		key := [2]int{unit.Group, unit.Frame}
		picture := set.townCursorGPU[key]
		if picture == nil {
			picture = ebiten.NewImageFromImage(set.mapSpritesHD[unit.Group][unit.Frame])
			set.townCursorGPU[key] = picture
		}
		options := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		options.GeoM.Scale(1.0/3, 1.0/3)
		options.GeoM.Translate(float64(unit.Rect.Min.X)/3, float64(unit.Rect.Min.Y)/3)
		options.GeoM.Concat(geometry)
		if unit.PaletteBand {
			options.ColorScale.Scale(0.55, 0.55, 0.6, 1)
		}
		options.ColorScale.Scale(float32(brightness), float32(brightness), float32(brightness), 1)
		screen.DrawImage(picture, options)
	}
}
