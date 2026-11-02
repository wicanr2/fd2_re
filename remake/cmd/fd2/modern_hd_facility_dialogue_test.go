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

var facilityDialogueHotelModes = []string{"menu", "saved", "loaded", "load_blocked", "slots", "loadslots"}
var facilityDialogueChurchModes = []string{"menu", "class_confirm", "revive_confirm", "revive_empty", "revive_insufficient", "transfer_full", "class", "status_roster", "transfer_source", "transfer_dest", "transfer_item", "revive"}

func facilityDialogueTestGame(t *testing.T, set *modernStoryPortraitSet) *Game {
	g := facilityHDTestGame(t, set)
	var err error
	g.nativeHotelUI, err = loadNativeHotelUIAssets(g.nativeClassUI)
	if err != nil {
		t.Fatal(err)
	}
	class, screen := newNativeChurchClassPathGame(t)
	screen.Dispose()
	unit := class.partyRoster[9]
	unit.NativeRecordRace, unit.HasNativeRecordRace = 1, true
	g.partyRoster[9] = unit
	g.classChangeTable = class.classChangeTable
	g.reviveFeeRates = []int{0, 7, 7, 7, 7, 7}
	return g
}

func facilityDialogueTestOwner(g *Game, owner, mode string, selection, pulse int) {
	g.camp = campaign.NewRunner(&campaign.Campaign{Start: "facility", Nodes: map[string]*campaign.Node{"facility": {Type: owner}}})
	g.nativeClassUIJob, g.nativeChurchUIJob, g.nativeShopUIJob = nil, nil, nil
	g.nativeShopMode, g.nativeHotelMode, g.churchMode = "", "", ""
	g.nativeChurchTextIndex = 586
	g.nativeClassUIPulse, g.nativeChurchUIPulse, g.nativeShopUIPulse = pulse, pulse, pulse
	g.churchSel, g.hotelSel = selection, selection
	g.churchIDs, g.churchClassID, g.churchReviveID = []int{9}, 9, 9
	g.churchReviveFee = 21
	g.churchTransferSource, g.churchTransferDest = 9, 9
	g.churchTransferItems = []int{0}
	g.churchRosterStart, g.churchVerticalStart, g.churchItemStart = 0, 0, 0
	if owner == "hotel" {
		g.nativeHotelMode = mode
	} else {
		g.churchMode = mode
	}
}

func facilityDialogueCompose(g *Game) ([]byte, bool) {
	if g.camp.Node().Type == "hotel" {
		return g.composeNativeHotelFrame(0)
	}
	switch g.churchMode {
	case "menu":
		return g.composeNativeChurchMenuFrame()
	case "class_confirm":
		return g.composeNativeClassConfirmationFrame()
	case "revive_confirm":
		return g.composeNativeChurchReviveConfirmation()
	case "revive_empty", "revive_insufficient":
		return g.composeNativeChurchReviveMessage()
	case "transfer_full":
		return g.composeNativeChurchTransferFullMessage()
	case "class":
		return g.composeNativeClassListFrame()
	case "status_roster", "transfer_source", "transfer_dest":
		return g.composeNativeChurchRosterFrame()
	case "transfer_item":
		return g.composeNativeChurchTransferItemFrame()
	case "revive":
		return g.composeNativeChurchReviveListFrame()
	}
	return nil, false
}

