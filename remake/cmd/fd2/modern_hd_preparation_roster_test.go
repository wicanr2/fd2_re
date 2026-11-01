package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
)

func preparationRosterThirty(g *Game) {
	base := g.partyRoster[g.prepIDs[0]]
	g.prepIDs = nil
	g.partyRoster = make(map[int]battle.Unit)
	g.partyDeploy = make(map[int]bool)
	for i := 0; i < 30; i++ {
		id := 200 + i
		unit := base
		unit.MapSelectorKey = []int{0, 1, 2, 50, 118}[i%5]
		unit.HasMapSelectorKey = true
		g.partyRoster[id] = unit
		g.prepIDs = append(g.prepIDs, id)
		g.partyDeploy[id] = i%2 == 0
	}
	g.prepSel, g.prepLimit = 29, 30
}

func verifyPreparationRosterCPU(t *testing.T, g *Game, set *modernStoryPortraitSet) {
	t.Helper()
	saved := *g
	defer func() { *g = saved }()
	g.nativeClassUIJob = nil
	g.modernStoryPortraits = set
	preparationHDTestPhase(g, "roster")
	preparationRosterThirty(g)
	before, _ := json.Marshal(g.partyRoster)
	for idle := 0; idle < 4; idle++ {
		g.prepIdleCycle = idle
		plan := g.prepareModernHDPreparationRoster()
		cycle, err := fdicon.NativeFrameIndex(0, false, idle, 0)
		if err != nil || plan == nil || len(plan.units) != 30 {
			t.Fatal("incomplete thirty-slot plan", idle, err)
		}
		for i, u := range plan.units {
			y := 100 + 30*(i/10)
			if i%2 == 0 {
				y += 3
			}
			x := 23 + 28*(i%10)
			if u.Rect != image.Rect(x*6, y*6, (x+24)*6, (y+24)*6) || u.Frame != cycle ||
				u.Group != []int{0, 1, 2, 50, 118}[i%5] || u.PaletteBand != (i%2 != 0) {
				t.Fatal("native slot/key/selection/cycle differs", i, idle, u)
			}
		}
		full, ok := g.composeNativePreparationStableFrame()
		if !ok {
			t.Fatal("native full missing")
		}
		layer, ok := g.composeNativePreparationStableFrameLayers(false)
		if !ok {
			t.Fatal("layer missing")
		}
		for i, id := range g.prepIDs {
			unit := g.partyRoster[id]
			sprite, err := g.nativePreparationUI.roster.Units.SpriteFor(unit.MapSelectorKey, 0, cycle)
			if err != nil {
				t.Fatal(err)
			}
			x, y := 23+28*(i%10), 100+30*(i/10)
			if g.partyDeploy[id] {
				y += 3
				err = sprite.BlitAt(layer, 320, x, y)
			} else {
				err = sprite.BlitPaletteBand(layer, 320, x, y)
			}
			if err != nil {
				t.Fatal(err)
			}
		}
		if !bytes.Equal(full, layer) {
			t.Fatal("native reconstruction changed")
		}
	}
	frames := set.mapSpritesHD[0]
	for _, fault := range []string{"missing", "geometry", "raw-key", "no-key", "cursor", "confirm", "slots", "job", "original"} {
		state := *g
		switch fault {
		case "missing":
			delete(set.mapSpritesHD, 0)
		case "geometry":
			set.mapSpritesHD[0] = []image.Image{image.NewNRGBA(image.Rect(0, 0, 72, 72))}
		case "raw-key", "no-key":
			u := g.partyRoster[200]
			if fault == "raw-key" {
				u.MapSelectorKey = 999
			} else {
				u.HasMapSelectorKey = false
			}
			g.partyRoster[200] = u
		case "cursor":
			g.prepSel = 30
		case "confirm":
			g.prepConfirm = true
		case "slots":
			g.prepRecordSlots = true
		case "job":
			g.nativeClassUIJob = &nativeClassUIJob{}
		case "original":
			g.modernStoryPortraits = nil
		}
		if g.prepareModernHDPreparationRoster() != nil {
			t.Fatal("unsafe plan admitted", fault)
		}
		*g = state
		u := g.partyRoster[200]
		u.MapSelectorKey = 0
		u.HasMapSelectorKey = true
		g.partyRoster[200] = u
		set.mapSpritesHD[0] = frames
	}
	after, _ := json.Marshal(g.partyRoster)
	if !bytes.Equal(before, after) {
		t.Fatal("plan changed party")
	}
	screen := ebiten.NewImage(640, 400)
	defer screen.Dispose()
	g.interfaceMode = playerInterfaceHD
	g.Draw(screen)
	if g.preparationRosterHD == nil {
		t.Fatal("HD roster not published")
	}
	for _, phase := range []string{"prompt", "confirmation", "required"} {
		preparationHDTestPhase(g, phase)
		g.Draw(screen)
		if g.preparationRosterHD != nil {
			t.Fatal("roster leaked to another owner", phase)
		}
	}
}

