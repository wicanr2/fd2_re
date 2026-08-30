package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func (g *Game) composeNativeChurchRosterFrame() ([]byte, bool) {
	a := g.nativeClassUI
	if a == nil ||
		(g.churchMode != "status_roster" &&
			g.churchMode != "transfer_source" &&
			g.churchMode != "transfer_dest") ||
		len(g.churchIDs) == 0 {
		return nil, false
	}
	start, visible := campaign.NativeTwoColumnWindow(
		len(g.churchIDs), g.churchSel, g.churchRosterStart,
	)
	if visible == 0 {
		return nil, false
	}
	g.churchRosterStart = start
	source, ok := g.composeNativeChurchSourceFrame()
	if !ok {
		return nil, false
	}
	rows := make([]campaign.NativeRosterRow, 0, visible)
	identities := make([]int, 0, visible)
	for i := 0; i < visible; i++ {
		unit, ok := g.partyRoster[g.churchIDs[start+i]]
		if !ok || !unit.HasNativeIdentity || !unit.HasMapSelectorKey {
			return nil, false
		}
		sprite, err := a.units.SpriteFor(unit.MapSelectorKey, 0, 0)
		if err != nil {
			return nil, false
		}
		rows = append(rows, campaign.NativeRosterRow{
			Sprite: sprite, NameTextIndex: unit.NativeIdentity + 1,
		})
		identities = append(identities, unit.NativeIdentity)
	}
	var frame []byte
	var err error
	if g.localeID != "" && g.localeID != "zh-Hant" {
		frame, err = g.composeLocalizedNativeRoster(
			source, a.panel, rows, identities, g.churchSel-start,
		)
	} else {
		frame, err = campaign.ComposeNativeRosterFrame(
			source, a.panel, rows, g.churchSel-start, a.strings, a.font,
		)
	}
	return frame, err == nil
}

func (g *Game) drawNativeChurchRoster(screen *ebiten.Image) bool {
	frame, ok := g.composeNativeChurchRosterFrame()
	if !ok {
		return false
	}
	g.presentNativeClassFrame(screen, frame)
	return true
}

func (g *Game) beginNativeChurchRosterOpening() bool {
	final, ok := g.composeNativeChurchRosterFrame()
	if !ok {
		return false
	}
	source, ok := g.composeNativeChurchSourceFrame()
	if !ok {
		return false
	}
	frames, err := campaign.NativeClassListOpeningFrames(source, final)
	if err != nil || len(frames) != 6 {
		return false
	}
	g.nativeClassUIJob = &nativeClassUIJob{frames: frames}
	return true
}

func (g *Game) beginNativeChurchRosterClosing(after func()) bool {
	final, ok := g.composeNativeChurchRosterFrame()
	if !ok {
		return false
	}
	source, ok := g.composeNativeChurchSourceFrame()
	if !ok {
		return false
	}
	frames, err := campaign.NativeClassListClosingFrames(source, final)
	if err != nil || len(frames) != 5 {
		return false
	}
	g.nativeClassUIJob = &nativeClassUIJob{frames: frames, restore: source, after: after}
	return true
}
