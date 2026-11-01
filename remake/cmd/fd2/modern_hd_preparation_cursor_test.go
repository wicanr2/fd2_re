package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func verifyPreparationCursorCPU(t *testing.T, g *Game, set *modernStoryPortraitSet) {
	t.Helper()
	saved := *g
	defer func() { *g = saved }()
	g.nativeClassUIJob = nil
	preparationHDTestPhase(g, "roster")
	preparationRosterThirty(g)
	g.interfaceMode = playerInterfaceHD
	g.modernStoryPortraits = set
	initialIdle, initialSelection := g.prepIdleCycle, g.prepSel
	original, _ := g.composeNativePreparationStableFrame()
	cursor := g.prepareModernHDPreparationCursor()
	if cursor == nil || cursor.picture.Bounds() != image.Rect(0, 0, 144, 144) {
		t.Fatal("cursor source unavailable")
	}
	if g.prepareModernHDPreparationCursor() != cursor {
		t.Fatal("cursor material not reused")
	}
	for y := 0; y < 144; y++ {
		for x := 0; x < 144; x++ {
			if got, want := cursor.picture.RGBAAt(x, y), referencePreparationCursorPixel(g, x, y); got != want {
				t.Fatal("corner material source differs", x, y, got, want)
			}
		}
	}
	bank := g.nativePreparationUI.roster.Cursor
	source := bank.Sprites[0]
	for _, fault := range []string{"pixels", "mask", "remap", "size", "palette"} {
		s := source
		palette := g.nativeClassUI.palette
		switch fault {
		case "pixels":
			s.Pixels = append([]byte(nil), source.Pixels...)
			s.Pixels[0] ^= 1
		case "mask":
			s.Mask = append([]byte(nil), source.Mask...)
			s.Mask[0] ^= 255
		case "remap":
			s.RemapMask = append([]byte(nil), source.RemapMask...)
			s.RemapMask[0] ^= 255
		case "size":
			s.Mask = s.Mask[:1]
		case "palette":
			g.nativeClassUI.palette = palette[:1]
		}
		bank.Sprites[0] = s
		if g.prepareModernHDPreparationCursor() != nil {
			t.Fatal("changed source reused cursor", fault)
		}
		bank.Sprites[0] = source
		g.nativeClassUI.palette = palette
	}
	for idle := 0; idle < 4; idle++ {
		g.prepIdleCycle = idle
		for slot := 0; slot < 30; slot++ {
			g.prepSel = slot
			g.modernStoryPortraits = nil
			p := g.prepareModernHDPreparationRoster()
			if p == nil || p.cursor == nil || p.native == nil || len(p.units) != 1 {
				t.Fatal("original art/HD UI missing", idle, slot)
			}
			u := p.units[0]
			selected := g.partyDeploy[g.prepIDs[slot]]
			x, y := 23+28*(slot%10), 100+30*(slot/10)
			if selected {
				y += 3
			}
			if u.Rect != image.Rect(x*6, y*6, (x+24)*6, (y+24)*6) || p.cursorRect != image.Rect(x*6, (104+30*(slot/10))*6, (x+24)*6, (128+30*(slot/10))*6) {
				t.Fatal("cursor or unit position changed", slot)
			}
			cycle, _ := fdicon.NativeFrameIndex(0, false, idle, 0)
			sprite, err := g.nativePreparationUI.roster.Units.SpriteFor(g.partyRoster[g.prepIDs[slot]].MapSelectorKey, 0, cycle)
			if err != nil {
				t.Fatal(err)
			}
			for sy := 0; sy < 24; sy++ {
				for sx := 0; sx < 24; sx++ {
					got := u.ColourImage.RGBAAt(sx*6+2, sy*6+2)
					want := color.RGBA{}
					if sprite.Mask[sy*24+sx] != 0 {
						index := sprite.Pixels[sy*24+sx]
						if !selected {
							index = (index & 7) + 0x18
						}
						want = color.RGBAModel.Convert(g.nativeClassUI.palette[index]).(color.RGBA)
					}
					if got != want {
						t.Fatal("native replay pixels differ", slot, sx, sy, got, want)
					}
				}
			}
			if g.prepareModernHDPreparationRoster().native != p.native {
				t.Fatal("native replay not cached")
			}
			g.interfaceMode = playerInterfaceOriginal
			if g.prepareModernHDPreparationRoster() != nil {
				t.Fatal("original UI retained HD cursor")
			}
			g.modernStoryPortraits = set
			if p := g.prepareModernHDPreparationRoster(); p == nil || p.cursor != nil || p.native != nil {
				t.Fatal("HD art/original UI changed")
			}
			g.interfaceMode = playerInterfaceHD
		}
	}
	g.prepIdleCycle, g.prepSel, g.modernStoryPortraits = initialIdle, initialSelection, set
	now, _ := g.composeNativePreparationStableFrame()
	if !bytes.Equal(original, now) {
		t.Fatal("preparing cursor changed native frame")
	}
	screen := ebiten.NewImage(640, 400)
	defer screen.Dispose()
	for _, phase := range []string{"prompt", "confirmation", "required"} {
		preparationHDTestPhase(g, phase)
		g.Draw(screen)
		if g.preparationRosterHD != nil {
			t.Fatal("cursor leaked to another owner", phase)
		}
	}
}

