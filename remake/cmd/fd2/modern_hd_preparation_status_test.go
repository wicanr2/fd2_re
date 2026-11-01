package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func verifyPreparationStatusCPU(t *testing.T, g *Game, set *modernStoryPortraitSet) {
	t.Helper()
	saved := *g
	defer func() { *g = saved }()
	g.partyRoster = make(map[int]battle.Unit)
	for id, u := range saved.partyRoster {
		g.partyRoster[id] = u
	}
	g.nativeClassUIJob = nil
	preparationHDTestPhase(g, "roster")
	g.modernStoryPortraits, g.modernThemeCache, g.interfaceMode = set, set, playerInterfaceHD
	screen := ebiten.NewImage(640, 400)
	defer screen.Dispose()
	before, ok := g.composeNativePreparationStableFrame()
	if !ok {
		t.Fatal("native source missing")
	}
	g.Draw(screen)
	h := g.prepareModernHDPreparationStatus(g.nativePreparationUI)
	if h == nil || h != g.prepareModernHDPreparationStatus(g.nativePreparationUI) {
		t.Fatal("status source/cache missing")
	}
	if h.values.picture.Bounds() != image.Rect(0, 0, 223*6, 86*6) {
		t.Fatal("wrong status rectangle")
	}
	after, ok := g.composeNativePreparationStableFrame()
	if !ok || !bytes.Equal(before, after) {
		t.Fatal("status material changed native pixels")
	}
	id := g.prepIDs[g.prepSel]
	unit := g.partyRoster[id]
	changed := unit
	changed.HP--
	g.partyRoster[id] = changed
	if next := g.prepareModernHDPreparationStatus(g.nativePreparationUI); next == nil || next == h {
		t.Fatal("live record retained stale bars")
	}
	g.partyRoster[id] = unit
	if next := g.prepareModernHDPreparationStatus(g.nativePreparationUI); next == nil || next.key != h.key {
		t.Fatal("record restoration failed")
	}
	for _, fault := range []string{"missing-icon", "changed-icon", "panel-source", "bar-cell", "palette", "font", "confirm", "slots", "job", "original"} {
		state := *g
		ui := *g.nativePreparationUI
		ui.status.RawCells = make(map[int]fdother.RawCell)
		for i, c := range g.nativePreparationUI.status.RawCells {
			ui.status.RawCells[i] = c
		}
		badSet := *set
		badSet.uiPanelIcons = make(map[int]*modernUIPanelIcon)
		for i, c := range set.uiPanelIcons {
			badSet.uiPanelIcons[i] = c
		}
		g.nativePreparationUI = &ui
		g.modernStoryPortraits = &badSet
		g.modernThemeCache = &badSet
		switch fault {
		case "missing-icon":
			delete(badSet.uiPanelIcons, 53)
		case "changed-icon":
			icon := *badSet.uiPanelIcons[53]
			icon.digest[0] ^= 255
			badSet.uiPanelIcons[53] = &icon
		case "panel-source":
			ui.status.ItemPanelUpper.Pixels = append([]byte(nil), ui.status.ItemPanelUpper.Pixels...)
			ui.status.ItemPanelUpper.Pixels[0] ^= 255
		case "bar-cell":
			cell := ui.status.RawCells[24]
			cell.Width = 2
			ui.status.RawCells[24] = cell
		case "palette":
			c := *g.nativeClassUI
			c.palette = nil
			g.nativeClassUI = &c
		case "font":
			g.font = nil
		case "confirm":
			g.prepConfirm = true
		case "slots":
			g.prepRecordSlots = true
		case "job":
			g.nativeClassUIJob = &nativeClassUIJob{}
		case "original":
			g.modernHDText = nil
		}
		if g.prepareModernHDPreparationStatus(g.nativePreparationUI) != nil {
			t.Fatal("invalid or wrong owner admitted", fault)
		}
		*g = state
	}
	for _, art := range []*modernStoryPortraitSet{nil, set} {
		g.modernStoryPortraits = art
		g.interfaceMode = playerInterfaceHD
		g.Draw(screen)
		if g.prepareModernHDPreparationStatus(g.nativePreparationUI) == nil {
			t.Fatal("F2 changed interface icon availability")
		}
	}
	for _, phase := range []string{"prompt", "confirmation", "required"} {
		preparationHDTestPhase(g, phase)
		g.Draw(screen)
		for _, panel := range g.modernHDText.panels {
			if panel.status {
				t.Fatal("status card leaked into modal owner", phase)
			}
		}
	}
}

func TestModernHDPreparationStatusRuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_HD_PREPARATION_STATUS_GPU") != "1" {
		t.Skip("dedicated GPU process")
	}
	runModernHDPreparationRosterMatrix(t, true)
}

