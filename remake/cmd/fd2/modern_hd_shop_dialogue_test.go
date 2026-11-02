package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

var shopDialogueModes = []struct {
	mode          string
	gold, choices bool
}{
	{"menu", true, false}, {"purchase", false, false}, {"confirm", true, true},
	{"insufficient", true, false}, {"recipient_consumable", false, false},
	{"recipient_equipment", false, false}, {"recipient_full", true, false},
	{"no_recipient", true, false}, {"equip_confirm", true, true},
	{"sell_roster", false, false}, {"sell_items", false, false},
	{"sell_empty", true, false}, {"sell_confirm", true, true}, {"equip_roster", false, false},
	{"transfer_intro", true, false}, {"transfer_empty", true, false},
	{"transfer_dest_prompt", true, false}, {"transfer_full", true, false},
	{"transfer_source", false, false}, {"transfer_dest", false, false}, {"transfer_items", false, false},
}

// These are rendering fixtures over the actual typed assets and compositors.
// They do not establish normal reachability of each feedback state.
func shopDialogueTestState(g *Game, variant int, mode string, choice, phase int) {
	g.camp = campaign.NewRunner(&campaign.Campaign{Start: "shop", Nodes: map[string]*campaign.Node{
		"shop": {Type: "shop", NativeHubVariant: variant, Goods: []campaign.Good{{ID: 0, Name: "item0", Price: 100}}},
	}})
	g.nativeClassUIJob, g.nativeChurchUIJob, g.nativeShopUIJob = nil, nil, nil
	g.nativeHotelMode, g.churchMode = "", ""
	g.nativeShopMode, g.nativeShopVariant, g.nativeShopUIPulse = mode, variant, phase
	g.nativeShopServiceSel, g.shopSel, g.nativeShopItemStart = 0, 0, 0
	g.churchSel, g.nativeClassUIPulse = 1-choice, 2-phase
	g.nativeShopConfirmSel, g.nativeShopEquipSel, g.nativeShopSellConfirmSel = 1-choice, 1-choice, 1-choice
	switch mode {
	case "confirm":
		g.nativeShopConfirmSel = choice
	case "equip_confirm":
		g.nativeShopEquipSel = choice
	case "sell_confirm":
		g.nativeShopSellConfirmSel = choice
	}
	g.shopItemTypes[0] = 0
	if mode == "recipient_consumable" {
		g.shopItemTypes[0] = 0x20
	}
	g.partyRoster[0] = setNativeTransferInventory(g.partyRoster[0], 0)
	g.nativeShopPendingUnit, g.nativeShopHasPendingUnit = g.partyRoster[0], true
	g.shopRecipients = append([]int(nil), g.partyJoinOrder...)
	g.shopRecipientSel, g.nativeShopRecipientStart, g.nativeShopRecipientCycle = 0, 0, 0
	g.shopSellUnitSel, g.shopSellSlotSel, g.nativeShopSellRosterTop = 0, 0, 0
	g.nativeShopSellRosterCycle, g.nativeShopSellItemTop = 0, 0
	g.nativeShopSellItemIDs = []int{0}
	g.nativeShopEquipUnitSel, g.nativeShopEquipRosterTop = 0, 0
	g.nativeShopTransferIDs = append([]int(nil), g.partyJoinOrder...)
	g.nativeShopTransferSource, g.nativeShopTransferDest = 0, 1
	g.nativeShopTransferItems = []int{0}
	g.nativeShopTransferSel, g.nativeShopTransferTop = 0, 0
}

func shopDialogueCompose(g *Game) ([]byte, bool) {
	switch g.nativeShopMode {
	case "menu":
		return g.composeNativeShopServiceMenu()
	case "purchase":
		return g.composeNativeShopPurchaseList()
	case "confirm":
		return g.composeNativeShopPurchaseConfirmation()
	case "insufficient":
		return g.composeNativeShopInsufficientGold()
	case "recipient_consumable", "recipient_equipment":
		return g.composeNativeShopRecipient()
	case "recipient_full":
		return g.composeNativeShopRecipientFull()
	case "no_recipient":
		return g.composeNativeShopNoEligibleRecipient()
	case "equip_confirm":
		return g.composeNativeShopEquipConfirmation()
	case "sell_roster":
		return g.composeNativeShopSellRoster()
	case "sell_items":
		return g.composeNativeShopSellItems()
	case "sell_empty":
		return g.composeNativeShopSellEmpty()
	case "sell_confirm":
		return g.composeNativeShopSellConfirmation()
	case "equip_roster":
		return g.composeNativeShopEquipRoster()
	case "transfer_intro", "transfer_empty", "transfer_dest_prompt", "transfer_full":
		return g.composeNativeShopTransferMessage()
	case "transfer_source", "transfer_dest":
		return g.composeNativeShopTransferRoster()
	case "transfer_items":
		return g.composeNativeShopTransferItems()
	}
	return nil, false
}