// Literal independent source lookup for the thin corner geometry.
func referencePreparationCursorPixel(g *Game, x, y int) color.RGBA {
	s := g.nativePreparationUI.roster.Cursor.Sprites[0]
	pixel := func(sx, sy int) color.RGBA {
		return color.RGBAModel.Convert(g.nativeClassUI.palette[s.Pixels[sy*24+sx]]).(color.RGBA)
	}
	if (x < 3 || x >= 141) && (y < 3 || y >= 141) {
		sx, sy := x, y
		if x >= 141 {
			sx = x - 120
		}
		if y >= 141 {
			sy = y - 120
		}
		return pixel(sx, sy)
	}
	if x < 3 && (y < 54 || y >= 78) {
		return pixel(x, 4)
	}
	if x >= 141 && (y < 54 || y >= 78) {
		return pixel(x-120, 4)
	}
	if y < 3 && (x < 60 || x >= 84) {
		return pixel(4, y)
	}
	if y >= 141 && (x < 60 || x >= 84) {
		return pixel(4, y-120)
	}
	return color.RGBA{}
}

func preparationCursorReferenceUnderlay(g *Game, plan *modernPreparationRoster, x, y int, base []byte) [3]float64 {
	b := ((y/3+24)*640 + x/3) * 4
	v := [3]float64{float64(base[b]), float64(base[b+1]), float64(base[b+2])}
	if plan.cursor != nil && image.Pt(x, y).In(plan.cursorRect) {
		c := referencePreparationCursorPixel(g, x-plan.cursorRect.Min.X, y-plan.cursorRect.Min.Y)
		for i, a := range []byte{c.R, c.G, c.B} {
			v[i] = float64(a) + v[i]*float64(255-c.A)/255
		}
	}
	return v
}

func verifyPreparationCursorGPU(p *facilityHDProbe, pixels, base []byte) error {
	g := p.g
	plan := g.preparationRosterHD
	if g.interfaceMode != playerInterfaceHD || !g.prepSelecting || g.prepConfirm || g.prepRecordSlots || g.nativeClassUIJob != nil {
		return nil
	}
	if plan == nil || plan.cursor == nil {
		return fmt.Errorf("case%d cursor missing", p.index)
	}
	total, ink, covered := 0, 0, 0
	for y := plan.cursorRect.Min.Y; y < plan.cursorRect.Max.Y; y++ {
		for x := plan.cursorRect.Min.X; x < plan.cursorRect.Max.X; x++ {
			want := preparationCursorReferenceUnderlay(g, plan, x, y, base)
			c := referencePreparationCursorPixel(g, x-plan.cursorRect.Min.X, y-plan.cursorRect.Min.Y)
			if c.A > 0 {
				ink++
			}
			for _, unit := range plan.units {
				if !image.Pt(x, y).In(unit.Rect) {
					continue
				}
				picture := unit.ColourImage
				var colour color.RGBA
				if picture != nil {
					colour = picture.RGBAAt(x-unit.Rect.Min.X, y-unit.Rect.Min.Y)
				} else {
					colour = color.RGBAModel.Convert(plan.theme.mapSpritesHD[unit.Group][unit.Frame].At(x-unit.Rect.Min.X, y-unit.Rect.Min.Y)).(color.RGBA)
				}
				scales := [3]float64{1, 1, 1}
				if unit.PaletteBand {
					scales = [3]float64{0.55, 0.55, 0.6}
				}
				for i, a := range []byte{colour.R, colour.G, colour.B} {
					want[i] = float64(a)*scales[i] + want[i]*float64(255-colour.A)/255
				}
				if colour.A > 0 && c.A > 0 {
					covered++
				}
			}
			at := ((y+72)*1920 + x) * 4
			for ch, v := range want {
				if delta := float64(pixels[at+ch]) - v; delta < -2 || delta > 2 {
					return fmt.Errorf("case%d cursor/unit ordering %d,%d channel%d delta%f", p.index, x, y, ch, delta)
				}
			}
			total++
		}
	}
	last := p.checks[len(p.checks)-1]
	last["roster_cursor_pixels"], last["roster_cursor_ink"], last["roster_cursor_occluded"] = total, ink, covered
	return nil
}

func appendPreparationCursorCases(p *facilityHDProbe, set *modernStoryPortraitSet) {
	g := p.g
	camp, town, source := g.camp, g.prepTownSource, append([]byte(nil), g.prepPromptSource...)
	for slot := 0; slot < 30; slot++ {
		for style := 0; style < 4; style++ {
			slot, style := slot, style
			if style == 3 && (slot == 0 || slot == 9 || slot == 10 || slot == 19 || slot == 20 || slot == 29) {
				p.captureCases[len(p.cases)] = true
			}
			p.cases = append(p.cases, func() {
				g.camp, g.prepTownSource, g.prepPromptSource = camp, town, source
				g.nativeClassUIJob = nil
				preparationHDTestPhase(g, "roster")
				g.prepLimit, g.prepSel = 30, slot
				g.prepIdleCycle = slot % 3
				for i, id := range g.prepIDs {
					g.partyDeploy[id] = i%2 == 0
				}
				g.modernThemeCache = set
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

func TestModernHDPreparationCursorRuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_HD_PREPARATION_CURSOR_GPU") != "1" {
		t.Skip("dedicated GPU process")
	}
	runModernHDPreparationInterfaceMatrix(t, true, true, true, true)
}

func TestModernHDPreparationCursorSourceAdapter(t *testing.T) {
	bank, err := fdother.LoadSeparatedRangeOverlayBank(separatedAssetPath("sprites/fdother_001_range_overlay"))
	if err != nil {
		t.Fatal(err)
	}
	s := bank.Sprites[0]
	t.Logf("typed source SHA: pixels=%x mask=%x remap=%x", sha256.Sum256(s.Pixels), sha256.Sum256(s.Mask), sha256.Sum256(s.RemapMask))
	if !preparationCursorSourceMatches(s) {
		t.Fatal("typed cursor source differs")
	}
}