func appendPreparationStatusCases(p *facilityHDProbe, set *modernStoryPortraitSet) {
	g := p.g
	camp, town, source := g.camp, g.prepTownSource, append([]byte(nil), g.prepPromptSource...)
	id := g.prepIDs[0]
	base := g.partyRoster[id]
	for flags := 0; flags < 16; flags++ {
		for style := 0; style < 4; style++ {
			flags, style := flags, style
			if style == 3 && (flags < 2 || flags > 13) {
				p.captureCases[len(p.cases)] = true
			}
			p.cases = append(p.cases, func() {
				g.camp, g.prepTownSource, g.prepPromptSource = camp, town, source
				g.nativeClassUIJob = nil
				preparationHDTestPhase(g, "roster")
				g.prepSel = 0
				unit := base
				unit.NativeRecordByte6 = byte(flags & 1)
				for i := 0; i < 3; i++ {
					unit.NativeTransient[3+i] = byte(flags & (2 << i))
				}
				unit.MaxHP, unit.MaxMP = 100, 50
				switch flags % 4 {
				case 0:
					unit.HP, unit.MP, unit.MaxHP, unit.MaxMP = 0, 0, 0, 0
				case 1:
					unit.HP, unit.MP = 1, 1
				case 2:
					unit.HP, unit.MP = 50, 25
				case 3:
					unit.HP, unit.MP = 100, 50
				}
				g.partyRoster[id] = unit
				g.modernStoryPortraits = nil
				if style >= 2 {
					g.modernStoryPortraits = set
				}
				g.interfaceMode = playerInterfaceOriginal
				if style%2 == 1 {
					g.interfaceMode = playerInterfaceHD
				}
			})
		}
	}
}

func verifyPreparationStatusGPU(p *facilityHDProbe, pixels []byte) error {
	g := p.g
	if g.interfaceMode == playerInterfaceOriginal {
		if g.modernHDText != nil {
			for _, panel := range g.modernHDText.panels {
				if panel.status {
					return fmt.Errorf("status HD in original UI")
				}
			}
		}
		return nil
	}
	h := g.prepareModernHDPreparationStatus(g.nativePreparationUI)
	if h == nil {
		return fmt.Errorf("case%d HD status missing", p.index)
	}
	unit := g.partyRoster[g.prepIDs[g.prepSel]]
	record, err := battle.NativeItemPanelRecordForUnit(&unit)
	if err != nil {
		return err
	}
	check := func(x, y int, want color.RGBA) error {
		at := ((y+72)*1920 + x) * 4
		if !bytes.Equal(pixels[at:at+4], []byte{want.R, want.G, want.B, want.A}) {
			return fmt.Errorf("case%d status pixel %d,%d: %v != %v", p.index, x, y, pixels[at:at+4], want)
		}
		return nil
	}
	edge, icons, bars := 0, 0, 0
	colors := nativeDialogueChromeSource(g.nativePreparationUI.dialogue, g.nativeClassUI.palette)
	for y := 0; y < 86*6; y++ {
		for x := 0; x < 223*6; x++ {
			if c, ok := itemPanelChromePixel(colors, x, y, 223*6, 86*6); ok {
				if err := check(92*6+x, 7*6+y, color.RGBA{c[0], c[1], c[2], c[3]}); err != nil {
					return err
				}
				edge++
			}
		}
	}
	type iconCall struct{ index, x, y int }
	index := 53
	if record[6] == 0 {
		index = 54
	}
	calls := []iconCall{{index, 101, 30}}
	for i := 0; i < 3; i++ {
		if record[37+i] != 0 {
			calls = append(calls, iconCall{55 + i, 194 + 35*i, 68})
		}
	}
	set := g.interfaceAssetTheme()
	for _, call := range calls {
		picture := set.uiPanelIcons[call.index].picture
		for y := 0; y < picture.Bounds().Dy(); y++ {
			for x := 0; x < picture.Bounds().Dx(); x++ {
				if err := check(call.x*6+x, call.y*6+y, picture.RGBAAt(x, y)); err != nil {
					return err
				}
				icons++
			}
		}
	}
	pal := func(index byte) color.RGBA {
		return color.RGBAModel.Convert(g.nativeClassUI.palette[index]).(color.RGBA)
	}
	upper := g.nativePreparationUI.status.ItemPanelUpper
	bg := pal(upper.Pixels[70*223+100])
	outline := pal(upper.Pixels[224])
	for i, ny := range []int{33, 52} {
		current := int(int16(binary.LittleEndian.Uint16(record[64+4*i:])))
		maximum := int(int16(binary.LittleEndian.Uint16(record[66+4*i:])))
		width := 0
		if current != 0 && maximum != 0 {
			n := 101*current/maximum + 1
			if n < 1 {
				n = 1
			}
			if n > 102 {
				n = 102
			}
			width = (n + 1) * 6
		}
		fill := pal(g.nativePreparationUI.status.RawCells[24+3*i].Pixels[1])
		empty := pal(g.nativePreparationUI.status.RawCells[29].Pixels[1])
		for y := (ny - 1) * 6; y < (ny+6)*6; y++ {
			for x := 197 * 6; x < 302*6; x++ {
				want := bg
				ix, iy := x-198*6, y-ny*6
				if ix >= 0 && ix < 103*6 && iy >= 0 && iy < 5*6 {
					want = empty
					if ix < width {
						want = fill
					}
					if ix == 0 || ix == 103*6-1 || iy == 0 || iy == 5*6-1 {
						want = outline
					}
				}
				if err := check(x, y, want); err != nil {
					return err
				}
				bars++
			}
		}
	}
	last := p.checks[len(p.checks)-1]
	last["status_edge_pixels"], last["status_icon_pixels"], last["status_bar_pixels"] = edge, icons, bars
	return nil
}
