package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

type modernFacilityRosterSource struct {
	panel                        fdother.LMI1Entry
	ids                          []int
	selected, start, cycle, cols int
	owner, mode                  string
}

type modernFacilityRosterUnit struct {
	group, frame int
	origin       image.Point // Native 320x200 coordinates.
}

type modernFacilityRoster struct {
	theme    *modernStoryPortraitSet
	source   modernFacilityRosterSource
	groups   []int
	units    []modernFacilityRosterUnit
	underlay *image.RGBA
}

func (g *Game) modernFacilityRosterSource() (s modernFacilityRosterSource, ok bool) {
	if g.camp == nil || g.camp.Node() == nil || g.nativeClassUI == nil ||
		g.nativeClassUI.units == nil || g.nativeClassUIJob != nil || g.nativeChurchUIJob != nil ||
		g.nativeShopUIJob != nil || g.helpVisible || g.transientUI || g.nativePlayerStatus != nil ||
		g.nativeFullDACBlack || g.nativeFullDACWhite {
		return
	}
	s.owner, s.cols = g.camp.Node().Type, 2
	s.panel = g.nativeClassUI.panel
	switch s.owner {
	case "shop":
		assets, _, _, valid := g.nativeShopState()
		if !valid || assets == nil || g.camp.Node().NativeHubVariant != g.nativeShopVariant ||
			assets.ResourceID != map[int]int{1: 12, 3: 29, 5: 63}[g.nativeShopVariant] {
			return s, false
		}
		s.panel, s.mode = assets.Panel, g.nativeShopMode
		switch s.mode {
		case "recipient_consumable", "recipient_equipment":
			s.ids, s.selected, s.start, s.cycle = g.shopRecipients, g.shopRecipientSel, g.nativeShopRecipientStart, g.nativeShopRecipientCycle
			if s.mode == "recipient_equipment" {
				s.cols = 1
			}
		case "sell_roster":
			s.ids, s.selected, s.start, s.cycle = g.partyJoinOrder, g.shopSellUnitSel, g.nativeShopSellRosterTop, g.nativeShopSellRosterCycle
		case "equip_roster":
			s.ids, s.selected, s.start = g.partyJoinOrder, g.nativeShopEquipUnitSel, g.nativeShopEquipRosterTop
		case "transfer_source", "transfer_dest":
			s.ids, s.selected, s.start = g.nativeShopTransferIDs, g.nativeShopTransferSel, g.nativeShopTransferTop
		default:
			return s, false
		}
	case "church":
		s.mode, s.ids, s.selected = g.churchMode, g.churchIDs, g.churchSel
		switch s.mode {
		case "class", "revive":
			s.cols, s.start = 1, g.churchVerticalStart
		case "status_roster", "transfer_source", "transfer_dest":
			s.start = g.churchRosterStart
		default:
			return s, false
		}
	default:
		return s, false
	}
	start, visible := campaign.NativeTwoColumnWindow(len(s.ids), s.selected, s.start)
	if s.cols == 1 {
		start, visible = campaign.NativeThreeRowWindow(len(s.ids), s.selected, s.start)
	}
	if visible == 0 || start != s.start || s.cycle < 0 || s.cycle > 2 {
		return s, false
	}
	s.ids = append([]int(nil), s.ids[start:start+visible]...)
	return s, true
}