func TestModernHDFacilityDialogueSourceAndOwnership(t *testing.T) {
	g := facilityDialogueTestGame(t, nil)
	for _, owner := range []string{"hotel", "church"} {
		modes := facilityDialogueHotelModes
		if owner == "church" {
			modes = facilityDialogueChurchModes
		}
		for _, mode := range modes {
			facilityDialogueTestOwner(g, owner, mode, 0, 0)
			frame, ok := facilityDialogueCompose(g)
			if !ok {
				t.Fatal("original producer unavailable", owner, mode)
			}
			before := append([]byte(nil), frame...)
			g.modernHDText = &modernHDTextFrame{}
			g.appendModernHDFacilityDialogue(frame)
			if len(g.modernHDText.panels) != 1 {
				t.Fatal("source frame refused", owner, mode)
			}
			gold, choices, _ := g.facilityDialoguePolicy()
			if g.modernHDText.panels[0].gold != gold ||
				(len(g.modernHDText.choices) == 2) != choices {
				t.Fatal("wrong panel/choices", owner, mode)
			}
			h := g.facilityDialogueHD
			g.appendModernHDFacilityDialogue(frame)
			if g.facilityDialogueHD != h || !bytes.Equal(before, frame) {
				t.Fatal("cache/source changed")
			}
			g.modernHDText = &modernHDTextFrame{}
			bad := append([]byte(nil), frame...)
			bad[112*320+5] ^= 1
			g.appendModernHDFacilityDialogue(bad)
			if len(g.modernHDText.panels) != 0 {
				t.Fatal("covered border admitted")
			}
			if choices {
				bad = append([]byte(nil), frame...)
				bad[168*320+232] ^= 1
				g.appendModernHDFacilityDialogue(bad)
				if len(g.modernHDText.panels) != 0 || len(g.modernHDText.choices) != 0 {
					t.Fatal("covered choices admitted")
				}
			}
			g.nativeShopUIJob = &nativeClassUIJob{}
			g.appendModernHDFacilityDialogue(frame)
			if len(g.modernHDText.panels) != 0 {
				t.Fatal("foreign job admitted")
			}
			g.nativeShopUIJob = nil
			g.camp = campaign.NewRunner(&campaign.Campaign{Start: "shop", Nodes: map[string]*campaign.Node{"shop": {Type: "shop"}}})
			g.appendModernHDFacilityDialogue(frame)
			if len(g.modernHDText.panels) != 0 {
				t.Fatal("foreign owner admitted")
			}
			g.modernHDText = nil
			g.appendModernHDFacilityDialogue(frame)
		}
	}
	facilityDialogueTestOwner(g, "church", "menu", 0, 0)
	original := g.nativeClassUI
	copyUI := *original
	copyUI.panel.Pixels = append([]byte(nil), original.panel.Pixels...)
	copyUI.panel.Pixels[0] ^= 1
	g.nativeClassUI = &copyUI
	if _, err := g.prepareModernHDFacilityDialogue(); err == nil {
		t.Fatal("changed panel source admitted")
	}
	g.nativeClassUI = original
	h, err := g.prepareModernHDFacilityDialogue()
	if err != nil {
		t.Fatal(err)
	}
	copyUI = *original
	copyUI.palette = append(color.Palette(nil), original.palette...)
	copyUI.palette[74] = color.RGBA{10, 20, 30, 255}
	g.nativeClassUI = &copyUI
	newMaterial, err := g.prepareModernHDFacilityDialogue()
	if err != nil || h == newMaterial {
		t.Fatal("changed palette reused old material", err)
	}
}

func TestModernHDFacilityDialogueRuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_HD_FACILITY_DIALOGUE_GPU") != "1" {
		t.Skip("dedicated GPU process")
	}
	set := facilityHDTestSet(t)
	g := facilityDialogueTestGame(t, set)
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
		gold, choices, known := g.facilityDialoguePolicy()
		if !known {
			return fmt.Errorf("case%d owner absent", p.index)
		}
		hd := g.interfaceMode == playerInterfaceHD
		panels, confirmations := 0, 0
		if g.modernHDText != nil {
			panels, confirmations = len(g.modernHDText.panels), len(g.modernHDText.choices)
		}
		if (panels == 1) != hd || (confirmations == 2) != (hd && choices) {
			return fmt.Errorf("case%d wrong panel/confirmation count %d/%d %s/%s", p.index, panels, confirmations, g.camp.Node().Type, g.churchMode)
		}
		// The UI material is fixed approved geometry. Compare every border
		// pixel against its CPU material, including the cleared native band.
		checked := 0
		if hd {
			h := g.facilityDialogueHD
			patch := h.chrome.patchFor(1860, 516)
			if gold {
				patch = h.chrome.goldPatchFor(1860, 516)
			}
			for y := 0; y < 516; y++ {
				for x := 0; x < 1860; x++ {
					if x >= 18 && x < 1842 && y >= 18 && y < 498 {
						continue
					}
					want := patch.RGBAAt(x, y)
					at := ((112*6+y+72)*1920 + 5*6 + x) * 4
					if !bytes.Equal(pixels[at:at+4], []byte{want.R, want.G, want.B, 255}) {
						return fmt.Errorf("case%d border pixel %d,%d", p.index, x, y)
					}
					checked++
				}
			}
			if choices {
				for _, c := range g.modernHDText.choices {
					if c.locale != g.localeID {
						return fmt.Errorf("case%d stale confirmation locale", p.index)
					}
				}
			}
		}
		// The chrome is transparent in the interior; retain all plain native
		// pixels when no HD text/confirmation covers that point.
		base := make([]byte, 640*424*4)
		offscreen.ReadPixels(base)
		interior := 0
		for y := 115 * 6; y < 195*6; y += 3 {
			for x := 8 * 6; x < 312*6; x += 3 {
				skip := false
				if g.modernHDText != nil {
					for _, run := range g.modernHDText.runs {
						if image.Pt(x/3, y/3).In(run.clip) {
							skip = true
							break
						}
					}
					for _, c := range g.modernHDText.choices {
						if image.Pt(x/6, y/6).In(c.rect) {
							skip = true
							break
						}
					}
				}
				if skip {
					continue
				}
				at := ((y+72)*1920 + x) * 4
				src := ((y/3+24)*640 + x/3) * 4
				if !bytes.Equal(pixels[at:at+4], base[src:src+4]) {
					return fmt.Errorf("case%d chrome changed interior %d,%d", p.index, x, y)
				}
				interior++
			}
		}
		p.checks = append(p.checks, map[string]any{"case": p.index, "owner": g.camp.Node().Type, "hotel_mode": g.nativeHotelMode, "church_mode": g.churchMode, "locale": g.localeID, "interface": g.interfaceMode, "art_hd": g.modernStoryPortraits != nil, "gold": gold, "confirmation": choices, "border_pixels": checked, "interior_points": interior})
		return nil
	}
	add := func(owner, mode, locale string, selection, pulse, style int) {
		if style == 3 {
			p.captureCases[len(p.cases)] = true
		}
		p.cases = append(p.cases, func() {
			facilityDialogueTestOwner(g, owner, mode, selection, pulse)
			catalog, err := loadOfficialLocale(locale)
			if err != nil {
				t.Fatal(err)
			}
			content, err := loadOfficialLocaleContent(locale)
			if err != nil {
				t.Fatal(err)
			}
			entities, err := loadOfficialLocaleEntities(locale)
			if err != nil {
				t.Fatal(err)
			}
			g.localeID, g.localeCatalog, g.localeContent, g.localeEntities = locale, catalog, content, entities
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
	for _, owner := range []string{"hotel", "church"} {
		modes := facilityDialogueHotelModes
		if owner == "church" {
			modes = facilityDialogueChurchModes
		}
		for _, mode := range modes {
			for style := 0; style < 4; style++ {
				add(owner, mode, "zh-Hant", 0, 0, style)
			}
		}
	}
	for _, mode := range []string{"class_confirm", "revive_confirm"} {
		for _, locale := range []string{"zh-Hant", "zh-Hans", "en", "ja"} {
			for selection := 0; selection < 2; selection++ {
				for pulse := 0; pulse < 2; pulse++ {
					for _, style := range []int{1, 3} {
						add("church", mode, locale, selection, pulse*2, style)
					}
				}
			}
		}
	}
	ebiten.SetWindowSize(1920, 1272)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil {
		t.Fatal(p.err)
	}
	if len(p.checks) != 136 {
		t.Fatal("incomplete matrix", len(p.checks))
	}
	raw, _ := json.MarshalIndent(map[string]any{"status": "PASS-FACILITY-DIALOGUE-GPU", "scope": "stable source-bound frames; 18 modes, four styles and two confirmation owners with four locales/two choices/two pulses; no new original oracle claim", "checks": p.checks}, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