// Native pixels remain in the offscreen underlay. Compare every source pixel
// against its premultiplied-alpha Over result, including the cursor underneath.
func verifyPreparationRosterGPU(p *facilityHDProbe, pixels, base []byte) error {
	g := p.g
	plan := g.preparationRosterHD
	if plan == nil {
		return nil
	}
	total, core, transparent, band := 0, 0, 0, 0
	for _, unit := range plan.units {
		picture := plan.theme.mapSpritesHD[unit.Group][unit.Frame]
		for sy := 0; sy < 144; sy++ {
			for sx := 0; sx < 144; sx++ {
				x, y := unit.Rect.Min.X+sx, unit.Rect.Min.Y+sy
				at := ((y+72)*1920 + x) * 4
				b := ((y/3+24)*640 + x/3) * 4
				if base[b+3] != 255 {
					return fmt.Errorf("roster panel underlay not opaque")
				}
				c := color.RGBAModel.Convert(picture.At(sx, sy)).(color.RGBA)
				values := []float64{float64(c.R), float64(c.G), float64(c.B)}
				if unit.PaletteBand {
					values[0] *= 0.55
					values[1] *= 0.55
					values[2] *= 0.6
					band++
				}
				for ch, v := range values {
					want := v + float64(base[b+ch])*float64(255-c.A)/255
					delta := float64(pixels[at+ch]) - want
					if delta < -2 || delta > 2 {
						return fmt.Errorf("case%d roster %d/%d pixel %d,%d channel%d delta%f", p.index, unit.Group, unit.Frame, sx, sy, ch, delta)
					}
				}
				total++
				if c.A >= 250 {
					core++
				}
				if c.A == 0 {
					transparent++
				}
			}
		}
	}
	if total == 0 || core < 100 || transparent < 100 {
		return fmt.Errorf("too few roster pixels")
	}
	last := p.checks[len(p.checks)-1]
	last["roster_pixels"], last["roster_cores"], last["roster_transparent"], last["roster_band_pixels"] = total, core, transparent, band
	return nil
}

func TestModernHDPreparationRosterRuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_HD_PREPARATION_ROSTER_GPU") != "1" {
		t.Skip("dedicated GPU process")
	}
	runModernHDPreparationRosterMatrix(t, false)
}

func runModernHDPreparationRosterMatrix(t *testing.T, status bool) {
	runModernHDPreparationRosterAndCountsMatrix(t, status, false)
}

func runModernHDPreparationRosterAndCountsMatrix(t *testing.T, status, counts bool) {
	runModernHDPreparationInterfaceMatrix(t, status, counts, false)
}