// Rebuild each complete native sprite rectangle from the proven opaque panel
// and the same actual raw sprite. A covered rectangle rejects the whole plan.
func (g *Game) prepareModernHDFacilityRoster(frame []byte) *modernFacilityRoster {
	if g.modernStoryPortraits == nil || len(frame) != 64000 {
		return nil
	}
	s, ok := g.modernFacilityRosterSource()
	if !ok || s.panel.Width != 310 || s.panel.Height != 86 || len(s.panel.Pixels) != 310*86 {
		return nil
	}
	sum := sha256.Sum256(s.panel.Pixels)
	if hex.EncodeToString(sum[:]) != "c009ad5ed72466295e3c60d4cb5117c3331444bcbe23d5d1befd2f413b15ac00" {
		return nil
	}
	palette, ok := preparationCursorPalette(g.nativeClassUI.palette)
	if !ok {
		return nil
	}
	plan := &modernFacilityRoster{theme: g.modernStoryPortraits, source: s,
		underlay: image.NewRGBA(image.Rect(0, 0, 320, 200))}
	for row, id := range s.ids {
		unit, exists := g.partyRoster[id]
		if !exists || !unit.HasNativeIdentity || !unit.HasMapSelectorKey {
			return nil
		}
		group := unit.MapSelectorKey
		sprite, err := g.nativeClassUI.units.SpriteFor(group, 0, s.cycle)
		if err != nil || len(sprite.Pixels) != 576 || len(sprite.Mask) != 576 || len(sprite.RemapMask) != 576 {
			return nil
		}
		origin := image.Pt(14+132*(row%s.cols), 117+26*(row/s.cols))
		base := make([]byte, 576)
		for y := 0; y < 24; y++ {
			at := (origin.Y+y-112)*310 + origin.X - 5
			copy(base[y*24:(y+1)*24], s.panel.Pixels[at:at+24])
		}
		expected := append([]byte(nil), base...)
		if sprite.BlitAt(expected, 24, 0, 0) != nil {
			return nil
		}
		for y := 0; y < 24; y++ {
			at := (origin.Y+y)*320 + origin.X
			if !bytes.Equal(expected[y*24:(y+1)*24], frame[at:at+24]) {
				return nil
			}
		}
		plan.groups = append(plan.groups, group)
		if !fdicon.NativeSourceGroupHasPixels(group) {
			continue
		}
		frames := plan.theme.mapSpritesHD[group]
		if s.cycle >= len(frames) || frames[s.cycle] == nil || frames[s.cycle].Bounds() != image.Rect(0, 0, 144, 144) {
			return nil
		}
		for y := 0; y < 24; y++ {
			for x := 0; x < 24; x++ {
				plan.underlay.SetRGBA(origin.X+x, origin.Y+y, palette[base[y*24+x]])
			}
		}
		plan.units = append(plan.units, modernFacilityRosterUnit{group: group, frame: s.cycle, origin: origin})
	}
	if len(plan.units) == 0 {
		return nil
	}
	return plan
}

func (g *Game) drawModernHDFacilityRosterUnderlay(screen *ebiten.Image, frame []byte) {
	g.facilityRosterHD = nil
	plan := g.prepareModernHDFacilityRoster(frame)
	if plan == nil {
		return
	}
	underlay := ebiten.NewImageFromImage(plan.underlay)
	defer underlay.Dispose()
	options := &ebiten.DrawImageOptions{}
	options.GeoM.Scale(2, 2)
	screen.DrawImage(underlay, options)
	g.facilityRosterHD = plan
}

func (p *modernHDPresentation) drawModernHDFacilityRoster(screen ebiten.FinalScreen, geometry ebiten.GeoM, brightness float64) {
	g := p.Game
	plan := g.facilityRosterHD
	if plan == nil || plan.theme != g.modernStoryPortraits {
		return
	}
	s, ok := g.modernFacilityRosterSource()
	if !ok || s.owner != plan.source.owner || s.mode != plan.source.mode || s.start != plan.source.start ||
		s.selected != plan.source.selected || s.cycle != plan.source.cycle || len(s.ids) != len(plan.source.ids) {
		return
	}
	for row, id := range s.ids {
		unit, exists := g.partyRoster[id]
		if id != plan.source.ids[row] || !exists || !unit.HasMapSelectorKey || unit.MapSelectorKey != plan.groups[row] {
			return
		}
	}
	set := plan.theme
	if set.townCursorGPU == nil {
		set.townCursorGPU = make(map[[2]int]*ebiten.Image)
	}
	for _, unit := range plan.units {
		key := [2]int{unit.group, unit.frame}
		picture := set.townCursorGPU[key]
		if picture == nil {
			picture = ebiten.NewImageFromImage(set.mapSpritesHD[unit.group][unit.frame])
			set.townCursorGPU[key] = picture
		}
		options := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		options.GeoM.Scale(1.0/3, 1.0/3)
		options.GeoM.Translate(float64(unit.origin.X*2), float64(unit.origin.Y*2))
		options.GeoM.Concat(geometry)
		value := float32(math.Max(0, math.Min(1, brightness)))
		options.ColorScale.Scale(value, value, value, 1)
		screen.DrawImage(picture, options)
	}
}
