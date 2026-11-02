package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"testing"
)

func TestModernHDFacilityMoneySourceAndFallback(t *testing.T) {
	g := facilityDialogueTestGame(t, nil)
	facilityRosterTestEightParty(g)
	cases, pixels, holes, rejected := 0, 0, 0, 0
	for _, owner := range []string{"shop", "church"} {
		variants := []int{1, 3, 5}
		if owner == "church" {
			variants = []int{0}
		}
		for _, variant := range variants {
			modes := facilityDialogueChurchModes
			if owner == "shop" {
				modes = nil
				for _, m := range shopDialogueModes {
					modes = append(modes, m.mode)
				}
			}
			for _, mode := range modes {
				if owner == "shop" {
					shopDialogueTestState(g, variant, mode, 0, 0)
				} else {
					facilityDialogueTestOwner(g, owner, mode, 0, 0)
				}
				var raw []byte
				var ok bool
				if owner == "shop" {
					raw, ok = shopDialogueCompose(g)
				} else {
					raw, ok = facilityDialogueCompose(g)
				}
				if !ok {
					t.Fatal("source missing")
				}
				before, _ := json.Marshal([]any{g.partyRoster, g.shopPending, g.gold, g.nativeRNGState})
				original := append([]byte(nil), raw...)
				g.modernHDText = &modernHDTextFrame{runs: []modernHDTextRun{{text: "retained"}}}
				g.appendModernHDFacilityMoney(raw, g.nativeClassUI.palette)
				d := g.modernHDText.money
				if d == nil || d.owner != owner || d.variant != variant || len(g.modernHDText.runs) != 1 || g.modernHDText.runs[0].text != "retained" {
					t.Fatal("source not bound or runs changed", owner, variant, mode)
				}
				run := d.currencyRun()
				if d.foreground != 169 || run.color != d.palette[169] {
					t.Fatal("currency selected outline instead of original brightest ink")
				}
				if run.text != "$" || run.font != g.font || run.x != 18 || run.y != 198 || run.pixels != 20 || run.clip != image.Rect(18, 198, 30, 214) || !run.inkTop {
					t.Fatal("currency placement differs")
				}
				patch := d.picture()
				if patch.Bounds() != image.Rect(0, 0, 378, 90) {
					t.Fatal("HD widget shape differs")
				}
				for y := 0; y < 90; y++ {
					for x := 0; x < 378; x++ {
						inside := image.Pt(x/6+5, y/6+95).In(moneyPrototypeDigits)
						alpha := byte(255)
						if inside {
							alpha = 0
							holes++
						}
						if patch.Pix[patch.PixOffset(x, y)+3] != alpha {
							t.Fatal("digit window was painted")
						}
						pixels++
					}
				}
				after, _ := json.Marshal([]any{g.partyRoster, g.shopPending, g.gold, g.nativeRNGState})
				if !bytes.Equal(before, after) || !bytes.Equal(raw, original) {
					t.Fatal("presentation mutated state or frame")
				}
				if mode == "menu" {
					for y := 95; y < 110; y++ {
						for x := 5; x < 68; x++ {
							if image.Pt(x, y).In(moneyPrototypeDigits) {
								continue
							}
							raw[y*320+x] ^= 1
							g.modernHDText.money = d
							g.appendModernHDFacilityMoney(raw, g.nativeClassUI.palette)
							if g.modernHDText.money != nil {
								t.Fatal("corrupt fixed source admitted")
							}
							raw[y*320+x] ^= 1
							rejected++
						}
					}
					for _, fault := range []string{"mode", "owner", "font", "palette", "help", "transient", "black", "white", "cell", "variant"} {
						if fault == "variant" && owner != "shop" {
							continue
						}
						font, palette, oldVariant := g.font, g.nativeClassUI.palette, g.nativeShopVariant
						cell, _ := moneyPrototypeCell(g, owner)
						switch fault {
						case "mode":
							if owner == "shop" {
								g.nativeShopMode = "unknown"
							} else {
								g.churchMode = "unknown"
							}
						case "owner":
							g.camp.Node().Type = "hotel"
						case "font":
							g.font = nil
						case "palette":
							g.nativeClassUI.palette = append(color.Palette(nil), palette...)
							g.nativeClassUI.palette[73] = nil
						case "cell":
							cell.Pixels[0] ^= 1
						case "variant":
							g.nativeShopVariant = 0
						}
						g.helpVisible, g.transientUI = fault == "help", fault == "transient"
						g.nativeFullDACBlack, g.nativeFullDACWhite = fault == "black", fault == "white"
						g.modernHDText.money = d
						g.appendModernHDFacilityMoney(raw, g.nativeClassUI.palette)
						if g.modernHDText.money != nil {
							t.Fatal("invalid source retained money", owner, fault)
						}
						if fault == "cell" {
							cell.Pixels[0] ^= 1
						}
						g.camp.Node().Type = owner
						g.nativeShopMode, g.churchMode = "menu", "menu"
						g.font = font
						g.nativeClassUI.palette = palette
						g.nativeShopVariant = oldVariant
						rejected++
					}
					g.helpVisible, g.transientUI, g.nativeFullDACBlack, g.nativeFullDACWhite = false, false, false, false
				}
				cases++
			}
		}
	}
	t.Logf("money stable_cases=%d hd_pixels=%d transparent_digit_pixels=%d fault_guards=%d", cases, pixels, holes, rejected)
}