func TestModernHDShopDialogueSourceAndOwnership(t *testing.T) {
	g := facilityDialogueTestGame(t, nil)
	for _, variant := range []int{1, 3, 5} {
		for _, state := range shopDialogueModes {
			for choice := 0; choice < 2; choice++ {
				for _, phase := range []int{0, 2} {
					shopDialogueTestState(g, variant, state.mode, choice, phase)
					g.modernHDText = nil
					frame, ok := shopDialogueCompose(g)
					if !ok {
						t.Fatal("actual compositor unavailable", variant, state.mode)
					}
					before := append([]byte(nil), frame...)
					g.modernHDText = &modernHDTextFrame{}
					g.appendModernHDFacilityDialogue(frame)
					if len(g.modernHDText.panels) != 1 || g.modernHDText.panels[0].gold != state.gold ||
						(len(g.modernHDText.choices) == 2) != state.choices {
						t.Fatal("wrong chrome", variant, state.mode, choice, phase)
					}
					if state.choices {
						for _, c := range g.modernHDText.choices {
							if (c.sourceIndex%3 == 1) != (c.option == choice && phase == 2) {
								t.Fatal("confirmation used foreign owner choice/pulse")
							}
						}
					}
					if !bytes.Equal(frame, before) {
						t.Fatal("native frame changed")
					}
					g.modernHDText = &modernHDTextFrame{}
					covered := append([]byte(nil), frame...)
					covered[112*320+5] ^= 1
					g.appendModernHDFacilityDialogue(covered)
					if len(g.modernHDText.panels) != 0 {
						t.Fatal("covered border admitted")
					}
					if state.choices {
						covered = append([]byte(nil), frame...)
						covered[168*320+232] ^= 1
						g.appendModernHDFacilityDialogue(covered)
						if len(g.modernHDText.panels) != 0 || len(g.modernHDText.choices) != 0 {
							t.Fatal("covered confirmation admitted")
						}
					}
					g.nativeShopUIJob = &nativeClassUIJob{}
					g.appendModernHDFacilityDialogue(frame)
					if len(g.modernHDText.panels) != 0 {
						t.Fatal("job admitted")
					}
					g.nativeShopUIJob = nil
					g.camp.Node().NativeHubVariant = 0
					g.appendModernHDFacilityDialogue(frame)
					if len(g.modernHDText.panels) != 0 {
						t.Fatal("foreign owner admitted")
					}
				}
			}
		}
	}
	for _, mode := range []string{"equip_panel", "success", "sell_success", "unknown"} {
		shopDialogueTestState(g, 1, mode, 0, 0)
		if _, _, ok := g.facilityDialoguePolicy(); ok {
			t.Fatal("non-stable owner admitted", mode)
		}
	}
}

func TestModernHDShopEquipmentRecipientReplacesGreeting(t *testing.T) {
	g := facilityDialogueTestGame(t, nil)
	for _, variant := range []int{1, 3, 5} {
		for _, locale := range []string{"zh-Hant", "zh-Hans", "en", "ja"} {
			shopDialogueTestState(g, variant, "recipient_equipment", 0, 0)
			var err error
			g.localeCatalog, err = loadOfficialLocale(locale)
			if err != nil {
				t.Fatal(err)
			}
			g.localeContent, err = loadOfficialLocaleContent(locale)
			if err != nil {
				t.Fatal(err)
			}
			g.localeEntities, err = loadOfficialLocaleEntities(locale)
			if err != nil {
				t.Fatal(err)
			}
			g.localeID, g.interfaceMode = locale, playerInterfaceHD
			g.modernHDText = &modernHDTextFrame{}
			g.font.indexedHD = &modernHDIndexedText{palette: g.nativeClassUI.palette}
			restoreDigits := g.beginModernHDShopDigits()
			_, ok := g.composeNativeShopEquipmentRecipient()
			restoreDigits()
			if !ok {
				t.Fatal("actual equipment recipient compositor unavailable")
			}
			names := make(map[string]bool)
			for _, id := range g.shopRecipients[:3] {
				name, err := g.localeEntities.CharacterName(g.partyRoster[id].NativeIdentity)
				if err != nil {
					t.Fatal(err)
				}
				names[name] = false
			}
			amount := false
			for _, run := range g.font.indexedHD.runs {
				if run.y < 220 {
					amount = true
					continue
				}
				if _, current := names[run.text]; current {
					names[run.text] = true
				} else if len(run.text) != 1 || run.text[0] < '0' || run.text[0] > '9' {
					t.Fatalf("variant%d/%s covered greeting remains in HD text: %q", variant, locale, run.text)
				}
			}
			for name, found := range names {
				if !found {
					t.Fatal("current visible name missing", name)
				}
			}
			if !amount {
				t.Fatal("clearing covered greeting removed the amount")
			}
			g.font.indexedHD = nil
		}
	}
}

func TestModernHDShopDialogueRuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_HD_SHOP_DIALOGUE_GPU") != "1" {
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
		var gold, choices, known bool
		for _, state := range shopDialogueModes {
			if state.mode == g.nativeShopMode {
				gold, choices, known = state.gold, state.choices, true
			}
		}
		if !known {
			return fmt.Errorf("unknown fixture mode %s", g.nativeShopMode)
		}
		hd := g.interfaceMode == playerInterfaceHD
		if hd && g.modernHDText != nil && g.nativeShopMode == "recipient_equipment" {
			names := make(map[string]bool)
			for _, id := range g.shopRecipients[:3] {
				name, err := g.localeEntities.CharacterName(g.partyRoster[id].NativeIdentity)
				if err != nil {
					return err
				}
				names[name] = true
			}
			for _, run := range g.modernHDText.runs {
				if run.y >= 220 && !names[run.text] && (len(run.text) != 1 || run.text[0] < '0' || run.text[0] > '9') {
					return fmt.Errorf("case%d covered greeting remains: %q", p.index, run.text)
				}
			}
		}
		panels, confirmations := 0, 0
		if g.modernHDText != nil {
			panels, confirmations = len(g.modernHDText.panels), len(g.modernHDText.choices)
		}
		if (panels == 1) != hd || (confirmations == 2) != (hd && choices) {
			return fmt.Errorf("case%d %s panel/choices=%d/%d", p.index, g.nativeShopMode, panels, confirmations)
		}
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
						return fmt.Errorf("case%d border pixel%d,%d", p.index, x, y)
					}
					checked++
				}
			}
			for _, c := range g.modernHDText.choices {
				if c.locale != g.localeID {
					return fmt.Errorf("case%d stale confirmation locale", p.index)
				}
			}
		}
		base := make([]byte, 640*424*4)
		offscreen.ReadPixels(base)
		interior := 0
		for y := 115 * 6; y < 195*6; y += 3 {
			for x := 8 * 6; x < 312*6; x += 3 {
				skip := false
				if frame := g.modernHDText; frame != nil {
					for _, run := range frame.runs {
						skip = skip || image.Pt(x/3, y/3).In(run.clip)
					}
					for _, c := range frame.choices {
						skip = skip || image.Pt(x/6, y/6).In(c.rect)
					}
					for _, icon := range frame.serviceIcons {
						skip = skip || image.Pt(x, y).In(image.Rectangle{Min: icon.origin.Mul(6), Max: icon.origin.Add(image.Pt(24, 20)).Mul(6)})
					}
				}
				if plan := g.facilityRosterHD; plan != nil {
					for _, unit := range plan.units {
						skip = skip || image.Pt(x, y).In(image.Rectangle{Min: unit.origin.Mul(6), Max: unit.origin.Add(image.Pt(24, 24)).Mul(6)})
					}
				}
				if skip {
					continue
				}
				at, source := ((y+72)*1920+x)*4, ((y/3+24)*640+x/3)*4
				if !bytes.Equal(pixels[at:at+4], base[source:source+4]) {
					return fmt.Errorf("case%d chrome changed interior%d,%d", p.index, x, y)
				}
				interior++
			}
		}
		p.checks = append(p.checks, map[string]any{"case": p.index, "variant": g.nativeShopVariant, "mode": g.nativeShopMode,
			"locale": g.localeID, "interface": g.interfaceMode, "art_hd": g.modernStoryPortraits != nil,
			"gold": gold, "confirmations": confirmations, "border_pixels": checked, "interior_points": interior})
		return nil
	}
	add := func(variant int, mode, locale string, choice, phase, style int) {
		if style == 3 && locale == "zh-Hant" && choice == 0 && phase == 0 {
			p.captureCases[len(p.cases)] = true
		}
		p.cases = append(p.cases, func() {
			shopDialogueTestState(g, variant, mode, choice, phase)
			var err error
			g.localeCatalog, err = loadOfficialLocale(locale)
			if err != nil {
				p.err = err
				return
			}
			g.localeContent, err = loadOfficialLocaleContent(locale)
			if err != nil {
				p.err = err
				return
			}
			g.localeEntities, err = loadOfficialLocaleEntities(locale)
			if err != nil {
				p.err = err
				return
			}
			g.localeID, g.modernStoryPortraits, g.interfaceMode = locale, nil, playerInterfaceOriginal
			if style >= 2 {
				g.modernStoryPortraits = set
			}
			if style%2 == 1 {
				g.interfaceMode = playerInterfaceHD
			}
		})
	}
	for _, variant := range []int{1, 3, 5} {
		for _, state := range shopDialogueModes {
			for style := 0; style < 4; style++ {
				add(variant, state.mode, "zh-Hant", 0, 0, style)
			}
		}
	}
	for _, mode := range []string{"confirm", "equip_confirm", "sell_confirm"} {
		for _, locale := range []string{"zh-Hant", "zh-Hans", "en", "ja"} {
			for choice := 0; choice < 2; choice++ {
				for _, phase := range []int{0, 2} {
					for _, style := range []int{1, 3} {
						add(1, mode, locale, choice, phase, style)
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
	if len(p.checks) != 348 {
		t.Fatal("incomplete shop mode/style/confirmation matrix", len(p.checks))
	}
	raw, _ := json.MarshalIndent(map[string]any{"status": "PASS-SHOP-DIALOGUE-GPU",
		"scope": "21 stable modes/three shops/four independent styles; three confirmation owners/four locales/two choices/two phases; actual Game.Draw; no new original oracle claim", "checks": p.checks}, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
