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
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func minimalFacilityRosterSet(t *testing.T) *modernStoryPortraitSet {
	t.Helper()
	raw, err := os.ReadFile(os.Getenv("FD2_TEST_FACILITY_CATALOG"))
	if err != nil {
		t.Fatal(err)
	}
	var catalog modernThemeCatalog
	if err := json.Unmarshal(raw, &catalog); err != nil {
		t.Fatal(err)
	}
	var entries []modernThemeAsset
	for _, a := range catalog.Assets {
		if a.Role == "map_sprite_set" || a.Role == "map_sprite_hd_set" ||
			(a.Role == "story_portrait_frame" || a.Role == "story_portrait_hd_frame") && a.SpeakerID == 0 {
			entries = append(entries, a)
		}
	}
	catalog.Assets = entries
	data, _ := json.Marshal(catalog)
	path := filepath.Join(t.TempDir(), "catalog.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	set, err := loadModernStoryPortraitSet(path, os.Getenv("FD2_TEST_FACILITY_PACK"))
	if err != nil {
		t.Fatal(err)
	}
	if len(set.mapSpritesHD) != 125 {
		t.Fatal("actual approved Q collection absent")
	}
	return set
}

var facilityRosterShopModes = []string{"recipient_consumable", "recipient_equipment", "sell_roster", "equip_roster", "transfer_source", "transfer_dest"}
var facilityRosterChurchModes = []string{"class", "status_roster", "transfer_source", "transfer_dest", "revive"}

func facilityRosterTestState(g *Game, owner, mode string, variant, cycle int) {
	if owner == "shop" {
		shopDialogueTestState(g, variant, mode, 0, 0)
		g.nativeShopRecipientCycle, g.nativeShopSellRosterCycle = cycle, cycle
	} else {
		facilityDialogueTestOwner(g, "church", mode, 0, 0)
		if mode != "class" {
			g.churchIDs = append([]int(nil), g.partyJoinOrder...)
		}
	}
	// The fixture names stay typed native IDs; only the raw map source keys vary.
	for id, key := range []int{0, 2, 3, 9, 46, 50} {
		unit := g.partyRoster[id]
		unit.MapSelectorKey, unit.HasMapSelectorKey = key, true
		g.partyRoster[id] = unit
	}
}

func facilityRosterCompose(g *Game) ([]byte, bool) {
	if g.camp.Node().Type == "shop" {
		return shopDialogueCompose(g)
	}
	return facilityDialogueCompose(g)
}

func TestModernHDFacilityRosterSourceAndFallback(t *testing.T) {
	set := minimalFacilityRosterSet(t)
	g := facilityDialogueTestGame(t, set)
	for _, owner := range []string{"shop", "church"} {
		modes, variants := facilityRosterShopModes, []int{1, 3, 5}
		if owner == "church" {
			modes, variants = facilityRosterChurchModes, []int{0}
		}
		for _, variant := range variants {
			for _, mode := range modes {
				for cycle := 0; cycle < 3; cycle++ {
					facilityRosterTestState(g, owner, mode, variant, cycle)
					frame, ok := facilityRosterCompose(g)
					if !ok {
						t.Fatal("actual native compositor unavailable", owner, mode, variant)
					}
					before := append([]byte(nil), frame...)
					party, _ := json.Marshal(g.partyRoster)
					plan := g.prepareModernHDFacilityRoster(frame)
					if plan == nil {
						t.Fatal("actual complete source refused", owner, mode, variant, cycle)
					}
					wantedCycle := 0
					if mode == "recipient_consumable" || mode == "recipient_equipment" || mode == "sell_roster" {
						wantedCycle = cycle
					}
					if len(plan.units) != len(plan.source.ids) {
						t.Fatal("visible typed roster count changed")
					}
					for row, unit := range plan.units {
						id := plan.source.ids[row]
						if unit.group != g.partyRoster[id].MapSelectorKey || unit.frame != wantedCycle ||
							unit.origin != image.Pt(14+132*(row%plan.source.cols), 117+26*(row/plan.source.cols)) {
							t.Fatal("raw key, cycle or native origin changed")
						}
					}
					for _, hdUI := range []bool{false, true} {
						g.interfaceMode = playerInterfaceOriginal
						if hdUI {
							g.interfaceMode = playerInterfaceHD
						}
						if g.prepareModernHDFacilityRoster(frame) == nil {
							t.Fatal("Q art followed the interface switch")
						}
						g.modernStoryPortraits = nil
						if g.prepareModernHDFacilityRoster(frame) != nil {
							t.Fatal("original art retained Q replacement")
						}
						g.modernStoryPortraits = set
					}
					covered := append([]byte(nil), frame...)
					covered[117*320+14] ^= 1
					if g.prepareModernHDFacilityRoster(covered) != nil {
						t.Fatal("covered native sprite admitted")
					}
					g.nativeShopUIJob = &nativeClassUIJob{}
					if g.prepareModernHDFacilityRoster(frame) != nil {
						t.Fatal("foreign animation job admitted")
					}
					g.nativeShopUIJob = nil
					after, _ := json.Marshal(g.partyRoster)
					if !bytes.Equal(frame, before) || !bytes.Equal(party, after) {
						t.Fatal("native source or typed party changed")
					}
				}
			}
		}
	}
	facilityRosterTestState(g, "shop", "sell_roster", 1, 0)
	frame, ok := facilityRosterCompose(g)
	if !ok {
		t.Fatal("actual sell roster absent")
	}
	unit := g.partyRoster[0]
	missing := unit
	missing.HasMapSelectorKey = false
	g.partyRoster[0] = missing
	if g.prepareModernHDFacilityRoster(frame) != nil {
		t.Fatal("missing raw selector admitted")
	}
	g.partyRoster[0] = unit
	frames := set.mapSpritesHD[unit.MapSelectorKey]
	delete(set.mapSpritesHD, unit.MapSelectorKey)
	if g.prepareModernHDFacilityRoster(frame) != nil {
		t.Fatal("missing HD group admitted")
	}
	set.mapSpritesHD[unit.MapSelectorKey] = frames
	for _, flag := range []string{"help", "transient", "black", "white"} {
		g.helpVisible, g.transientUI = flag == "help", flag == "transient"
		g.nativeFullDACBlack, g.nativeFullDACWhite = flag == "black", flag == "white"
		if g.prepareModernHDFacilityRoster(frame) != nil {
			t.Fatal("covered owner retained roster", flag)
		}
	}
	g.helpVisible, g.transientUI, g.nativeFullDACBlack, g.nativeFullDACWhite = false, false, false, false
	g.camp = campaign.NewRunner(&campaign.Campaign{Start: "hotel", Nodes: map[string]*campaign.Node{"hotel": {Type: "hotel"}}})
	if g.prepareModernHDFacilityRoster(frame) != nil {
		t.Fatal("foreign owner retained roster")
	}
}

func facilityRosterTestEightParty(g *Game) {
	for id := 6; id < 8; id++ {
		unit := g.partyRoster[id-2]
		unit.NativeIdentity, unit.HasNativeIdentity = id, true
		g.partyRoster[id] = unit
	}
	g.partyJoinOrder = []int{0, 1, 2, 3, 4, 5, 6, 7}
}

func facilityRosterTestScroll(g *Game, owner, mode string) {
	if owner == "shop" {
		if mode == "recipient_equipment" {
			g.shopRecipientSel, g.nativeShopRecipientStart = 5, 3
		} else {
			g.shopSellUnitSel, g.nativeShopSellRosterTop = 7, 2
		}
	} else if mode == "revive" {
		g.churchSel, g.churchVerticalStart = 7, 5
	} else {
		g.churchSel, g.churchRosterStart = 7, 2
	}
}

func TestModernHDFacilityRosterViewportAndMissingFrame(t *testing.T) {
	set := minimalFacilityRosterSet(t)
	g := facilityDialogueTestGame(t, set)
	facilityRosterTestEightParty(g)
	for _, state := range []struct{ owner, mode string }{{"shop", "recipient_equipment"}, {"shop", "sell_roster"}, {"church", "status_roster"}, {"church", "revive"}} {
		facilityRosterTestState(g, state.owner, state.mode, 1, 2)
		facilityRosterTestScroll(g, state.owner, state.mode)
		frame, ok := facilityRosterCompose(g)
		if !ok {
			t.Fatal("actual scrolled compositor unavailable", state)
		}
		plan := g.prepareModernHDFacilityRoster(frame)
		if plan == nil {
			t.Fatal("actual scrolled source refused", state)
		}
		start, count := 2, 6
		if state.mode == "recipient_equipment" {
			start, count = 3, 3
		} else if state.mode == "revive" {
			start, count = 5, 3
		}
		if plan.source.start != start || len(plan.units) != count || plan.source.ids[0] != start {
			t.Fatal("raw viewport start or visible party changed", state, plan.source)
		}
		unit := plan.units[0]
		original := set.mapSpritesHD[unit.group][unit.frame]
		set.mapSpritesHD[unit.group][unit.frame] = image.NewNRGBA(image.Rect(0, 0, 143, 144))
		if g.prepareModernHDFacilityRoster(frame) != nil {
			t.Fatal("wrong HD frame shape admitted")
		}
		set.mapSpritesHD[unit.group][unit.frame] = original
	}
}

func TestModernHDFacilityRosterRuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_HD_FACILITY_ROSTER_GPU") != "1" {
		t.Skip("dedicated GPU process")
	}
	set := facilityHDTestSet(t)
	g := facilityDialogueTestGame(t, set)
	facilityRosterTestEightParty(g)
	out := os.Getenv("FD2_TEST_FACILITY_OUT")
	if out == "" {
		t.Fatal("explicit output required")
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	p := &facilityHDProbe{t: t, g: g, out: out, presentation: &modernHDPresentation{Game: g, toolbar: true},
		captured: make(map[*modernFacilityScene]bool), captureCases: make(map[int]bool)}
	p.verifyScene = func(p *facilityHDProbe, pixels []byte, offscreen *ebiten.Image) error {
		s, ok := g.modernFacilityRosterSource()
		if !ok {
			return fmt.Errorf("case%d actual roster source absent", p.index)
		}
		art := g.modernStoryPortraits != nil
		plan := g.facilityRosterHD
		if (plan != nil) != art || art && len(plan.units) != len(s.ids) {
			return fmt.Errorf("case%d wrong Q publication", p.index)
		}
		native, ok := facilityRosterCompose(g)
		if !ok {
			return fmt.Errorf("case%d actual native compositor absent", p.index)
		}
		checked := 0
		for row, id := range s.ids {
			origin := image.Pt(14+132*(row%s.cols), 117+26*(row/s.cols))
			group := g.partyRoster[id].MapSelectorKey
			for y := 0; y < 144; y++ {
				for x := 0; x < 144; x++ {
					value := native[(origin.Y+y/6)*320+origin.X+x/6]
					want := color.RGBAModel.Convert(g.nativeClassUI.palette[value]).(color.RGBA)
					tolerance := 0
					if art {
						base := plan.underlay.RGBAAt(origin.X+x/6, origin.Y+y/6)
						source := color.NRGBAModel.Convert(set.mapSpritesHD[group][s.cycle].At(x, y)).(color.NRGBA)
						blend := func(a, b byte) byte { return byte((int(a)*int(source.A) + int(b)*(255-int(source.A)) + 127) / 255) }
						want = color.RGBA{blend(source.R, base.R), blend(source.G, base.G), blend(source.B, base.B), 255}
						tolerance = 2
					}
					at := ((origin.Y*6+y+72)*1920 + origin.X*6 + x) * 4
					for channel, value := range []byte{want.R, want.G, want.B, 255} {
						delta := int(pixels[at+channel]) - int(value)
						if delta < -tolerance || delta > tolerance {
							return fmt.Errorf("case%d group%d frame%d pixel%d,%d channel%d: %d != %d", p.index, group, s.cycle, x, y, channel, pixels[at+channel], value)
						}
					}
					checked++
				}
			}
		}
		// Render the identical native underlay/text/scenery with just the Q layer
		// disabled. Every byte outside the admitted sprite rectangles must match.
		g.facilityRosterHD = nil
		reference := ebiten.NewImage(1920, 1272)
		defer reference.Dispose()
		var geometry ebiten.GeoM
		geometry.Scale(3, 3)
		p.presentation.DrawFinalScreen(reference, offscreen, geometry)
		control := make([]byte, len(pixels))
		reference.ReadPixels(control)
		g.facilityRosterHD = plan
		outside := append([]byte(nil), pixels...)
		if art {
			zero := make([]byte, 144*4)
			for _, unit := range plan.units {
				for y := 0; y < 144; y++ {
					at := ((unit.origin.Y*6+y+72)*1920 + unit.origin.X*6) * 4
					copy(outside[at:at+len(zero)], zero)
					copy(control[at:at+len(zero)], zero)
				}
			}
		}
		if !bytes.Equal(outside, control) {
			return fmt.Errorf("case%d Q layer changed pixels outside actual sprite rectangles", p.index)
		}
		p.checks = append(p.checks, map[string]any{"case": p.index, "owner": s.owner, "mode": s.mode,
			"variant": g.nativeShopVariant, "interface": g.interfaceMode, "art_hd": art,
			"source_start": s.start, "source_cycle": s.cycle, "source_ids": s.ids,
			"roster_pixels": checked, "hd_rgb_tolerance": 2, "native_rgb_tolerance": 0, "all_outside_bytes_equal": true})
		return nil
	}
	add := func(owner, mode string, variant, cycle, style int, scroll bool) {
		if style == 3 {
			p.captureCases[len(p.cases)] = true
		}
		p.cases = append(p.cases, func() {
			facilityRosterTestState(g, owner, mode, variant, cycle)
			if scroll {
				facilityRosterTestScroll(g, owner, mode)
			}
			g.modernStoryPortraits, g.interfaceMode = nil, playerInterfaceOriginal
			if style >= 2 {
				g.modernStoryPortraits = set
			}
			if style%2 == 1 {
				g.interfaceMode = playerInterfaceHD
			}
		})
	}
	for _, variant := range []int{1, 3, 5} {
		for _, mode := range facilityRosterShopModes {
			for style := 0; style < 4; style++ {
				add("shop", mode, variant, 0, style, false)
			}
		}
		for _, mode := range []string{"recipient_consumable", "recipient_equipment", "sell_roster"} {
			for cycle := 1; cycle < 3; cycle++ {
				for style := 0; style < 4; style++ {
					add("shop", mode, variant, cycle, style, false)
				}
			}
		}
	}
	for _, mode := range facilityRosterChurchModes {
		for style := 0; style < 4; style++ {
			add("church", mode, 0, 0, style, false)
		}
	}
	for _, state := range []struct{ owner, mode string }{{"shop", "recipient_equipment"}, {"shop", "sell_roster"}, {"church", "status_roster"}, {"church", "revive"}} {
		for style := 0; style < 4; style++ {
			add(state.owner, state.mode, 1, 2, style, true)
		}
	}
	ebiten.SetWindowSize(1920, 1272)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil {
		t.Fatal(p.err)
	}
	if len(p.checks) != 180 {
		t.Fatal("incomplete facility roster matrix", len(p.checks))
	}
	raw, _ := json.MarshalIndent(map[string]any{"status": "PASS-FACILITY-ROSTER-GPU",
		"scope": "11 stable modes/three shops/native map keys and phases/four independent styles/two-column and three-row scrolling; full sprite RGBA plus all bytes outside sprite rectangles; no new original oracle claim", "checks": p.checks}, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