func runModernHDPreparationInterfaceMatrix(t *testing.T, status, counts, icons bool) {
	set := facilityHDTestSet(t)
	g := facilityHDTestGame(t, set)
	var err error
	g.nativePreparationUI, err = loadNativePreparationUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	out := os.Getenv("FD2_TEST_FACILITY_OUT")
	if out == "" {
		t.Fatal("explicit output required")
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	p := &facilityHDProbe{t: t, g: g, out: out, presentation: &modernHDPresentation{Game: g, toolbar: true},
		captured: make(map[*modernFacilityScene]bool), captureCases: make(map[int]bool), verifyScene: verifyPreparationHDScene}
	if status || counts {
		p.verifyScene = func(p *facilityHDProbe, pixels []byte, offscreen *ebiten.Image) error {
			if err := verifyPreparationHDScene(p, pixels, offscreen); err != nil {
				return err
			}
			if err := verifyPreparationStatusGPU(p, pixels); err != nil {
				return err
			}
			if counts {
				return verifyPreparationCountsGPU(p, pixels)
			}
			return nil
		}
	}
	originalRoster, originalDeploy, originalOrder := g.partyRoster, g.partyDeploy, g.partyJoinOrder
	for variant := 0; variant < 4; variant++ {
		g.partyRoster, g.partyDeploy, g.partyJoinOrder = originalRoster, originalDeploy, originalOrder
		townHDDepartureTestOwner(t, g, variant%3)
		g.nativeTownDeparture = nil
		if variant == 3 {
			g.camp.Node().Cancel = ""
		}
		g.setupPreparation(g.camp.Node())
		g.nativeClassUIJob = nil
		camp, town, source := g.camp, g.prepTownSource, append([]byte(nil), g.prepPromptSource...)
		preparationRosterThirty(g)
		for cycle := 0; cycle < 3; cycle++ {
			for style := 0; style < 4; style++ {
				for invert := 0; invert < 2; invert++ {
					style, cycle, invert := style, cycle, invert
					if cycle == 0 && invert == 0 {
						p.captureCases[len(p.cases)] = true
					}
					p.cases = append(p.cases, func() {
						g.camp, g.prepTownSource, g.prepPromptSource = camp, town, source
						g.nativeClassUIJob = nil
						preparationHDTestPhase(g, "roster")
						g.prepIdleCycle = cycle
						g.prepSel = 29 - invert*20
						for i, id := range g.prepIDs {
							g.partyDeploy[id] = (i+invert)%2 == 0
						}
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
	}
	if status {
		appendPreparationStatusCases(p, set)
	}
	if counts {
		appendPreparationCountCases(p, set)
	}
	if icons {
		appendPreparationLegacyCountCases(p, set)
	}
	ebiten.SetWindowSize(1920, 1272)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil {
		t.Fatal(p.err)
	}
	wantCases := 96
	if status {
		wantCases = 160
	}
	if counts {
		wantCases = 208
	}
	if icons {
		wantCases = 212
	}
	if len(p.checks) != wantCases {
		t.Fatal("incomplete matrix", len(p.checks))
	}
	receiptStatus, scope := "PASS-PREPARATION-ROSTER-GPU", "96 fixture Game.Draw cases, 30 slots, 3 town variants and standalone black, 3 cycles, 4 styles, 2 selection patterns; no new original E2"
	if status {
		receiptStatus, scope = "PASS-PREPARATION-STATUS-GPU", "160 fixture Game.Draw cases: 96 roster cases + 64 native base/flag icon and HP/MP cases; no new original E2"
	}
	if counts {
		receiptStatus, scope = "PASS-PREPARATION-COUNTS-GPU", "208 fixture Game.Draw cases: 160 status cases + 48 quota/remaining cases; no new original E2"
	}
	if icons {
		receiptStatus, scope = "PASS-PREPARATION-COUNT-ICON-GPU", "212 fixture Game.Draw cases, 208 current optional icon cases and 4 legacy absent-role cases; no identity binding or new original E2"
	}
	raw, _ := json.MarshalIndent(map[string]any{"status": receiptStatus, "scope": scope, "checks": p.checks}, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
