package main

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

type nativeTownUIAssets struct {
	scene *campaign.NativeTownAssets
}

// A caller owns its resolved source, independently of later roster changes.
// The decoded assets are immutable; the selected pulse array is a value copy.
type nativeTownFrameSource struct {
	scene                            campaign.NativeTownAssets
	strings                          *fdtxt.Strings
	font                             *fdtxt.Font
	variant, selection, pulse, group int
	indexed                          []byte
}

func (g *Game) snapshotNativeTownSource() *nativeTownFrameSource {
	if g.camp == nil || g.nativeTownUI == nil || g.nativeTownUI.scene == nil || g.nativeClassUI == nil {
		return nil
	}
	n := g.camp.Node()
	if n == nil || n.Type != "town" || n.NativeTownVariant == nil {
		return nil
	}
	d := &nativeTownFrameSource{scene: *g.nativeTownUI.scene, strings: g.nativeClassUI.strings,
		font: g.nativeClassUI.font, variant: *n.NativeTownVariant, selection: g.campSel, pulse: g.nativeTownUIPulse}
	if _, err := campaign.NativeTownCursorGeometryFor(d.variant, d.selection, d.pulse); err != nil {
		return nil
	}
	if len(g.partyJoinOrder) > 0 {
		unit, ok := g.partyRoster[g.partyJoinOrder[0]]
		if !ok {
			return nil
		}
		if unit.HasMapSelectorKey {
			if g.nativeClassUI.units == nil {
				return nil
			}
			d.group = unit.MapSelectorKey
			for cycle := range d.scene.Pulse {
				sprite, err := g.nativeClassUI.units.SpriteFor(d.group, 0, cycle)
				if err != nil {
					return nil
				}
				d.scene.Pulse[cycle] = sprite
			}
		}
	}
	return d
}

func (d *nativeTownFrameSource) compose(fill *byte, cursor bool, font *fdtxt.Font) ([]byte, bool) {
	if d == nil {
		return nil, false
	}
	scene := d.scene
	if fill != nil {
		scene.Backgrounds[d.variant] = bytes.Repeat([]byte{*fill}, 320*200)
	}
	compose := campaign.ComposeNativeTownFrame
	if !cursor {
		compose = campaign.ComposeNativeTownFrameWithoutCursor
	}
	frame, err := compose(&scene, d.strings, font, d.variant, d.selection, d.pulse)
	return frame, err == nil
}

func (g *Game) captureNativeTownCaller() {
	g.prepPromptSource, g.prepTownSource = nil, nil
	d := g.snapshotNativeTownSource()
	if d == nil {
		return
	}
	frame, ok := d.compose(nil, true, d.font)
	if !ok {
		return
	}
	d.indexed = append([]byte(nil), frame...)
	g.prepPromptSource, g.prepTownSource = frame, d
}

func parseNativeTownShotState(spec string) (selection, pulse int, ok bool) {
	parts := strings.Split(spec, ",")
	if len(parts) != 2 {
		return 0, 0, false
	}
	selection, err := strconv.Atoi(parts[0])
	if err != nil || selection < 0 || selection > 5 {
		return 0, 0, false
	}
	pulse, err = strconv.Atoi(parts[1])
	if err != nil || pulse < 0 || pulse > 3 {
		return 0, 0, false
	}
	return selection, pulse, true
}

// setNativeTownShotState is an explicit screenshot-only oracle hook. It
// refuses to create town state outside a native town node and resets the
// clock so Update cannot advance the requested pulse before the next Draw.
func (g *Game) setNativeTownShotState(selection, pulse int) bool {
	if selection < 0 || selection > 5 || pulse < 0 || pulse > 3 ||
		g.camp == nil || g.nativeTownUI == nil {
		return false
	}
	n := g.camp.Node()
	if n == nil || n.Type != "town" || n.NativeTownVariant == nil {
		return false
	}
	g.campSel = selection
	g.resetNativeTownUIPulse()
	g.nativeTownUIPulse = pulse
	return true
}

// nativeTownMoveSelection preserves 0x2cf01..0x2cf65: right decrements and
// wraps normal 0 to 4; left increments and wraps every value above 4 to 0.
// This also defines how the revealed hidden selection 5 returns to the five
// normal facilities.
func nativeTownMoveSelection(selection, delta int) (int, bool) {
	if selection < 0 || selection > 5 || (delta != -1 && delta != 1) {
		return 0, false
	}
	selection += delta
	if selection < 0 {
		selection = 4
	}
	if selection > 4 {
		selection = 0
	}
	return selection, true
}

// moveNativeTownSelection changes only the selector. The original
// 0x2ce7a/0x2ceac branches do not write the shared pulse counter at 0x54133.
func (g *Game) moveNativeTownSelection(delta int) bool {
	selection, ok := nativeTownMoveSelection(g.campSel, delta)
	if !ok {
		return false
	}
	g.campSel = selection
	return true
}

// revealNativeTownSecret changes only the selector. The original 0x2cef7
// branch writes 5 to 0x5412b and does not touch pulse counter 0x54133.
func (g *Game) revealNativeTownSecret(scanCode int) bool {
	if g.camp == nil ||
		!g.camp.MatchNativeTownSecret(g.campSel, scanCode) {
		return false
	}
	g.campSel = 5
	return true
}

func loadNativeTownUIAssets() (*nativeTownUIAssets, error) {
	scene, err := campaign.LoadNativeTownAssets(separatedAssetPath(""))
	if err != nil {
		return nil, err
	}
	return &nativeTownUIAssets{scene: scene}, nil
}

func (g *Game) resetNativeTownUIPulse() {
	g.nativeTownUIClock.Reset()
	g.nativeTownUIPulse = 0
	g.nativeTownUILastTick = 0
	g.nativeTownUIHasTick = false
}

// 0x2d1b5 advances the four-state counter after a signed BIOS low-word delta
// of four. Counter 3 is rendered with FDICON sprite 1 by the compositor.
func (g *Game) stepNativeTownUIPulseTick(rawTick int) {
	if !g.nativeTownUIHasTick {
		g.nativeTownUILastTick = rawTick
		g.nativeTownUIHasTick = true
		return
	}
	delta := int16(uint16(rawTick) - uint16(g.nativeTownUILastTick))
	if delta < 4 {
		return
	}
	g.nativeTownUILastTick = rawTick
	g.nativeTownUIPulse = (g.nativeTownUIPulse + 1) & 3
}

func (g *Game) stepNativeTownUILifecycle(nowTick int) {
	if g.camp == nil || g.nativeTownUI == nil {
		return
	}
	n := g.camp.Node()
	if n == nil || n.Type != "town" || n.NativeTownVariant == nil {
		return
	}
	g.stepNativeTownUIPulseTick(nowTick)
}

func (g *Game) composeNativeTownFrame() ([]byte, bool) {
	return g.composeNativeTownFrameWithBackground(nil)
}

func (g *Game) composeNativeTownFrameWithBackground(fill *byte) ([]byte, bool) {
	return g.composeNativeTownFrameLayers(fill, true)
}

func (g *Game) composeNativeTownFrameLayers(fill *byte, cursor bool) ([]byte, bool) {
	d := g.snapshotNativeTownSource()
	if d == nil {
		return nil, false
	}
	return d.compose(fill, cursor, d.font)
}

func (g *Game) drawNativeTown(screen *ebiten.Image) bool {
	if g.drawModernHDTown(screen) {
		return true
	}
	frame, ok := g.composeNativeTownFrame()
	if !ok {
		return false
	}
	g.presentNativeClassFrame(screen, frame)
	return true
}
